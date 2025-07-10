package notion

import (
	"encoding/json"
	"github.com/obi2na/petrel/config"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetAuthURL(t *testing.T) {
	config.C.Notion.RedirectURI = "http://localhost:3000/callback"
	config.C.Notion.ClientID = "test-client-id"
	state := "some-state"
	redirectUrl := GetAuthURL(state)

	tests := []struct {
		name          string
		expectedParam string
	}{
		{
			name:          "test redirect url contains state",
			expectedParam: "state=some-state",
		},
		{
			name:          "test redirect url contains client-id",
			expectedParam: "client_id=test-client-id",
		},
		{
			name:          "test redirect url contains owner",
			expectedParam: "owner=user",
		},
		{
			name:          "test redirect url contains callback url",
			expectedParam: "redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fcallback",
		},
		{
			name:          "test redirect url contains code",
			expectedParam: "response_type=code",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Contains(t, redirectUrl, tc.expectedParam)
		})
	}

}

type MockHTTPClient struct {
	// use DoFunc as bridge to satisfy HTTPClient interface contract
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func getMockResponse() TokenResponse {
	return TokenResponse{
		AccessToken:   "access_token_123",
		TokenType:     "bearer",
		BotID:         "bot_123",
		WorkspaceName: "Petrel Workspace",
		WorkspaceIcon: "🐦",
		WorkspaceID:   "workspace_123",
		Owner: Owner{
			Type: "user",
			User: User{
				ID:   "user_123",
				Name: "Jane Doe",
				Person: Person{
					Email: "jane@example.com",
				},
			},
		},
	}
}

func getMockClient(body []byte) *MockHTTPClient {
	return &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(string(body))),
			}, nil
		},
	}
}

func TestExchangeCodeForToken(t *testing.T) {
	config.C.Notion.ClientID = "client-id"
	config.C.Notion.ClientSecret = "secret"
	config.C.Notion.RedirectURI = "http://localhost/callback"

	mockResp := getMockResponse()
	body, _ := json.Marshal(mockResp)

	mockClient := getMockClient(body)
	token, err := ExchangeCodeForToken("code123", mockClient)
	assert.NoError(t, err)
	assert.Equal(t, mockResp.AccessToken, token.AccessToken)
	assert.Equal(t, mockResp.WorkspaceID, token.WorkspaceID)
}
