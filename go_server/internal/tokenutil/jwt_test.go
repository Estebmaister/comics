package tokenutil

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("JWT_SECRET_KEY", "test-secret-key")
	os.Setenv("JWT_REFRESH_SECRET_KEY", "test-refresh-secret-key")
}

func TestGenerateToken(t *testing.T) {
	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	userID := uuid.New()

	token, err := GenerateToken(userID, secretKey, time.Hour)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateTokenWithRole(t *testing.T) {
	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	userID := uuid.New()

	token, err := GenerateTokenWithRole(userID, secretKey, time.Hour, ROLE_ADMIN)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestRefreshToken(t *testing.T) {
	secretKey := []byte(os.Getenv("JWT_REFRESH_SECRET_KEY"))
	badToken := "xxx-bad-token"
	userID := uuid.New()
	token, _ := GenerateTokenWithRole(userID, secretKey, time.Hour, ROLE_ADMIN)

	token, err := RefreshTokenWithRole(token, secretKey, secretKey, ROLE_ADMIN)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	token, err = RefreshToken(token, secretKey, secretKey)
	assert.Error(t, err)
	assert.Empty(t, token)

	token, err = RefreshTokenWithRole(badToken, secretKey, secretKey, ROLE_ADMIN)
	assert.Error(t, err)
	assert.Empty(t, token)
}

func TestVerifyToken_IncorrectSigningMethod(t *testing.T) {
	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))

	// New token with different signing method
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	token := jwt.New(jwt.SigningMethodPS256)
	tokenString, err := token.SignedString(privateKey)
	assert.NoError(t, err)

	claims, err := VerifyToken(tokenString, secretKey)

	assert.Error(t, err)
	assert.Empty(t, claims)
}

func TestVerifyToken_InvalidToken(t *testing.T) {
	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	tokenString := "XXXXX"

	claims, err := VerifyToken(tokenString, secretKey)

	assert.Error(t, err)
	assert.Empty(t, claims)
}

func TestVerifyToken(t *testing.T) {
	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))

	type args struct {
		SecretKey []byte
		Role      string
		time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "valid token",
			args: args{
				SecretKey: secretKey,
				Role:      "user",
				Duration:  time.Hour,
			},
			want:    "user",
			wantErr: false,
		},
		{
			name: "expired token",
			args: args{
				SecretKey: secretKey,
				Role:      "user",
				Duration:  time.Nanosecond,
			},
			wantErr: true,
		},
		{
			name: "wrong secret key",
			args: args{
				SecretKey: []byte("wrong-secret-key"),
				Role:      "user",
				Duration:  time.Hour,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()
			tokenString, _ := GenerateTokenWithRole(userID, secretKey, tt.args.Duration, tt.args.Role)
			claims, err := VerifyToken(tokenString, tt.args.SecretKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyToken() error = %v, wantErr %v, claims: %v", err, tt.wantErr, claims)
				return
			} else if err != nil {
				return
			}
			if !reflect.DeepEqual(claims.Subject, tt.want) {
				t.Errorf("VerifyToken() = %#v, want %#v", claims.Subject, tt.want)
			}
		})
	}
}
