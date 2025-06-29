package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/internal/auth"
	"github.com/obi2na/petrel/internal/logger"
	"net/http"
)

const (
	NotionAuthPath = "/auth/notion"
	HealthPath     = "/health"
)

func RegisterRoutes(r *gin.Engine) {
	r.GET(HealthPath, appHealth)
	r.GET(NotionAuthPath, notionAuthRedirect)
}

func appHealth(c *gin.Context) {
	ctx := c.Request.Context()
	logger.With(ctx).Info("Health check requested")
	c.JSON(http.StatusOK, gin.H{
		"status": "Petrel is healthy",
	})
}

func notionAuthRedirect(c *gin.Context) {
	ctx := c.Request.Context()
	state := "some-random-state" // TODO: generate securely
	redirectUrl := auth.GetAuthURL(state)
	logger.With(ctx).Info("Redirecting to Notion OAuth")
	c.Redirect(http.StatusFound, redirectUrl)
}
