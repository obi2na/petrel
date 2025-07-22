package notion

import (
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/pkg"
	"github.com/obi2na/petrel/internal/service/notion"
	"go.uber.org/zap"
	"net/http"
)

func RegisterNotionRoutes(r *gin.RouterGroup, oauthSvc utils.OAuthService[notion.NotionTokenResponse]) {
	notionHandler := NewNotionHandler(oauthSvc)
	r.GET("/auth", notionHandler.AuthRedirect)
	r.GET("/auth/callback", notionHandler.NotionAuthCallback)
}

type NotionHandler[T any] struct {
	OauthService utils.OAuthService[T]
	JWTManager   utils.JWTManager
}

func NewNotionHandler[T notion.NotionTokenResponse](oauthSvc utils.OAuthService[T]) *NotionHandler[T] {
	return &NotionHandler[T]{
		OauthService: oauthSvc,
		JWTManager:   utils.NewJWTProvider(),
	}
}

func (h *NotionHandler[T]) AuthRedirect(c *gin.Context) {
	ctx := c.Request.Context()
	state, err := h.JWTManager.GenerateStateJWT(config.C.Notion.StateSecret)
	if err != nil {
		logger.With(ctx).Error("Failed to generate JWT state", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid or expired state",
		})
		return
	}
	redirectUrl := h.OauthService.GetAuthURL(state)
	logger.With(ctx).Debug("Generated Notion OAuth URL", zap.String("url", redirectUrl))
	logger.With(ctx).Info("Redirecting to Notion OAuth")
	c.Redirect(http.StatusFound, redirectUrl)

}

func (h *NotionHandler[T]) NotionAuthCallback(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Query("code")
	state := c.Query("state")

	logger.With(ctx).Info("Notion OAuth callback", zap.String("code", code), zap.String("state", state))
	// Validate signed token
	if err := h.JWTManager.ValidateStateJWT(state, config.C.Notion.StateSecret); err != nil {
		logger.With(ctx).Warn("Invalid or expired state JWT", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid or expired state",
		})
	}
	logger.With(ctx).Debug("JWT state validation successful", zap.String("code", code))

	token, err := h.OauthService.ExchangeCodeForToken(ctx, utils.TokenRequestParams{
		Code:        code,
		RedirectURI: config.C.Notion.RedirectURI,
	})
	if err != nil {
		logger.With(ctx).Warn("Token exchange failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "token exchange failed",
		})
		return
	}

	//cast to Notion Token
	notionToken := any(token).(*notion.NotionTokenResponse)
	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"workspace_id":   notionToken.WorkspaceID,
		"workspace_name": notionToken.WorkspaceName,
	})

}
