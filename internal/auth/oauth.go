package auth

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/obi2na/petrel/config"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authURL  = "https://api.notion.com/v1/oauth/authorize"
	tokenURL = "https://api.notion.com/v1/oauth/token"
)

func GetAuthURL(state string) string {
	v := url.Values{}
	v.Set("client_id", config.C.Notion.ClientID)
	v.Set("response_type", "code")
	v.Set("owner", "user")
	v.Set("redirect_uri", config.C.Notion.RedirectURI)
	v.Set("state", state)

	return fmt.Sprintf("%s?%s", authURL, v.Encode())
}

func GenerateStateJWT() (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(5 * time.Minute).Unix(), // expires in 5 minutes
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.C.Notion.StateSecret))
}

func ValidateStateJWT(stateToken string) error {
	token, err := jwt.Parse(stateToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v\n", t.Header["alg"])
		}
		return []byte(config.C.Notion.StateSecret), nil
	})

	if err != nil {
		return err
	}

	// check  that signature is valid
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return fmt.Errorf("invalid token claims")
	}

	// Expiration Check
	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < time.Now().Unix() {
			return fmt.Errorf("token expired at %s", time.Unix(int64(exp), 0))
		}
	} else {
		return fmt.Errorf("token expired at %s", time.Unix(int64(exp), 0))
	}

	return nil
}

type TokenResponse struct {
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

func ExchangeCodeForToken(code string) (*TokenResponse, error) {
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

	//setup client to fire the request
	client := &http.Client{}
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
