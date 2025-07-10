package jwtutil

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

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

// TODO: Verify ID token signature and claims using Auth0's JWKS
// 1. Fetch JWKS from https://<your-auth0-domain>/.well-known/jwks.json
// 2. Match the `kid` in token header to key in JWKS
// 3. Use key to verify token signature and standard claims (exp, aud, iss)
func ExtractEmailFromIDToken(idToken string) (string, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("parsing id token failed: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if email, ok := claims["email"].(string); ok {
			return email, nil
		}
	}
	return "", fmt.Errorf("email not found in ID token")
}
