package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"comics/internal/tokenutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	bearer        = "Bearer "
	accessGranted = "Access Granted"
)

var (
	accessGrantedMsg = gin.H{"message": accessGranted}
	secretKey        string
	validUserToken   string
	validAdminToken  string
	noRoleToken      string
)

func init() {
	secretKey = os.Getenv("ACCESS_TOKEN_SECRET")
	validUserToken, _ = tokenutil.GenerateToken(
		uuid.New(), []byte(secretKey), time.Minute*5)
	validAdminToken, _ = tokenutil.GenerateTokenWithRole(
		uuid.New(), []byte(secretKey), time.Minute*5, tokenutil.RoleAdmin)
	noRoleToken, _ = tokenutil.GenerateTokenWithRole(
		uuid.New(), []byte(secretKey), time.Minute*5, "")
}

func TestAuthenticationMiddleware(t *testing.T) {
	t.Parallel()
	type args struct {
		secretKey string
		token     string
	}
	type want struct {
		code int
		msg  string
	}
	tests := []struct {
		name string
		args args
		want want
	}{{
		name: "Valid Token",
		args: args{secretKey: secretKey, token: bearer + validUserToken},
		want: want{code: http.StatusOK, msg: accessGranted},
	}, {
		name: "Invalid Token from different secret key",
		args: args{secretKey: "wrong", token: bearer + validUserToken},
		want: want{code: http.StatusUnauthorized, msg: "Invalid authentication token"},
	}, {
		name: "No token",
		args: args{secretKey: secretKey, token: ""},
		want: want{code: http.StatusUnauthorized, msg: "Missing authentication token"},
	}, {
		name: "Wrong structured token",
		args: args{secretKey: secretKey, token: validUserToken},
		want: want{code: http.StatusUnauthorized, msg: "Invalid structure token"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.Default()
			router.Use(AuthenticationMiddleware(tt.args.secretKey))
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, accessGrantedMsg)
			})

			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set(KeyAccept, ContentTypeJSON)
			req.Header.Set(KeyAuthorization, tt.args.token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.want.code, w.Code)
			assert.Contains(t, w.Body.String(), tt.want.msg)
		})
	}
}

func TestRoleMiddleware(t *testing.T) {
	t.Parallel()
	type args struct {
		token string
		role  string
	}
	type want struct {
		code int
		msg  string
	}
	tests := []struct {
		name string
		args args
		want want
	}{{
		name: "Valid Role",
		args: args{role: tokenutil.RoleAdmin, token: bearer + validAdminToken},
		want: want{code: http.StatusOK, msg: accessGranted},
	}, {
		name: "Invalid Role from token",
		args: args{role: tokenutil.RoleAdmin, token: bearer + validUserToken},
		want: want{code: http.StatusForbidden, msg: "Insufficient privileges"},
	}, {
		name: "No token",
		args: args{role: tokenutil.RoleAdmin, token: bearer + noRoleToken},
		want: want{code: http.StatusUnauthorized, msg: "Missing role"},
	}, {
		name: "Wrong structured token",
		args: args{role: tokenutil.RoleAdmin, token: validUserToken},
		want: want{code: http.StatusUnauthorized, msg: "Invalid structure token"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.Default()
			router.Use(AuthenticationMiddleware(secretKey))
			router.Use(RoleMiddleware(tt.args.role))
			router.GET("/role-protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, accessGrantedMsg)
			})

			req, _ := http.NewRequest("GET", "/role-protected", nil)
			req.Header.Set(KeyAuthorization, tt.args.token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.want.code, w.Code)
			assert.Contains(t, w.Body.String(), tt.want.msg)
		})
	}
}
