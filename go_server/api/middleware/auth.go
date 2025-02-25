package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"comics/internal/tokenutil"

	"github.com/gin-gonic/gin"
)

const (
	// Headers
	KeyUserID        = "user_id"
	KeyRole          = "role"
	KeyAuthorization = "Authorization"
	KeyAccept        = "Accept"
	ContentTypeJSON  = "application/json"

	// Cookies
	KeyAccessToken  = "access_token"
	KeyRefreshToken = "refresh_token"
)

// ExtractCookieAccessToken set the JWT if found in the cookie
func ExtractCookieAccessToken(c *gin.Context, headerToken *string) {
	if headerToken == nil || *headerToken != "" {
		return
	}
	cookieToken, err := c.Cookie(KeyAccessToken)
	if err == nil && cookieToken != "" {
		*headerToken = "Bearer " + cookieToken
	}
}

// AuthenticationMiddleware checks if the user has a valid JWT
func AuthenticationMiddleware(accessTokenSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader(KeyAuthorization)
		ExtractCookieAccessToken(c, &tokenString)
		if tokenString == "" {
			if c.GetHeader(KeyAccept) != ContentTypeJSON {
				c.Redirect(http.StatusFound, "/login")
				return
			}
			c.Error(fmt.Errorf("missing authentication token"))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication token"})
			c.Abort()
			return
		}

		// The token should be prefixed with "Bearer "
		tokenParts := strings.Split(tokenString, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Error(fmt.Errorf("invalid structure token"))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid structure token"})
			c.Abort()
			return
		}

		tokenString = tokenParts[1]

		claims, err := tokenutil.VerifyToken(tokenString, []byte(accessTokenSecret))
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
			c.Abort()
			return
		}

		c.Set(KeyUserID, claims.UserID)
		c.Set(KeyRole, claims.Subject)

		c.Next() // Proceed to the next handler if authorized
	}
}

// Role-based Middleware
func RoleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := c.Get(KeyRole)
		if !ok || role == "" {
			c.Error(fmt.Errorf("missing role on headers"))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing role"})
			c.Abort()
			return
		}

		if role != requiredRole {
			c.Error(fmt.Errorf("%s: insufficient privileges", role))
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient privileges"})
			c.Abort()
			return
		}

		// Allow access
		c.Next()
	}
}
