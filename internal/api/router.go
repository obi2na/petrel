package api

import (
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/internal/api/auth"
	"github.com/obi2na/petrel/internal/api/notion"
	"github.com/obi2na/petrel/internal/logger"
	"net/http"
)

const (
	HealthPath = "/health"
)

func RegisterRoutes(r *gin.Engine) {
	r.GET(HealthPath, appHealth)

	authGroup := r.Group("/auth")
	auth.RegisterAuthRoutes(authGroup)

	notionGroup := r.Group("/notion")
	notion.RegisterNotionRoutes(notionGroup)
}

func appHealth(c *gin.Context) {
	ctx := c.Request.Context()
	logger.With(ctx).Info("Health check requested")
	c.JSON(http.StatusOK, gin.H{
		"status": "Petrel is healthy",
	})
}
