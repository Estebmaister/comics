package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"comics/internal/tokenutil"

	"github.com/gin-gonic/gin"
)

// Headers
const (
	KeyUserID        = "user_id"
	KeyRole          = "role"
	KeyAuthorization = "Authorization"
	KeyAccept        = "Accept"
	ContentTypeJSON  = "application/json"
)

// Cookies
const (
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

// AuthenticationMiddleware validates an access JWT and permits the existing
// HttpOnly cookie fallback used by browser profile/dashboard routes.
func AuthenticationMiddleware(accessTokenSecret string) gin.HandlerFunc {
	return authenticationMiddleware(accessTokenSecret, true)
}

// BearerAuthenticationMiddleware validates only an Authorization bearer token.
// API callers never fall back to browser cookies or redirects.
func BearerAuthenticationMiddleware(accessTokenSecret string) gin.HandlerFunc {
	return authenticationMiddleware(accessTokenSecret, false)
}

func authenticationMiddleware(accessTokenSecret string, allowCookie bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader(KeyAuthorization)
		if allowCookie {
			ExtractCookieAccessToken(c, &tokenString)
		}

		if tokenString == "" {
			if allowCookie && c.GetHeader(KeyAccept) != ContentTypeJSON {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}

			c.Error(fmt.Errorf("missing authentication token")) // nolint:errcheck
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication token"})
			c.Abort()
			return
		}

		tokenParts := strings.Split(tokenString, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" || tokenParts[1] == "" {
			c.Error(fmt.Errorf("invalid authentication token structure")) // nolint:errcheck
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid structure token"})
			c.Abort()
			return
		}

		claims, err := tokenutil.VerifyToken(tokenParts[1], []byte(accessTokenSecret))
		if err != nil {
			c.Error(err) // nolint:errcheck
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
			c.Abort()
			return
		}

		c.Set(KeyUserID, claims.UserID)
		c.Set(KeyRole, claims.Subject)
		c.Next()
	}
}

// RoleMiddleware checks if the user has the required role (extracted from JWT)
func RoleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := c.Get(KeyRole)
		if !ok || role == "" {
			c.Error(fmt.Errorf("missing role on headers")) // nolint:errcheck
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing role"})
			c.Abort()
			return
		}

		if role != requiredRole {
			c.Error(fmt.Errorf("%s: insufficient privileges", role)) // nolint:errcheck
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient privileges"})
			c.Abort()
			return
		}

		// Allow access
		c.Next()
	}
}
