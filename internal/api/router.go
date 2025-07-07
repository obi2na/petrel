package api

import (
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/internal/api/notion"
	"github.com/obi2na/petrel/internal/logger"
	"net/http"
)

const (
	HealthPath         = "/health"
	NotionAuthCallback = "auth/notion/callback"
	NotionAuthPath     = "/auth/notion"
)

func RegisterRoutes(r *gin.Engine) {
	r.GET(HealthPath, appHealth)
	r.GET(NotionAuthPath, notion.NotionAuthRedirect)
	r.GET(NotionAuthCallback, notion.NotionAuthCallback)
}

func appHealth(c *gin.Context) {
	ctx := c.Request.Context()
	logger.With(ctx).Info("Health check requested")
	c.JSON(http.StatusOK, gin.H{
		"status": "Petrel is healthy",
	})
}
