package authService

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/pkg/jwtutil"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type AuthService interface {
	SendMagicLink(ctx context.Context, email string) error
	CompleteMagicLink(ctx context.Context, code, state string) (string, error)
}

type Auth0Service struct {
	config.Auth0Config
	HTTPClient *http.Client
}

func NewAuthService(cfg config.Auth0Config, client *http.Client) *Auth0Service {
	return &Auth0Service{
		Auth0Config: cfg,
		HTTPClient:  client,
	}
}

func (s *Auth0Service) SendMagicLink(ctx context.Context, email string) error {

	//generate state JWT
	state, err := jwtutil.GenerateStateJWT(config.C.Auth0.StateSecret)
	if err != nil {
		logger.With(ctx).Error("Failed to generate state", zap.Error(err))
		return fmt.Errorf("internal error")
	}

	// setup payload that will be used in request body
	payload := buildMagicLinkPayload(s.ClientID, s.ClientSecret, s.Connection, email, state)
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		logger.With(ctx).Error("Failed to marshal json payload", zap.Error(err))
		return fmt.Errorf("internal error")
	}

	//setup request
	url := fmt.Sprintf("https://%s/passwordless/start", s.Domain)
	reqBody := bytes.NewBuffer(jsonPayload)
	httpReq, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		logger.With(ctx).Error("Failed to create request", zap.Error(err))
		return fmt.Errorf("internal error")
	}
	httpReq.Header.Set("Content-Type", "application/json")

	//fire request to auth0 using client
	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		logger.With(ctx).Error("Request to Auth0 failed", zap.Error(err))
		return fmt.Errorf("failed to reach auth0")
	}
	defer resp.Body.Close()

	// read response body. will be used to populate logs if error occurs
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.With(ctx).Error("Failed to read response from Auth0", zap.Error(err))
		return fmt.Errorf("failed to read response from auth0")
	}

	//check for non-200 status
	if resp.StatusCode != http.StatusOK {
		logger.With(ctx).Error("Auth0 returned non-200 status",
			zap.Int("status", resp.StatusCode),
			zap.ByteString("response", bodyBytes),
			zap.Error(err))
		return fmt.Errorf("auth0 err : %s", string(bodyBytes))
	}

	return nil
}

func (s *Auth0Service) CompleteMagicLink(ctx context.Context, code, state string) (string, error) {
	if code == "" || state == "" {
		logger.With(ctx).Error("Missing code or state in callback")
		return "", fmt.Errorf("Missing code or state in callback")
	}

	// 1. validate state
	if err := jwtutil.ValidateStateJWT(state, config.C.Auth0.StateSecret); err != nil {
		logger.With(ctx).Error("Invalid state token", zap.Error(err))
		return "", err
	}

	// 2. Exchange code for tokens via Auth0
	tokenResp, err := s.ExchangeCodeForToken(ctx, code)
	if err != nil {
		logger.With(ctx).Error("Failed to exchange code for token", zap.Error(err))
		return "", err
	}

	// 3. extract email from ID token
	email, err := jwtutil.ExtractEmailFromIDToken(tokenResp.IDToken)
	if err != nil {
		logger.With(ctx).Error("Failed to parse ID token", zap.Error(err))
		return "", err
	}

	logger.With(ctx).Info("email extracted from id_token",
		zap.String("email", email),
	)

	// TODO: 4. Lookup or create user in DB

	return email, nil
}

func (s *Auth0Service) ExchangeCodeForToken(ctx context.Context, code string) (*TokenResponse, error) {
	data := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     s.ClientID,
		"client_secret": s.ClientSecret,
		"code":          code,
		"redirect_uri":  s.RedirectURI,
	}
	jsonData, _ := json.Marshal(data)
	auth0Url := fmt.Sprintf("https://%s/oauth/token", s.Domain)

	req, _ := http.NewRequest("POST", auth0Url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		logger.With(ctx).Error("Request to Auth0 failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		logger.With(ctx).Error("auth0 token exchange failed", zap.Error(err))
		return nil, fmt.Errorf("auth0 token exchange failed: %s", body)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}
	return &tokenResp, nil
}

func buildMagicLinkPayload(clientID, clientSecret, connection, email, state string) map[string]interface{} {
	return map[string]interface{}{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"connection":    connection,
		"email":         email,
		"send":          "link",
		"authParams": map[string]interface{}{
			"state": state,
		},
	}
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
