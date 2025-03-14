package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/XNL-21bct0051-SDE-2/order-engine/internal/auth"
)

// RequireRole middleware ensures the user has the required role
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
			c.Abort()
			return
		}

		userClaims, ok := claims.(*auth.Claims)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid claims type"})
			c.Abort()
			return
		}

		if !userClaims.HasRole(role) {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole middleware ensures the user has any of the required roles
func RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
			c.Abort()
			return
		}

		userClaims, ok := claims.(*auth.Claims)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid claims type"})
			c.Abort()
			return
		}

		if !userClaims.HasAnyRole(roles...) {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin middleware ensures the user is an admin
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(auth.RoleAdmin)
}

// RequireTrader middleware ensures the user is a trader
func RequireTrader() gin.HandlerFunc {
	return RequireRole(auth.RoleTrader)
}

// RequireAnalyst middleware ensures the user is an analyst
func RequireAnalyst() gin.HandlerFunc {
	return RequireRole(auth.RoleAnalyst)
}

// RequireTraderOrAdmin middleware ensures the user is either a trader or admin
func RequireTraderOrAdmin() gin.HandlerFunc {
	return RequireAnyRole(auth.RoleTrader, auth.RoleAdmin)
}

// RequireAnalystOrAdmin middleware ensures the user is either an analyst or admin
func RequireAnalystOrAdmin() gin.HandlerFunc {
	return RequireAnyRole(auth.RoleAnalyst, auth.RoleAdmin)
} 