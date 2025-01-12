package middleware

import (
	"net/http"
	"strings"

	"comics/internal/tokenutil"

	"github.com/gin-gonic/gin"
)

// AuthenticationMiddleware checks if the user has a valid JWT
func AuthenticationMiddleware(accessTokenSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication token"})
			c.Abort()
			return
		}

		// The token should be prefixed with "Bearer "
		tokenParts := strings.Split(tokenString, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid structure token"})
			c.Abort()
			return
		}

		tokenString = tokenParts[1]

		claims, err := tokenutil.VerifyToken(tokenString, []byte(accessTokenSecret))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Subject)
		c.Next() // Proceed to the next handler if authorized
	}
}

// Role-based Middleware
func RoleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := c.Get("role")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing role"})
			c.Abort()
			return
		}

		if role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient privileges"})
			c.Abort()
			return
		}

		// Allow access
		c.Next()
	}
}
