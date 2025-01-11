package tokenutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

var (
	defaultTokenDuration = 1 // Token valid for 1 hour
)

// GenerateToken generates a JWT with the user ID and role as part of the claims
func GenerateTokeWithRole(userID uuid.UUID, secretKey []byte, tokenDuration int, role string) (string, error) {
	// Set the token duration to the default if not provided
	if tokenDuration == 0 {
		tokenDuration = defaultTokenDuration
	}
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(tokenDuration) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// GenerateToken generates a JWT with the user ID as part of the claims
func GenerateToken(userID uuid.UUID, secretKey []byte, tokenDuration int) (string, error) {
	return GenerateTokeWithRole(userID, secretKey, tokenDuration, "")
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

	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	return claims, nil
}

// RefreshToken generates a new access token if the refresh token is valid
func RefreshToken(refreshToken string, refreshSecretKey, accessSecretKey []byte) (string, error) {

	claims, err := VerifyToken(refreshToken, refreshSecretKey)
	if err != nil {
		return "", err
	}

	// Generate a new access token (short-lived)
	return GenerateToken(claims.UserID, accessSecretKey, defaultTokenDuration)
}
