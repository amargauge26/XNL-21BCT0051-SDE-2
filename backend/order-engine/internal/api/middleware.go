package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware handles authentication for protected routes
func AuthMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// TODO: Implement proper token validation
		// This is a placeholder for actual JWT validation
		if !isValidToken(token) {
			logger.Warn("Invalid auth token", zap.String("token", token))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Add user info to context for downstream handlers
		// TODO: Extract actual user info from token
		c.Set("user_id", "test_user")
		c.Next()
	}
}

// LoggerMiddleware logs request details
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		if len(c.Errors) > 0 {
			// Log errors if any occurred during request handling
			logger.Error("Request failed",
				zap.String("path", path),
				zap.String("query", query),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", latency),
				zap.String("method", c.Request.Method),
				zap.String("ip", c.ClientIP()),
				zap.Strings("errors", c.Errors.Errors()),
			)
		} else {
			logger.Info("Request processed",
				zap.String("path", path),
				zap.String("query", query),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", latency),
				zap.String("method", c.Request.Method),
				zap.String("ip", c.ClientIP()),
			)
		}
	}
}

// RateLimiterMiddleware implements a simple rate limiter
func RateLimiterMiddleware() gin.HandlerFunc {
	// TODO: Implement proper rate limiting using a token bucket or similar algorithm
	return func(c *gin.Context) {
		c.Next()
	}
}

// isValidToken is a placeholder for proper token validation
func isValidToken(token string) bool {
	// TODO: Implement proper token validation
	return token != ""
} 