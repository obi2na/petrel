package notion

import (
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/pkg"
	"go.uber.org/zap"
	"net/http"
)

func RegisterNotionRoutes(r *gin.RouterGroup) {
	r.GET("/auth", NotionAuthRedirect)
	r.GET("/auth/callback", NotionAuthCallback)
}

// redirect for authorization
func NotionAuthRedirect(c *gin.Context) {
	ctx := c.Request.Context()
	state, err := utils.GenerateStateJWT(config.C.Notion.StateSecret)
	if err != nil {
		logger.With(ctx).Error("Failed to generate JWT state", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid or expired state",
		})
		return
	}
	redirectUrl := GetAuthURL(state)
	logger.With(ctx).Debug("Generated Notion OAuth URL", zap.String("url", redirectUrl))
	logger.With(ctx).Info("Redirecting to Notion OAuth")
	c.Redirect(http.StatusFound, redirectUrl)
}

func NotionAuthCallback(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Query("code")
	state := c.Query("state")

	logger.With(ctx).Info("Notion OAuth callback", zap.String("code", code), zap.String("state", state))

	// Validate signed token
	if err := utils.ValidateStateJWT(state, config.C.Notion.StateSecret); err != nil {
		logger.With(ctx).Warn("Invalid or expired state JWT", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid or expired state",
		})
	}
	logger.With(ctx).Debug("JWT state validation successful", zap.String("code", code))

	token, err := ExchangeCodeForToken(code, &http.Client{})
	if err != nil {
		logger.With(ctx).Warn("Token exchange failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "token exchange failed",
		})
	}

	logger.With(ctx).Info("Successfully exchanged token")
	// Debug log token details (never exposed to user)
	logger.With(ctx).Debug("Notion token exchanged",
		zap.String("access_token", token.AccessToken),
		zap.String("workspace_id", token.WorkspaceID),
		zap.String("user_email", token.Owner.User.Person.Email),
		zap.String("user_id", token.Owner.User.ID),
	)

	c.JSON(http.StatusOK, gin.H{
		"status":         "Token Exchanged successfully",
		"workspace_name": token.WorkspaceName,
	})
}
