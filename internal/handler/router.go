package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/internal/logger"
	"net/http"
)

func RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		ctx := c.Request.Context()
		logger.With(ctx).Info("Health check requested")
		c.JSON(http.StatusOK, gin.H{
			"status": "Petrel is healthy",
		})
	})
}
