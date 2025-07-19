package api

import (
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/api/auth"
	"github.com/obi2na/petrel/internal/api/notion"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/middleware"
	utils "github.com/obi2na/petrel/internal/pkg"
	"github.com/obi2na/petrel/internal/service/bootstrap"
	"net/http"
)

const (
	HealthPath = "/health"
)

func RegisterRoutes(r *gin.Engine, services *bootstrap.ServiceContainer) {
	r.GET(HealthPath, appHealth)

	authGroup := r.Group("/auth")
	auth.RegisterAuthRoutes(authGroup, services.AuthSvc)

	notionGroup := r.Group("/notion")
	notionGroup.Use(middleware.AuthMiddleware(config.C.Auth0.PetrelJWTSecret, services.UserSvc, utils.NewJWTProvider()))
	notion.RegisterNotionRoutes(notionGroup)
}

func appHealth(c *gin.Context) {
	ctx := c.Request.Context()
	logger.With(ctx).Info("Health check requested")
	c.JSON(http.StatusOK, gin.H{
		"status": "Petrel is healthy",
	})
}
