package tokenutil

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/google/uuid"
)

var (
	secretKey     = []byte("secretpassword") // TODO: move to secrets
	tokenDuration = time.Hour * 1            // Token valid for 1 hour
)

// GenerateToken generates a JWT with the user ID as part of the claims
func GenerateToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{}
	claims["user_id"] = userID
	claims["exp"] = time.Now().Add(tokenDuration).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// VerifyToken verifies a JWT validate
func VerifyToken(tokenString string) (jwt.MapClaims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, VerifySigningMethod)
	if err != nil {
		return nil, err
	}

	// Validate the token
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// VerifySigningMethod verifies the signing method for JWT
func VerifySigningMethod(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Claims.(*jwt.StandardClaims); !ok {
		return nil, fmt.Errorf("invalid signing method")
	}
	return secretKey, nil
}
