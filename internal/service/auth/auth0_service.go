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
