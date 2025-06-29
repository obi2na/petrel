package auth

import (
	"fmt"
	"github.com/obi2na/petrel/config"
	"net/url"
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
