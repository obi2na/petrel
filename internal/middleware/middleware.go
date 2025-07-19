package middleware

import (
	"encoding/json"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

func AuthMiddleware(secret string, userSvc userservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse and validate token
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Manually check expiration
		if expRaw, ok := claims["exp"]; ok {
			switch exp := expRaw.(type) {
			case float64:
				if int64(exp) < time.Now().Unix() {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
					return
				}
			case json.Number: // If using json.Unmarshal
				expInt, err := exp.Int64()
				if err != nil || expInt < time.Now().Unix() {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
					return
				}
			default:
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid exp format"})
				return
			}
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "exp not found in token"})
			return
		}

		// Extract user ID
		sub, ok := claims["sub"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid subject"})
			return
		}

		//TODO: use sub to validate user

		// Add to context
		c.Set("user_id", sub)
		c.Next()
	}
}
