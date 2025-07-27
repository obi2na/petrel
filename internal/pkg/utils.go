package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgraph-io/ristretto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jomei/notionapi"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Interface to allow for mocking of http.Client
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type JWTManager interface {
	GenerateStateJWT(stateSecret string) (string, error)
	ValidateStateJWT(stateToken, stateSecret string) error
	ExtractUserInfoFromIDToken(idToken string) (UserInfo, error)
	GeneratePetrelJWT(userID, email, secret string) (string, error)
	ParseTokenAndExtractSub(tokenString, secret string) (string, error)
}

type JWTProvider struct{}

func NewJWTProvider() *JWTProvider {
	return &JWTProvider{}
}

// GenerateStateJWT creates a signed JWT token to be used as the `state` parameter
// in the OAuth authorization URL. The token includes an expiration claim
// and is signed with the application's Notion state secret
func (j *JWTProvider) GenerateStateJWT(stateSecret string) (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(5 * time.Minute).Unix(), // expires in 5 minutes
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(stateSecret))
}

// ValidateStateJWT parses and validates the provided state JWT token.
// It checks that the token is signed with the correct secret,
// is structurally valid, and has not expired
func (j *JWTProvider) ValidateStateJWT(stateToken, stateSecret string) error {
	token, err := jwt.Parse(stateToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v\n", t.Method.Alg())
		}
		return []byte(stateSecret), nil
	})

	if err != nil {
		return err
	}

	// type assertion that claims is of jwt.MapClaims
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

// TODO: Verify ID token signature and claims using Auth0's JWKS
// 1. Fetch JWKS from https://<your-auth0-domain>/.well-known/jwks.json
// 2. Match the `kid` in token header to key in JWKS
// 3. Use key to verify token signature and standard claims (exp, aud, iss)
func (j *JWTProvider) ExtractUserInfoFromIDToken(idToken string) (UserInfo, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return UserInfo{}, fmt.Errorf("parsing id token failed: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		email, _ := claims["email"].(string)
		name, _ := claims["name"].(string)
		avatar, _ := claims["picture"].(string)

		if email == "" || name == "" {
			return UserInfo{}, fmt.Errorf("email or name missing in ID token")
		}

		return UserInfo{
			Email:     email,
			Name:      name,
			AvatarURL: avatar,
		}, nil
	}

	return UserInfo{}, fmt.Errorf("invalid token")
}

func (j *JWTProvider) GeneratePetrelJWT(userID, email, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID, // principal user the token is about
		"iss":   "petrel",
		"aud":   "petrel-client",                       // use later during
		"exp":   time.Now().Add(24 * time.Hour).Unix(), // expires in 24 hours
		"iat":   time.Now().Unix(),
		"email": email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (j *JWTProvider) ParseTokenAndExtractSub(tokenString, secret string) (string, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	// Manually check expiration
	if expRaw, ok := claims["exp"]; ok {
		switch exp := expRaw.(type) {
		case float64:
			if int64(exp) < time.Now().Unix() {
				return "", errors.New("token expired")
			}
		case json.Number: // If using json.Unmarshal
			expInt, err := exp.Int64()
			if err != nil || expInt < time.Now().Unix() {
				return "", errors.New("token expired")
			}
		default:
			return "", errors.New("invalid exp format")
		}
	} else {
		return "", errors.New("exp not found in token")
	}

	// Extract user ID
	sub, ok := claims["sub"].(string)
	if !ok {
		return "", errors.New("invalid subject")
	}

	return sub, nil
}

// GenerateStateJWT creates a signed JWT token to be used as the `state` parameter
// in the OAuth authorization URL. The token includes an expiration claim
// and is signed with the application's Notion state secret
func GenerateStateJWT(stateSecret string) (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(5 * time.Minute).Unix(), // expires in 5 minutes
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(stateSecret))
}

// ValidateStateJWT parses and validates the provided state JWT token.
// It checks that the token is signed with the correct secret,
// is structurally valid, and has not expired
func ValidateStateJWT(stateToken, stateSecret string) error {
	token, err := jwt.Parse(stateToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v\n", t.Method.Alg())
		}
		return []byte(stateSecret), nil
	})

	if err != nil {
		return err
	}

	// type assertion that claims is of jwt.MapClaims
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

type UserInfo struct {
	Email     string
	Name      string
	AvatarURL string
}

// ------ Cache Implementation ----------

type Cache interface {
	Set(key string, value interface{}, ttl int64) bool
	Get(key string) (interface{}, bool)
	Del(key string)
}

type RistrettoCache struct {
	store *ristretto.Cache
}

func NewRistrettoCache() (Cache, error) {
	store, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // for hight performance
		MaxCost:     1 << 30, // 1 GB
		BufferItems: 64,      //recommended default
	})
	if err != nil {
		return nil, err
	}

	return &RistrettoCache{
		store,
	}, nil
}

func (r *RistrettoCache) Set(key string, value interface{}, ttl int64) bool {
	return r.store.SetWithTTL(key, value, 1, time.Duration(ttl)*time.Second)
}

func (r *RistrettoCache) Get(key string) (interface{}, bool) {
	return r.store.Get(key)
}

func (r *RistrettoCache) Del(key string) {
	r.store.Del(key)
}

var (
	cache     Cache
	cacheErr  error
	cacheOnce sync.Once
)

func InitCache(env string) error {
	cacheOnce.Do(func() {
		switch env {
		case "local", "dev", "test":
			cache, cacheErr = NewRistrettoCache()
			return
		default:
			str := fmt.Sprintf("unsupported env: %s", env)
			cacheErr = errors.New(str)
			return
		}
	})
	return cacheErr
}

func GetCache() (Cache, error) {
	return cache, cacheErr
}

// ------ Cache Implementation ends----------

// ------ OAuth Implementation starts 	-----------

type TokenRequestParams struct {
	Code         string
	RedirectURI  string
	CodeVerifier string
}

type OAuthService[T any] interface {
	GetAuthURL(state string) string
	ExchangeCodeForToken(ctx context.Context, params TokenRequestParams) (*T, error)
}

// ------ OAuth Implementation ends		-----------

// ------ Notion Api Client starts -------------

type NotionApiClient interface {
	CreatePage(ctx context.Context, token string, req *notionapi.PageCreateRequest) (*notionapi.Page, error)
}

type JomeiClient struct{}

func NewJomeiClient() *JomeiClient {
	return &JomeiClient{}
}

func (j *JomeiClient) CreatePage(ctx context.Context, token string, req *notionapi.PageCreateRequest) (*notionapi.Page, error) {
	client := notionapi.NewClient(notionapi.Token(token))
	return client.Page.Create(ctx, req)
}

func BuildNotionDraftRepoUrl(pageID string) string {
	return "https://www.notion.so/" + strings.ReplaceAll(pageID, "-", "")
}

// ------ Notion Api Client starts -------------
