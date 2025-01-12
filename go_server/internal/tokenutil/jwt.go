package tokenutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

const (
	DEFAULT_TOKEN_DURATION = 1 * time.Hour // Token valid for 1 hour
	ISSUER                 = "comic-auth-service"
	ROLE_ADMIN             = "admin"
	ROLE_USER              = "user"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT with the user ID and role as part of the claims
func GenerateTokenWithRole(userID uuid.UUID, secretKey []byte, tokenDuration time.Duration, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ISSUER,
			Subject:   role,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// GenerateToken generates a JWT with the user ID as part of the claims
func GenerateToken(userID uuid.UUID, secretKey []byte, tokenDuration time.Duration) (string, error) {
	return GenerateTokenWithRole(userID, secretKey, tokenDuration, ROLE_USER)
}

// VerifyToken verifies a JWT token
func VerifyToken(tokenString string, secretKey []byte) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token, err: %v", err)
	}

	return claims, nil
}

// RefreshToken generates a new access token if the refresh token is valid
func RefreshToken(refreshToken string, refreshSecretKey, accessSecretKey []byte) (string, error) {
	return RefreshTokenWithRole(refreshToken, refreshSecretKey, accessSecretKey, ROLE_USER)
}

// RefreshToken generates a new access token if the refresh token is valid
func RefreshTokenWithRole(refreshToken string, refreshSecretKey, accessSecretKey []byte, role string) (string, error) {

	claims, err := VerifyToken(refreshToken, refreshSecretKey)
	if err != nil {
		return "", err
	}

	if claims.Subject != role {
		return "", fmt.Errorf("invalid role to refresh given token")
	}

	// Generate a new access token (short-lived)
	return GenerateTokenWithRole(claims.UserID, accessSecretKey, DEFAULT_TOKEN_DURATION, role)
}
