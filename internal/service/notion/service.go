package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/db/models"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/pkg"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	authURL  = "https://api.notion.com/v1/oauth/authorize"
	tokenURL = "https://api.notion.com/v1/oauth/token"
)

type NotionTokenResponse struct {
	AccessToken   string `json:"access_token"`
	TokenType     string `json:"token_type"`
	BotID         string `json:"bot_id"`
	WorkspaceName string `json:"workspace_name"`
	WorkspaceIcon string `json:"workspace_icon"`
	WorkspaceID   string `json:"workspace_id"`
	Owner         Owner  `json:"owner"`
}

type Owner struct {
	Type string `json:"type"`
	User User   `json:"user"`
}

type User struct {
	Object    string `json:"object"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Type      string `json:"type"`
	Person    Person `json:"person"`
}

type Person struct {
	Email string `json:"email"`
}

type NotionOAuthService struct {
	httpClient utils.HTTPClient
}

func NewNotionOAuthService(client utils.HTTPClient) utils.OAuthService[NotionTokenResponse] {
	return &NotionOAuthService{httpClient: client}
}

func (n *NotionOAuthService) GetAuthURL(state string) string {
	v := url.Values{}
	v.Set("client_id", config.C.Notion.ClientID)
	v.Set("response_type", "code")
	v.Set("owner", "user")
	v.Set("redirect_uri", config.C.Notion.RedirectURI)
	v.Set("state", state)

	return fmt.Sprintf("%s?%s", authURL, v.Encode())
}

func (n *NotionOAuthService) ExchangeCodeForToken(ctx context.Context, params utils.TokenRequestParams) (*NotionTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", params.Code)
	data.Set("redirect_uri", params.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(config.C.Notion.ClientID, config.C.Notion.ClientSecret)

	res, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var token NotionTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

type Service interface {
	SaveIntegration(ctx context.Context, userID uuid.UUID, token *NotionTokenResponse) error
}

type NotionService struct {
	DB         models.Querier
	HttpClient utils.HTTPClient
}

func NewNotionService(db *pgxpool.Pool, client utils.HTTPClient) *NotionService {
	return &NotionService{
		DB:         models.New(db),
		HttpClient: client,
	}
}

func (s *NotionService) SaveIntegration(ctx context.Context, userID uuid.UUID, token *NotionTokenResponse) error {

	// many queries returns an empty slice
	integrations, err := s.DB.GetNotionIntegrationsForUser(ctx, pgtype.UUID{userID, true})
	if err != nil {
		logger.With(ctx).Error("GetNotionIntegrationsForUser query failed", zap.Error(err))
		return fmt.Errorf("failed to fetch existing integrations: %w", err)
	}

	// For each integration, lookup its corresponding notion_integration and match workspace_id
	for _, integration := range integrations {
		notionMeta, err := s.DB.GetNotionIntegrationByIntegrationID(ctx, integration.ID)
		if err == nil && notionMeta.WorkspaceID == token.WorkspaceID {
			// confirm access token works
			if s.isAccessTokenValid(ctx, integration.AccessToken) {
				logger.With(ctx).Info("notion already integrated and token still valid")
				return nil // Already integrated and token still valid, skip resaving
			}
			break // stop loop we found a match
		}
	}

	integrationID := uuid.New()

	integrationParams := models.CreateIntegrationParams{
		ID:           integrationID,
		UserID:       pgtype.UUID{userID, true},
		Service:      "notion",
		AccessToken:  token.AccessToken,
		RefreshToken: pgtype.Text{}, // Notion doesn't issue refresh tokens
		TokenType:    pgtype.Text{String: token.TokenType, Valid: true},
		ExpiresAt:    pgtype.Timestamptz{}, // not used
	}

	_, dbErr := s.DB.CreateIntegration(ctx, integrationParams)
	if dbErr != nil {
		logger.With(ctx).Error("CreateIntegration query failed", zap.Error(err))
		return fmt.Errorf("failed to create new integrations for user %s: %w", userID, err)
	}

	_, err = s.DB.CreateNotionIntegration(ctx, models.CreateNotionIntegrationParams{
		ID:               uuid.New(),
		IntegrationID:    integrationID,
		WorkspaceID:      token.WorkspaceID,
		WorkspaceName:    pgtype.Text{String: token.WorkspaceName, Valid: true},
		WorkspaceIcon:    pgtype.Text{String: token.WorkspaceIcon, Valid: true},
		BotID:            pgtype.Text{String: token.BotID, Valid: true},
		NotionUserID:     pgtype.Text{String: token.Owner.User.ID, Valid: true},
		NotionUserName:   pgtype.Text{String: token.Owner.User.Name, Valid: true},
		NotionUserAvatar: pgtype.Text{String: token.Owner.User.AvatarURL, Valid: true},
		NotionUserEmail:  pgtype.Text{String: token.Owner.User.Person.Email, Valid: true},
	})
	if err != nil {
		logger.With(ctx).Error("CreateNotionIntegration failed", zap.Error(err))
		return fmt.Errorf("failed to save Notion metadata: %w", err)
	}

	str := fmt.Sprintf("new notion integration added for user %s", userID.String())
	logger.With(ctx).Info(str)

	return nil
}

func (s *NotionService) isAccessTokenValid(ctx context.Context, accessToken string) bool {
	logger.With(ctx).Info("checking token health")
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.notion.com/v1/users/me", nil)
	if err != nil {
		logger.With(ctx).Error("isAccessTokenValid request failed failed", zap.Error(err))
		return false
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Notion-Version", "2022-06-28")

	res, err := s.HttpClient.Do(req)
	if err == nil && res.StatusCode == http.StatusOK {
		logger.With(ctx).Info("Token is still valid")
		return true // token is still valid, skip re-saving
	}

	return false
}
