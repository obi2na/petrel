package auth

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/pkg"
	"github.com/obi2na/petrel/internal/service/auth"
	"go.uber.org/zap"
	"net/http"
	"time"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func RegisterAuthRoutes(r *gin.RouterGroup, authSvc authService.AuthService) {

	//create auth handler
	authHandler := NewHandler(authSvc)

	//register routes gotten from authHadler
	r.GET("/login", authHandler.BeginLogin)
	r.GET("/callback", authHandler.Callback)
}

type MagicLinkRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type Handler struct {
	AuthService authService.AuthService
}

func NewHandler(authService authService.AuthService) *Handler {
	return &Handler{
		AuthService: authService,
	}
}

// TODO: complete this when ready to setup frontend
func (h *Handler) StartLoginWithMagicLink(c *gin.Context) {
	ctx := c.Request.Context()
	var reqBody MagicLinkRequest
	// 	c.ShouldBindJson tells gin to
	// 	1.	Read the request body (usually JSON sent via POST or PUT).
	//	2.	Parse (unmarshal) the JSON into the req struct.
	//	3.	Return an error if the JSON is malformed or missing required fields.
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		logger.With(ctx).Error("Failed to bind json", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid email",
		})
		return
	}

	logger.With(ctx).Info("email extracted from /login response body",
		zap.String("email", reqBody.Email),
	)

	// initiates a magic-link flow using Auth0
	if err := h.AuthService.SendMagicLink(ctx, reqBody.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send magic link",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Magic link sent",
	})
}

func (h *Handler) Callback(c *gin.Context) {
	ctx := c.Request.Context()

	code := c.Query("code")
	state := c.Query("state")

	loginResult, err := h.AuthService.CompleteMagicLink(ctx, code, state)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication Failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Token exchange successful",
		"email":        loginResult.Email,
		"bearer_token": loginResult.Token,
	})
}

// using this to confirm backend login flow works
func (h *Handler) BeginLogin(c *gin.Context) {
	state, err := utils.GenerateStateJWT(config.C.Auth0.StateSecret)
	if err != nil {
		logger.With(c.Request.Context()).Error("Failed to generate state token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	authURL := fmt.Sprintf(
		"https://%s/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid%%20email%%20profile&state=%s",
		config.C.Auth0.Domain, config.C.Auth0.ClientID, config.C.Auth0.RedirectURI, state,
	)

	c.Redirect(http.StatusFound, authURL)
}
