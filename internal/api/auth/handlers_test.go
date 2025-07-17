package auth

import (
	"bytes"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) SendMagicLink(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

func (m *MockAuthService) CompleteMagicLink(ctx context.Context, code, state string) (authService.LoginResult, error) {
	args := m.Called(ctx, code, state)
	return args.Get(0).(authService.LoginResult), args.Error(1)
}

func TestStartLoginWithMagicLink(t *testing.T) {

	// initialize logger so code doesn't panic.
	logger.Init()

	//Arrange: create mock auth service
	mockAuthService := new(MockAuthService)

	// create handler
	h := NewHandler(mockAuthService)

	// create router
	router := gin.Default()
	router.POST("/login", h.StartLoginWithMagicLink) // configure route

	tests := []struct {
		name             string
		errExpected      bool
		failMockService  bool
		expectedErr      string
		reqBody          string
		expectedRespCode int
	}{
		{
			name:             "test valid request body",
			reqBody:          `{"email": "user@example.com"}`,
			errExpected:      false,
			expectedRespCode: http.StatusOK,
		},
		{
			name:             "test email missing from request body",
			reqBody:          `{}`,
			errExpected:      true,
			expectedErr:      "Invalid email",
			expectedRespCode: http.StatusBadRequest,
		},
		{
			name:             "test invalid email in request body",
			reqBody:          `{"email": "userexample.com"}`,
			errExpected:      true,
			expectedErr:      "Invalid email",
			expectedRespCode: http.StatusBadRequest,
		},
		{
			name:             "test SendMagicLink failed",
			reqBody:          `{"email": "user@example.com"}`,
			failMockService:  true,
			errExpected:      true,
			expectedErr:      "Failed to send magic link",
			expectedRespCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			//reset mock expectations per test case
			mockAuthService.ExpectedCalls = nil
			mockAuthService.Calls = nil

			if tc.failMockService {
				mockAuthService.On("SendMagicLink", mock.Anything, "user@example.com").Return(errors.New("SendMagicLink failed"))
			} else {
				mockAuthService.On("SendMagicLink", mock.Anything, "user@example.com").Return(nil)
			}

			//setup request
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tc.reqBody))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedRespCode, resp.Code)

			if tc.errExpected {
				assert.Contains(t, resp.Body.String(), tc.expectedErr)
				return
			}
			assert.Contains(t, resp.Body.String(), "Magic link sent")
			mockAuthService.AssertExpectations(t)

		})
	}

}

func TestCallback(t *testing.T) {
	// initialize logger so code doesn't panic.
	logger.Init()

	//Arrange: create mock auth service
	mockAuthService := new(MockAuthService)

	// create handler
	h := NewHandler(mockAuthService)

	// create router
	router := gin.Default()
	router.GET("/callback", h.Callback) // configure route

	tests := []struct {
		name             string
		errExpected      bool
		failMockService  bool
		expectedErr      string
		expectedRespCode int
	}{
		{
			name:             "test successful callback",
			errExpected:      false,
			failMockService:  false,
			expectedRespCode: http.StatusOK,
		},
		{
			name:             "test authentication failed callback",
			errExpected:      true,
			failMockService:  true,
			expectedRespCode: http.StatusUnauthorized,
			expectedErr:      "Authentication Failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			//reset mock expectations per test case
			mockAuthService.ExpectedCalls = nil
			mockAuthService.Calls = nil

			if tc.failMockService {
				mockAuthService.On("CompleteMagicLink", mock.Anything, "", "").Return(authService.LoginResult{}, errors.New("CompleteMagicLink failed"))
			} else {
				mockAuthService.On("CompleteMagicLink", mock.Anything, "", "").Return(authService.LoginResult{"user@example.com", "some-token"}, nil)
			}

			//setup request
			req := httptest.NewRequest(http.MethodGet, "/callback", nil)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedRespCode, resp.Code)

			if tc.errExpected {
				assert.Contains(t, resp.Body.String(), tc.expectedErr)
				return
			}
			assert.Contains(t, resp.Body.String(), "Token exchange successful")
			assert.Contains(t, resp.Body.String(), "user@example.com")
			mockAuthService.AssertExpectations(t)

		})
	}

}

func TestBeginLogin(t *testing.T) {
	// Initialize logger
	logger.Init()

	// Save and restore original config after test
	originalConfig := config.C
	defer func() { config.C = originalConfig }()

	// Set up test Auth0 config
	config.C.Auth0 = config.Auth0Config{
		Domain:      "test.auth0.com",
		ClientID:    "test-client-id",
		RedirectURI: "http://localhost:3000/auth/callback",
		StateSecret: "test-secret", // used in jwtutil
	}

	// Set up the handler with a dummy AuthService (not used in BeginLogin)
	h := NewHandler(nil)

	// Set up router
	router := gin.Default()
	router.GET("/login", h.BeginLogin)

	// Send request
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Check response
	assert.Equal(t, http.StatusFound, resp.Code)
	location := resp.Header().Get("Location")
	assert.Contains(t, location, "https://test.auth0.com/authorize?")
	assert.Contains(t, location, "client_id=test-client-id")
	assert.Contains(t, location, "redirect_uri=http://localhost:3000/auth/callback")
	assert.Contains(t, location, "state=") // JWT state param included
}
