package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/logger"
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
