package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/logger"
	userservice "github.com/obi2na/petrel/internal/service/user"
	"net/http"
	"strings"
	"time"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		const reqIdKey = "X-Request-Id"
		//Generate UUID if no request_id header
		reqID := c.GetHeader(reqIdKey)
		if reqID == "" {
			reqID = uuid.NewString()
		}

		//save context and set header
		ctx := logger.InjectRequestID(c.Request.Context(), reqID)
		// replace the current request context with new context that includes request ID
		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set(reqIdKey, reqID)

		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     config.C.CORS.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

func AuthMiddleware(userSvc userservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx := c.Request.Context()

		//get bearer token
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			logger.With(ctx).Error("missing or invalid token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// validate bearer token
		user, err := userSvc.GetUserByTokenOrID(ctx, tokenString)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Add to context
		c.Set("user_id", user.ID)
		c.Next()
	}
}
