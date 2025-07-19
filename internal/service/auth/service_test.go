package authService

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/db/models"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"net/http"
	"os"
	"testing"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetOrCreateUser(ctx context.Context, email, name, avatarURL string) (*models.User, error) {
	args := m.Called(ctx, email, name, avatarURL)
	return args.Get(0).(*models.User), args.Error(1)
}

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

type MockJWTManager struct {
	mock.Mock
}

func (m *MockJWTManager) GenerateStateJWT(secret string) (string, error) {
	args := m.Called(secret)
	return args.String(0), args.Error(1)
}

func (m *MockJWTManager) ValidateStateJWT(state, secret string) error {
	args := m.Called(state, secret)
	return args.Error(0)
}

func (m *MockJWTManager) ExtractUserInfoFromIDToken(token string) (utils.UserInfo, error) {
	args := m.Called(token)
	return args.Get(0).(utils.UserInfo), args.Error(1)
}

func (m *MockJWTManager) GeneratePetrelJWT(userID, email, secret string) (string, error) {
	args := m.Called(userID, email, secret)
	return args.String(0), args.Error(1)
}

func (m *MockJWTManager) ParseTokenAndExtractSub(tokenString string, secret string) (string, error) {
	args := m.Called(tokenString, secret)
	return args.String(0), args.Error(1)
}

// --- TESTs ----

func TestSendMagicLink(t *testing.T) {
	t.Setenv("CONFIG_DIR", "./testdata")
	if err := os.Setenv("APP_ENV", "test"); err != nil {
		t.Fatalf("Setenv Failed with %v", err)
	}

	if _, err := config.InitConfig("test"); err != nil {
		t.Fatalf("InitConfig Failed with %v", err)
	}

	logger.Init()

	tests := []struct {
		name               string
		errExpected        bool
		expectedErr        string
		mockJWTReturnState string
		mockJWTReturnErr   error
		mockHTTPRespBody   string
		mockHttpStatusCode int
	}{
		{
			name:               "test SendMagicLink successful",
			errExpected:        false,
			mockJWTReturnState: "generated_state",
			mockJWTReturnErr:   nil,
			mockHTTPRespBody:   `{"msg":"Email sent"}`,
			mockHttpStatusCode: http.StatusOK,
		},
		{
			name:               "test SendMagicLink failure from JWT",
			errExpected:        true,
			mockJWTReturnState: "",
			mockJWTReturnErr:   errors.New("JWT generation failed"),
		},
		{
			name:               "test SendMagicLink failure from httpClient",
			errExpected:        true,
			mockJWTReturnState: "generated_state",
			mockJWTReturnErr:   nil,
			mockHTTPRespBody:   `{"msg":"internal server error"}`,
			mockHttpStatusCode: http.StatusInternalServerError,
		},
	}

	//create mocks
	mockHTTP := new(MockHTTPClient)
	mockJWT := new(MockJWTManager)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			// reset mocks
			mockJWT.Calls = nil
			mockJWT.ExpectedCalls = nil
			mockHTTP.Calls = nil
			mockHTTP.ExpectedCalls = nil

			// set mock expectations
			mockJWT.On("GenerateStateJWT", config.C.Auth0.StateSecret).Return(tc.mockJWTReturnState, tc.mockJWTReturnErr)
			mockHTTP.On("Do", mock.Anything).Return(&http.Response{
				StatusCode: tc.mockHttpStatusCode,
				Body:       io.NopCloser(bytes.NewReader([]byte(tc.mockHTTPRespBody))),
			}, nil)

			authService := Auth0Service{
				Auth0Config: config.C.Auth0,
				HTTPClient:  mockHTTP,
				JwtProvider: mockJWT,
			}

			err := authService.SendMagicLink(context.Background(), "test@example.com")

			mockJWT.AssertExpectations(t)
			if tc.errExpected {
				assert.Error(t, err, "")
				return
			}
			assert.NoError(t, err)
			mockHTTP.AssertExpectations(t)
		})
	}

}

func TestExchangeCodeForToken(t *testing.T) {
	t.Setenv("CONFIG_DIR", "./testdata")
	if err := os.Setenv("APP_ENV", "test"); err != nil {
		t.Fatalf("Setenv Failed with %v", err)
	}

	if _, err := config.InitConfig("test"); err != nil {
		t.Fatalf("InitConfig Failed with %v", err)
	}

	logger.Init()

	//create mocks
	mockHTTP := new(MockHTTPClient)

	tests := []struct {
		name           string
		errExpected    bool
		expectedErr    string
		httpRespBody   string
		httpRespStatus int
		httpErr        error
		expectedResult *TokenResponse
	}{
		{
			name:           "test ExchangeCodeForToken successful",
			errExpected:    false,
			httpRespBody:   `{"id_token":"test-id-token"}`,
			httpRespStatus: http.StatusOK,
			expectedResult: &TokenResponse{IDToken: "test-id-token"},
			httpErr:        nil,
		},
		{
			name:           "test ExchangeCodeForToken fails for HTTP error",
			errExpected:    true,
			expectedErr:    "network error",
			httpRespStatus: 0,
			httpErr:        errors.New("network error"),
		},
		{
			name:           "non-200 status from auth0",
			errExpected:    true,
			expectedErr:    "auth0 token exchange failed",
			httpRespStatus: http.StatusUnauthorized,
			httpRespBody:   `{"error":"invalid_grant"}`,
		},
		{
			name:           "invalid json in body",
			errExpected:    true,
			expectedErr:    "invalid character",
			httpRespStatus: http.StatusOK,
			httpRespBody:   `invalid-json`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHTTP.Calls = nil
			mockHTTP.ExpectedCalls = nil

			mockHTTP.On("Do", mock.Anything).Return(&http.Response{
				StatusCode: tc.httpRespStatus,
				Body:       io.NopCloser(bytes.NewReader([]byte(tc.httpRespBody))),
			}, tc.httpErr)

			authService := Auth0Service{
				HTTPClient:  mockHTTP,
				Auth0Config: config.C.Auth0,
			}

			tokenResp, err := authService.ExchangeCodeForToken(context.Background(), "code")

			if tc.errExpected {
				assert.Error(t, err)
				if tc.expectedErr != "" {
					assert.Contains(t, err.Error(), tc.expectedErr)
				}
				return
			}

			assert.NoError(t, err)
			mockHTTP.AssertExpectations(t)
			//compare expected result to result
			if diff := cmp.Diff(tokenResp, tc.expectedResult); diff != "" {
				t.Errorf("unexpected config diff (-want +got):\n%s", diff)
			}
		})
	}

}

func TestCompleteMagicLink(t *testing.T) {
	t.Setenv("CONFIG_DIR", "./testdata")
	if err := os.Setenv("APP_ENV", "test"); err != nil {
		t.Fatalf("Setenv Failed with %v", err)
	}

	if _, err := config.InitConfig("test"); err != nil {
		t.Fatalf("InitConfig Failed with %v", err)
	}

	logger.Init()

	mockJWT := new(MockJWTManager)
	mockHTTP := new(MockHTTPClient)
	mockUserService := new(MockUserService)

	// Shared mock response
	mockTokenResp := `{"id_token": "mock-id-token"}`
	mockUserInfo := utils.UserInfo{
		Email:     "user@example.com",
		Name:      "John Doe",
		AvatarURL: "https://example.com/avatar.png",
	}
	mockUser := &models.User{
		ID:    uuid.MustParse("f8a2f92f-1354-45b8-a1e3-0387b02d11b0"),
		Email: mockUserInfo.Email,
		Name:  mockUserInfo.Name,
	}

	tests := []struct {
		name            string
		code            string
		state           string
		mockJWTError    error
		exchangeErr     error
		userInfoErr     error
		userServiceErr  error
		jwtGenErr       error
		expectError     bool
		expectedErrText string
	}{
		{
			name:        "successful callback",
			code:        "valid-code",
			state:       "valid-state",
			expectError: false,
		},
		{
			name:            "missing code",
			code:            "",
			state:           "some-state",
			expectError:     true,
			expectedErrText: "Missing code or state",
		},
		{
			name:            "invalid state token",
			code:            "code",
			state:           "bad-state",
			mockJWTError:    fmt.Errorf("invalid token"),
			expectError:     true,
			expectedErrText: "invalid token",
		},
		{
			name:            "token exchange fails",
			code:            "code",
			state:           "state",
			exchangeErr:     fmt.Errorf("token exchange failed"),
			expectError:     true,
			expectedErrText: "token exchange failed",
		},
		{
			name:            "id token parse fails",
			code:            "code",
			state:           "state",
			userInfoErr:     fmt.Errorf("bad token"),
			expectError:     true,
			expectedErrText: "bad token",
		},
		{
			name:            "user creation fails",
			code:            "code",
			state:           "state",
			userServiceErr:  fmt.Errorf("db error"),
			expectError:     true,
			expectedErrText: "db error",
		},
		{
			name:            "jwt generation fails",
			code:            "code",
			state:           "state",
			jwtGenErr:       fmt.Errorf("signing error"),
			expectError:     true,
			expectedErrText: "signing error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset mocks
			mockJWT.ExpectedCalls = nil
			mockJWT.Calls = nil
			mockHTTP.ExpectedCalls = nil
			mockHTTP.Calls = nil
			mockUserService.ExpectedCalls = nil
			mockUserService.Calls = nil

			authService := Auth0Service{
				Auth0Config: config.C.Auth0,
				HTTPClient:  mockHTTP,
				JwtProvider: mockJWT,
				UserService: mockUserService,
			}

			// Short-circuit: missing code or state
			if tc.code == "" || tc.state == "" {
				_, err := authService.CompleteMagicLink(context.Background(), tc.code, tc.state)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrText)
				return
			}

			// Mock JWT validation
			mockJWT.On("ValidateStateJWT", tc.state, config.C.Auth0.StateSecret).Return(tc.mockJWTError)

			if tc.mockJWTError != nil {
				_, err := authService.CompleteMagicLink(context.Background(), tc.code, tc.state)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrText)
				return
			}

			// Mock HTTP token exchange
			mockHTTP.On("Do", mock.Anything).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockTokenResp)),
			}, tc.exchangeErr)

			if tc.exchangeErr != nil {
				_, err := authService.CompleteMagicLink(context.Background(), tc.code, tc.state)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrText)
				return
			}

			// Mock ExtractUserInfoFromIDToken
			mockJWT.On("ExtractUserInfoFromIDToken", "mock-id-token").Return(mockUserInfo, tc.userInfoErr)

			if tc.userInfoErr != nil {
				_, err := authService.CompleteMagicLink(context.Background(), tc.code, tc.state)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrText)
				return
			}

			// Mock GetOrCreateUser
			mockUserService.On("GetOrCreateUser", mock.Anything, mockUserInfo.Email, mockUserInfo.Name, mockUserInfo.AvatarURL).
				Return(mockUser, tc.userServiceErr)

			if tc.userServiceErr != nil {
				_, err := authService.CompleteMagicLink(context.Background(), tc.code, tc.state)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrText)
				return
			}

			// Mock GeneratePetrelJWT
			mockJWT.On("GeneratePetrelJWT", mockUser.ID.String(), mockUser.Email, config.C.Auth0.PetrelJWTSecret).
				Return("mock-petrel-jwt-token", tc.jwtGenErr)

			if tc.jwtGenErr != nil {
				_, err := authService.CompleteMagicLink(context.Background(), tc.code, tc.state)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrText)
				return
			}

			// SUCCESS
			loginResult, err := authService.CompleteMagicLink(context.Background(), tc.code, tc.state)
			assert.NoError(t, err)
			assert.Equal(t, mockUser.Email, loginResult.Email)
			assert.Equal(t, "mock-petrel-jwt-token", loginResult.Token)

			mockJWT.AssertExpectations(t)
			mockHTTP.AssertExpectations(t)
			mockUserService.AssertExpectations(t)
		})
	}
}
