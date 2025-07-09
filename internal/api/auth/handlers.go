package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/service/auth"
	"go.uber.org/zap"
	"net/http"
	"time"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func RegisterAuthRoutes(r *gin.RouterGroup) {
	//create auth service
	auth0 := authService.NewAuthService(config.C.Auth0, httpClient)
	authHandler := NewHandler(auth0)

	//register routes gotten from authHadler
	r.POST("/login", authHandler.StartLoginWithMagicLink)
	// Add more auth routes here later
}

type MagicLinkRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type Handler struct {
	Service authService.AuthService
}

func NewHandler(service authService.AuthService) *Handler {
	return &Handler{service}
}

func (h *Handler) StartLoginWithMagicLink(c *gin.Context) {
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

	logger.With(ctx).Info("email extracted from /login response body",
		zap.String("email", req.Email),
	)

	// initiates a magic-link flow using Auth0
	if err := h.Service.SendMagicLink(ctx, req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send magic link",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Magic link sent",
	})
}
