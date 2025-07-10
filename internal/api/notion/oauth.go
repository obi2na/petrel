package notion

import (
	"encoding/json"
	"fmt"
	"github.com/obi2na/petrel/config"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	authURL  = "https://api.notion.com/v1/oauth/authorize"
	tokenURL = "https://api.notion.com/v1/oauth/token"
)

// GetAuthURL builds the Notion OAuth authorization URL.
// It accepts a `state` string, which is used to prevent CSRF attacks,
// and returns the full URL that the user should be redirected to for authorization.
func GetAuthURL(state string) string {
	v := url.Values{}
	v.Set("client_id", config.C.Notion.ClientID)
	v.Set("response_type", "code")
	v.Set("owner", "user")
	v.Set("redirect_uri", config.C.Notion.RedirectURI)
	v.Set("state", state)

	return fmt.Sprintf("%s?%s", authURL, v.Encode())
}

// TokenResponse represents the JSON structure returned by Notion
// when exchanging an authorization code for an access token
type TokenResponse struct {
	AccessToken   string `json:"access_token"`
	TokenType     string `json:"token_type"`
	BotID         string `json:"bot_id"`
	WorkspaceName string `json:"workspace_name"`
	WorkspaceIcon string `json:"workspace_icon"`
	WorkspaceID   string `json:"workspace_id"`
	Owner         Owner  `json:"owner"`
}

// Owner contains information about the Notion workspace owner.
type Owner struct {
	Type string `json:"type"`
	User User   `json:"user"`
}

// User represents a user in Notion, including metadata and contact details.
type User struct {
	Object    string `json:"object"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Type      string `json:"type"`
	Person    Person `json:"person"`
}

// Person holds personal information like email for a Notion user.
type Person struct {
	Email string `json:"email"`
}

// Interface to allow mocking of http.Client used for ExchangeCodeForToken
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ExchangeCodeForToken exchanges the authorization code for an access token
// by making a POST request to Notion's OAuth token endpoint.
// It returns a parsed TokenResponse or an error if the exchange fails
func ExchangeCodeForToken(code string, client HTTPClient) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", config.C.Notion.RedirectURI)

	//setup new request for token exchange request
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(config.C.Notion.ClientID, config.C.Notion.ClientSecret) // prove client identity when exchanging auth code

	//fire the request using client
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close() //close the body after done with it. Body is I/O stream. don't want memory leaks

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed %s: ", string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}

	return &token, nil
}
