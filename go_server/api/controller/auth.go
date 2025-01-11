package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"comics/bootstrap"
	"comics/domain"
	"comics/internal/tokenutil"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	USER = "user-uuid-gen-01" // UUID namespace for generating user IDs, 16 bytes
)

func Login(accessToken string, env *bootstrap.Env, user domain.LoginRequest) (map[string]any, int, error) {
	// Load secret key from environment variable
	secretKey := []byte(env.AccessTokenSecret)
	refreshSecretKey := []byte(env.RefreshTokenSecret)
	if len(secretKey) == 0 || len(refreshSecretKey) == 0 {
		return nil, http.StatusInternalServerError, fmt.Errorf("JWT secret keys not set")
	}

	// Check for Authorization header
	if accessToken != "" {
		token := strings.TrimPrefix(accessToken, "Bearer ")

		// Validate the token
		claims, err := tokenutil.VerifyToken(token, secretKey)
		if err == nil {
			return map[string]any{
				"message": "Authenticated with token",
				"user_id": claims.UserID,
			}, http.StatusOK, nil
		}
		// If token is invalid, fallback to password-based login
	}

	// Fallback to standard password-based login
	// Simulate fetching user from database (replace with real DB call)
	dbUser, err := getUserByEmail(user.Email)
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid user credentials")
	}

	// Validate password (assuming bcrypt hash comparison)
	if err := bcrypt.CompareHashAndPassword(
		[]byte(dbUser.Password), []byte(user.Password)); err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid credentials")
	}

	// Generate a JWT
	accessToken, errAccess := tokenutil.GenerateToken(
		dbUser.ID, secretKey, env.AccessTokenExpiryHour)
	refreshToken, errRefresh := tokenutil.GenerateToken(
		dbUser.ID, refreshSecretKey, env.RefreshTokenExpiryHour)
	if errAccess != nil && errRefresh != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error generating token")
	}

	// Return tokens in response
	return map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"message":       "Login successful",
	}, http.StatusOK, nil
}

func Register(user domain.User) (map[string]any, int, error) {

	// Simulate fetching user from database (replace with real DB call)
	_, err := getUserByEmail(user.Email)
	if err == nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("email already exists")
	}

	// Simulate fetching user from database (replace with real DB call)
	_, err = getUserByUsername(user.Username)
	if err == nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("username already exists")
	}

	newID, err := uuid.NewV7FromReader(bytes.NewReader([]byte(USER)))
	if err != nil {
		println("Error generating UUID:", err.Error())
		newID = uuid.New() // Fallback to a random UUID if generation fails
	}
	user.ID = newID

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error hashing password: %w", err)
	}
	user.Password = string(hashedPassword)

	// Store the user in the database (replace this logic with real database operations)
	// ...

	return map[string]any{
		"message": "User registered successfully",
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"password": user.Password,
			"role":     user.Role,
		},
	}, http.StatusCreated, nil
}

// getUserByEmail simulates a database fetch (replace with real DB interaction)
func getUserByEmail(email string) (domain.User, error) {
	// Mocked user data for demonstration
	mockUser := domain.User{
		ID:       uuid.New(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "$2a$10$WXvnJNuriTzpJThUJ6BJWOm6cxZNhQUsITXlXJTf0VGQYKATc9QMu", // bcrypt hash for "password123"
		Role:     "user",
	}
	if email == mockUser.Email {
		return mockUser, nil
	}
	return domain.User{}, fmt.Errorf("user not found")
}

// getUserByUsername simulates a database fetch (replace with real DB interaction)
func getUserByUsername(username string) (domain.User, error) {
	// Mocked user data for demonstration
	mockUser := domain.User{
		ID:       uuid.New(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "$2a$12$KTOlJj2JZlyeZtTcfIhB5uR9l3.KYvYInPqvIjUMKYw9E6aBD5l2W", // bcrypt hash for "password123"
		Role:     "user",
	}
	if username == mockUser.Username {
		return mockUser, nil
	}
	return domain.User{}, fmt.Errorf("user not found")
}
