package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/logger"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

func RegisterAuthRoutes(r *gin.RouterGroup) {
	r.POST("/login", StartMagicLink)
	// Add more auth routes here later
}

type MagicLinkRequest struct {
	Email string `json:"email" binding:"required,email"`
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// StartMagicLink initiates a magic-link flow using Auth0
func StartMagicLink(c *gin.Context) {
	ctx := c.Request.Context()
	var req MagicLinkRequest
	// 	c.ShouldBindJson tells gin to
	// 	1.	Read the request body (usually JSON sent via POST or PUT).
	//	2.	Parse (unmarshal) the JSON into the req struct.
	//	3.	Return an error if the JSON is malformed or missing required fields.
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.With(ctx).Error("Failed to bind json", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid email",
		})
		return
	}

	logger.With(ctx).Info(req.Email)

	domain := config.C.Auth0.Domain
	clientID := config.C.Auth0.ClientID
	connection := config.C.Auth0.Connection
	clientSecret := config.C.Auth0.ClientSecret

	payload := getPayload(clientID, clientSecret, connection, req.Email)

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		logger.With(ctx).Error("Failed to marshal json payload", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal Error",
		})
		return
	}

	url := fmt.Sprintf("https://%s/passwordless/start", domain)
	reqBody := bytes.NewBuffer(jsonPayload)

	httpReq, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		logger.With(ctx).Error("Failed to create request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal Error",
		})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		logger.With(ctx).Error("Request to Auth0 failed", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Failed to connect to Auth0 Service",
		})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.With(ctx).Error("Failed to read response from Auth0", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from Auth0"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		logger.With(ctx).Error("Auth0 returned non-200 statud",
			zap.Int("status", resp.StatusCode),
			zap.ByteString("response", bodyBytes),
			zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Auth0 Service error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Magic link sent",
	})
}

func getPayload(clientID, clientSecret, connection, email string) map[string]interface{} {
	return map[string]interface{}{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"connection":    connection,
		"email":         email,
		"send":          "link",
	}
}
