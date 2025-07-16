package utils

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/obi2na/petrel/config"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGenerateStateJWT(t *testing.T) {
	// Setup fake config
	config.C.Notion.StateSecret = "test-secret"

	tokenStr, err := GenerateStateJWT(config.C.Notion.StateSecret)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	// Parse token
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.C.Notion.StateSecret), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	// Assert that token is using the expected algorithm (HS256)
	_, ok := token.Method.(*jwt.SigningMethodHMAC)
	assert.True(t, ok, "Signing method should be SigningMethodHMAC")
	assert.Equal(t, jwt.SigningMethodHS256.Alg(), token.Method.Alg())

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok, "claims should be map claims")

	// Assert expected claims exist
	exp, hasExp := claims["exp"].(float64)
	_, hasIat := claims["iat"].(float64)
	assert.True(t, hasExp, "token should contain exp")
	assert.True(t, hasIat, "token should contain iat")

	// Check expiration is in the future
	now := time.Now().Unix()
	assert.Greater(t, int64(exp), now, "token should not be expired")
}

func TestValidateStateJWT(t *testing.T) {
	config.C.Notion.StateSecret = "test-secret"

	tests := []struct {
		name        string
		getToken    getTokenFunc
		errExpected bool
		expectedErr string
	}{
		{
			name:        "test valid state JWT",
			getToken:    getValidStateJWT,
			errExpected: false,
		},
		{
			name:        "test expired state JWT",
			getToken:    getExpiredStateJWT,
			errExpected: true,
			expectedErr: "token is expired",
		},
		{
			name:        "test invalid signature state JWT",
			getToken:    getInvalidSignatureStateJWT,
			errExpected: true,
			expectedErr: "signature is invalid",
		},
		{
			name:        "test malformed state JWT",
			getToken:    getMalformedTokenStateJWT,
			errExpected: true,
			expectedErr: "token is malformed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := tc.getToken()
			assert.NoError(t, err, "Token generation failed")

			t.Logf("Generated token: %s", token)

			err = ValidateStateJWT(token, config.C.Notion.StateSecret)
			if tc.errExpected {
				assert.Error(t, err, "Token should be invalid")
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				assert.NoError(t, err, "Token should be valid")
			}
		})
	}

}

type getTokenFunc func() (string, error)

func getValidStateJWT() (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(5 * time.Minute).Unix(), // expires in 5 minutes
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.C.Notion.StateSecret))
}

func getExpiredStateJWT() (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(-5 * time.Minute).Unix(), // expired 5 minutes ago
		"iat": time.Now().Add(-10 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.C.Notion.StateSecret))
}

func getInvalidSignatureStateJWT() (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(5 * time.Minute).Unix(), // expired 5 minutes ago
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("invalid-signature"))
}

func getMalformedTokenStateJWT() (string, error) {
	return "this.is.notajwt", nil
}
