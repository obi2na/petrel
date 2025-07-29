package api

import (
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/internal/api/auth"
	"github.com/obi2na/petrel/internal/api/manuscript"
	"github.com/obi2na/petrel/internal/api/notion"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/middleware"
	"github.com/obi2na/petrel/internal/service/bootstrap"
	"net/http"
)

const (
	HealthPath = "/health"
)

func RegisterRoutes(r *gin.Engine, services *bootstrap.ServiceContainer) {
	r.GET(HealthPath, appHealth)

	// register auth routes
	authGroup := r.Group("/auth")
	auth.RegisterAuthRoutes(authGroup, services.AuthSvc)

	// register notion services
	notionOauthSvc := services.NotionOauthSvc
	notionSvc := services.NotionSvc
	notionGroup := r.Group("/notion")
	notionGroup.Use(middleware.AuthMiddleware(services.UserSvc))
	notion.RegisterNotionRoutes(notionGroup, notionOauthSvc, notionSvc)

	// register manuscript routes
	manuscriptGroup := r.Group("/manuscript")
	manuscriptGroup.Use(middleware.AuthMiddleware(services.UserSvc))
	manuscriptSvc := services.ManuscriptSvc
	manuscript.RegisterManuscriptRoutes(manuscriptGroup, manuscriptSvc)
}

func appHealth(c *gin.Context) {
	ctx := c.Request.Context()
	logger.With(ctx).Info("Health check requested")
	c.JSON(http.StatusOK, gin.H{
		"status": "Petrel is healthy",
	})
}
