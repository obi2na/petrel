package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/internal/auth"
	"github.com/obi2na/petrel/internal/logger"
	"go.uber.org/zap"
	"net/http"
)

const (
	HealthPath         = "/health"
	NotionAuthCallback = "auth/notion/callback"
	NotionAuthPath     = "/auth/notion"
)

func RegisterRoutes(r *gin.Engine) {
	r.GET(HealthPath, appHealth)
	r.GET(NotionAuthPath, notionAuthRedirect)
	r.GET(NotionAuthCallback, notionAuthCallback)
}

func appHealth(c *gin.Context) {
	ctx := c.Request.Context()
	logger.With(ctx).Info("Health check requested")
	c.JSON(http.StatusOK, gin.H{
		"status": "Petrel is healthy",
	})
}

// redirect for authorization
func notionAuthRedirect(c *gin.Context) {
	ctx := c.Request.Context()
	state, err := auth.GenerateStateJWT()
	if err != nil {
		logger.With(ctx).Error("Failed to generate JWT state", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid or expired state",
		})
		return
	}
	redirectUrl := auth.GetAuthURL(state)
	logger.With(ctx).Info("Redirecting to Notion OAuth")
	c.Redirect(http.StatusFound, redirectUrl)
}

func notionAuthCallback(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Query("code")
	state := c.Query("state")

	logger.With(ctx).Info("Notion OAuth callback", zap.String("code", code), zap.String("state", state))

	// TODO: validate state
	if err := auth.ValidateStateJWT(state); err != nil {
		logger.With(ctx).Warn("Invalid or expired state", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid or expired state",
		})
	}

	// TODO: exchange code for access token
	c.JSON(http.StatusOK, gin.H{
		"message": "OAuth Successful ",
		"code":    code,
	})
}
