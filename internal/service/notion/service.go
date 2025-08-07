package notion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jomei/notionapi"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/db/models"
	"github.com/obi2na/petrel/internal/logger"
	petrelmodels "github.com/obi2na/petrel/internal/models"
	"github.com/obi2na/petrel/internal/pkg"
	"github.com/yuin/goldmark/ast"
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

// -----  notion oauth service begins here
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
	logger.With(ctx).Info("starting token exchange")
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

	logger.With(ctx).Info("token exchange successful")
	return &token, nil
}

// -----  notion oauth service ends here

// ---- notion service starts here

type DatabaseService interface {
	SaveIntegration(ctx context.Context, userID uuid.UUID, token *NotionTokenResponse) (models.NotionIntegration, error)
	CreatePetrelDraftsRepo(ctx context.Context, accessToken string) (string, error)
	UserHasWorkspace(ctx context.Context, userID uuid.UUID, workspaceID string) (petrelmodels.NotionUserIntegration, bool)
	IsValidDraftPage(ctx context.Context, userID uuid.UUID, pageID string) (bool, error)
}

type NotionDatabaseService struct {
	DB           utils.DB
	DBPool       *pgxpool.Pool
	HttpClient   utils.HTTPClient
	NotionClient utils.NotionApiClient
}

func NewNotionDatabaseService(pool *pgxpool.Pool, httpClient utils.HTTPClient, notionClient utils.NotionApiClient) *NotionDatabaseService {
	return &NotionDatabaseService{
		DB:           models.New(pool),
		DBPool:       pool,
		HttpClient:   httpClient,
		NotionClient: notionClient,
	}
}

func (s *NotionDatabaseService) SaveIntegration(ctx context.Context, userID uuid.UUID, token *NotionTokenResponse) (models.NotionIntegration, error) {

	// Lookup all notion integrations for user
	integrations, err := s.DB.GetNotionIntegrationsForUser(ctx, pgtype.UUID{userID, true})
	if err != nil {
		logger.With(ctx).Error("GetNotionIntegrationsForUser query failed", zap.Error(err))
		return models.NotionIntegration{}, fmt.Errorf("failed to fetch existing integrations: %w", err)
	}

	// For each integration, lookup its corresponding notion_integration and match workspace_id
	for _, integration := range integrations {
		notionMeta, err := s.DB.GetNotionIntegrationByIntegrationID(ctx, integration.ID)
		if err == nil && notionMeta.WorkspaceID == token.WorkspaceID {
			// confirm access token works
			if s.isAccessTokenValid(ctx, integration.AccessToken) {
				logger.With(ctx).Info("notion already integrated and token still valid")
				return notionMeta, nil // Already integrated and token still valid, skip resaving
			}
			break // stop loop we found a match
		}
	}

	// Begin db transaction
	tx, err := s.DBPool.Begin(ctx)
	if err != nil {
		logger.With(ctx).Error("failed to begin transaction", zap.Error(err))
		return models.NotionIntegration{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.DB.WithTx(tx)

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

	// Create integration record
	_, dbErr := qtx.CreateIntegration(ctx, integrationParams)
	if dbErr != nil {
		logger.With(ctx).Error("CreateIntegration query failed", zap.Error(err))
		return models.NotionIntegration{}, fmt.Errorf("failed to create new integrations for user %s: %w", userID, err)
	}

	// Create Drafts Repo to get draftsPageID
	draftsPageID, err := s.CreatePetrelDraftsRepo(ctx, token.AccessToken)
	if err != nil {
		return models.NotionIntegration{}, err
	}

	// Save notion integration
	notionIntegration, err := qtx.CreateNotionIntegration(ctx, models.CreateNotionIntegrationParams{
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
		DraftsPageID:     pgtype.Text{String: draftsPageID, Valid: true},
	})
	if err != nil {
		logger.With(ctx).Error("CreateNotionIntegration failed", zap.Error(err))
		return models.NotionIntegration{}, fmt.Errorf("failed to save Notion metadata: %w", err)
	}

	//  Commit transaction
	if err := tx.Commit(ctx); err != nil {
		logger.With(ctx).Error("Transaction commit failed", zap.Error(err))
		return models.NotionIntegration{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	str := fmt.Sprintf("new notion integration added for user %s", userID.String())
	logger.With(ctx).Info(str)

	return notionIntegration, nil
}

func (s *NotionDatabaseService) isAccessTokenValid(ctx context.Context, accessToken string) bool {
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

func (s *NotionDatabaseService) CreatePetrelDraftsRepo(ctx context.Context, accessToken string) (string, error) {
	createReq := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:      notionapi.ParentTypeWorkspace,
			Workspace: true,
		},
		Properties: notionapi.Properties{
			"title": notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{
						Text: &notionapi.Text{
							Content: "📝 Petrel Drafts Repo",
						},
					},
				},
			},
		},
		Children: []notionapi.Block{
			// Intro message
			&notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeParagraph,
				},
				Paragraph: notionapi.Paragraph{
					RichText: []notionapi.RichText{
						{
							Text: &notionapi.Text{
								Content: "This is your Petrel-managed drafts repository. All unpublished AI content lives here.",
							},
						},
					},
				},
			},
			// Deletion warning
			&notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeParagraph,
				},
				Paragraph: notionapi.Paragraph{
					RichText: []notionapi.RichText{
						{
							Annotations: &notionapi.Annotations{
								Bold:  true,
								Color: notionapi.ColorRed,
							},
							Text: &notionapi.Text{
								Content: "⚠️ Warning: Do not delete this page. Petrel uses it to manage your AI-generated drafts.",
							},
						},
					},
				},
			},
		},
	}

	page, err := s.NotionClient.CreatePage(ctx, accessToken, createReq)
	if err != nil {
		logger.With(ctx).Error("failed to create Petrel Drafts Repo", zap.Error(err))
		return "", fmt.Errorf("failed to create Petrel Drafts Repo: %w", err)
	}

	logger.With(ctx).Info("Petrel Drafts Repo created", zap.String("page_id", page.ID.String()))
	return page.ID.String(), nil
}

func (s *NotionDatabaseService) GetAccessTokenByUserAndWorkspace(ctx context.Context, userID uuid.UUID, workspaceID string) (models.GetNotionIntegrationAndTokenByUserAndWorkspaceRow, error) {
	integration, err := s.DB.GetNotionIntegrationAndTokenByUserAndWorkspace(ctx,
		models.GetNotionIntegrationAndTokenByUserAndWorkspaceParams{
			UserID:      pgtype.UUID{Bytes: userID, Valid: true},
			WorkspaceID: workspaceID,
		})
	if err != nil {
		return models.GetNotionIntegrationAndTokenByUserAndWorkspaceRow{}, fmt.Errorf("failed to fetch integration: %w", err)
	}
	return integration, nil
}

func (s *NotionDatabaseService) UserHasWorkspace(ctx context.Context, userID uuid.UUID, workspaceID string) (petrelmodels.NotionUserIntegration, bool) {
	integration, err := s.GetAccessTokenByUserAndWorkspace(ctx, userID, workspaceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.With(ctx).Warn("User does not have access to this workspace", zap.String("workspace_id", workspaceID))
			return petrelmodels.NotionUserIntegration{}, false
		}
		logger.With(ctx).Error("DB error while checking workspace access", zap.Error(err))
		return petrelmodels.NotionUserIntegration{}, false
	}
	return petrelmodels.NotionUserIntegration{
		Token:        integration.AccessToken,
		DraftsRepoID: integration.DraftsPageID.String,
	}, true
}

func (s *NotionDatabaseService) IsValidDraftPage(ctx context.Context, userID uuid.UUID, pageID string) (bool, error) {
	validNotionDraftPageParams := models.IsValidNotionDraftPageParams{
		UserID:       userID,
		NotionPageID: pageID,
	}
	return s.DB.IsValidNotionDraftPage(ctx, validNotionDraftPageParams)
}

// notion service ends here

// Notion Integration Service Starts Here

type IntegrationService interface {
	CompleteOAuth(ctx context.Context, code, state string, userID uuid.UUID) (models.NotionIntegration, error)
	StartOauth(ctx context.Context) (string, error)
}

// ----------------------------------------
//
//	NotionIntegrationService
//	Responsible for coordinating the full
//	integration lifecycle:
//	- validate state
//	- exchange token
//	- save metadata
//	- return enriched response
//
// -----------------------------------------
type NotionIntegrationService[T any] struct {
	NotionDBSvc  DatabaseService
	JWTManager   utils.JWTManager
	OAuthService utils.OAuthService[T]
}

func NewIntegrationService[T NotionTokenResponse](oauthSvc utils.OAuthService[T], notionDBSvc DatabaseService, jwt utils.JWTManager) *NotionIntegrationService[T] {
	return &NotionIntegrationService[T]{
		OAuthService: oauthSvc,
		NotionDBSvc:  notionDBSvc,
		JWTManager:   jwt,
	}
}

func (s *NotionIntegrationService[T]) StartOauth(ctx context.Context) (string, error) {

	state, err := s.JWTManager.GenerateStateJWT(config.C.Notion.StateSecret)
	if err != nil {
		logger.With(ctx).Error("Failed to generate JWT state", zap.Error(err))

		return "", err
	}

	return s.OAuthService.GetAuthURL(state), nil
}

func (s *NotionIntegrationService[T]) CompleteOAuth(ctx context.Context, code, state string, userID uuid.UUID) (models.NotionIntegration, error) {
	// 1. Validate the state token
	logger.With(ctx).Debug("validating notion state jwt")
	if err := s.JWTManager.ValidateStateJWT(state, config.C.Notion.StateSecret); err != nil {
		logger.With(ctx).Warn("Token state validation failed", zap.Error(err))
		return models.NotionIntegration{}, fmt.Errorf("invalid state: %w", err)
	}
	logger.With(ctx).Debug("notion state jwt validation successful")

	// 2. Exchange code for token
	token, err := s.OAuthService.ExchangeCodeForToken(ctx, utils.TokenRequestParams{
		Code:        code,
		RedirectURI: config.C.Notion.RedirectURI,
	})
	if err != nil {
		logger.With(ctx).Warn("Token exchange failed", zap.Error(err))
		return models.NotionIntegration{}, fmt.Errorf("token exchange failed: %w", err)
	}

	//cast to Notion Tokens
	notionToken, ok := any(token).(*NotionTokenResponse)
	if !ok {
		logger.With(ctx).Error("token type mismatch")
		return models.NotionIntegration{}, errors.New("token type mismatch")
	}

	// 3. Save to DB
	return s.NotionDBSvc.SaveIntegration(ctx, userID, notionToken)
}

// Notion Integration Service ends here

// Notion Draft Service starts here

type DraftService interface {
	StageDraft(ctx context.Context, userID uuid.UUID, notionDestinations []petrelmodels.ValidatedDestination, doc ast.Node, source []byte) ([]petrelmodels.DraftResultEntry, error)
}

type NotionDraftService struct {
	NotionClient utils.NotionApiClient
	Mapper       MarkdownToNotionMapper
}

func NewNotionDraftService(notionClient utils.NotionApiClient, notionMapper *PetrelMarkdownToNotionMapper) *NotionDraftService {
	return &NotionDraftService{
		NotionClient: notionClient,
		Mapper:       notionMapper,
	}
}

func (s *NotionDraftService) StageDraft(ctx context.Context, userID uuid.UUID, notionDestinations []petrelmodels.ValidatedDestination,
	doc ast.Node, source []byte) ([]petrelmodels.DraftResultEntry, error) {
	var results []petrelmodels.DraftResultEntry

	// Map AST -> Notion blocks
	blockTree, err := s.Mapper.Map(ctx, doc, source)
	if err != nil {
		return nil, err
	}
	// Flatten and transform each BlockWithChildren -> []notionapi.Block
	blocks := flattenBlockTree(blockTree)

	// iterate through notion workspaces
	for _, dest := range notionDestinations {
		var page *notionapi.Page
		var err error

		if dest.Append {
			// TODO: append blocks to existing page
		} else {
			page, err = s.createNewDraftPage(ctx, dest.Token, dest.DraftsRepoID, blocks)
		}

		if err != nil {
			results = append(results, petrelmodels.DraftResultEntry{
				Platform:     "notion",
				WorkspaceID:  dest.Workspace,
				PageID:       "",
				Status:       "fail",
				ErrorMessage: err.Error(),
			})

			logger.With(ctx).Error("Error pushing to notion", zap.Error(err))
			return results, err
		}

		results = append(results, petrelmodels.DraftResultEntry{
			Platform:     "notion",
			WorkspaceID:  dest.Workspace,
			PageID:       page.ID.String(),
			URL:          page.URL,
			Status:       "draft",
			Action:       "created",
			LintWarnings: nil,
		})
	}

	return results, nil
}

func (s *NotionDraftService) createNewDraftPage(ctx context.Context, token, draftsRepoID string, children []notionapi.Block) (*notionapi.Page, error) {
	req := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:   notionapi.ParentTypePageID,
			PageID: notionapi.PageID(draftsRepoID),
		},
		Properties: notionapi.Properties{
			"title": notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{
						Text: &notionapi.Text{
							Content: "Draft from Petrel",
						},
					},
				},
			},
		},
		Children: children,
	}
	return s.NotionClient.CreatePage(ctx, token, req)
}

func flattenBlockTree(tree []*BlockWithChildren) []notionapi.Block {
	var blocks []notionapi.Block

	// walk tree and recursively flatten
	func(nodes []*BlockWithChildren) {
		for _, b := range nodes {
			if len(b.Children) > 0 {
				setChildren(b.Block, flattenBlockTree(b.Children))
			}
			blocks = append(blocks, b.Block)
		}
	}(tree)

	return blocks
}

func setChildren(block notionapi.Block, children []notionapi.Block) {
	switch b := block.(type) {
	case *notionapi.ToggleBlock:
		b.Toggle.Children = children
	case *notionapi.BulletedListItemBlock:
		b.BulletedListItem.Children = children
	case *notionapi.NumberedListItemBlock:
		b.NumberedListItem.Children = children
	case *notionapi.QuoteBlock:
		b.Quote.Children = children
	}
}

// Notion Draft Service ends here
