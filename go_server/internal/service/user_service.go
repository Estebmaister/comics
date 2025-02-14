package service

import (
	"bytes"
	"context"
	"fmt"

	"comics/bootstrap"
	"comics/domain"
	"comics/internal/tokenutil"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	// UUID namespace for generating user IDs, 16 bytes
	userNamespace = "user-uuid-gen-01"
	// Error to wrap existing credentials on creation
	ErrCredsAlreadyExist = fmt.Errorf("already exist")
)

// userServiceImpl implements UserService
type userServiceImpl struct {
	userRepo domain.UserStore
	env      *bootstrap.Env
}

// NewUserService creates a new UserService instance
func NewUserService(userRepo domain.UserStore, env *bootstrap.Env) domain.UserService {
	return &userServiceImpl{
		userRepo: userRepo,
		env:      env,
	}
}

// GetByID returns a user by an ID normally extracted from a JWT
func (s *userServiceImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// Login authenticates a user by email and password
func (s *userServiceImpl) Login(ctx context.Context, user domain.LoginRequest) (*domain.User, error) {
	// Fetch user by email
	dbUser, err := s.userRepo.GetByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	// Compare passwords
	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password)); err != nil {
		return nil, err
	}

	return dbUser, err
}

// Register creates a new user
func (s *userServiceImpl) Register(ctx context.Context, user domain.SignUpRequest) (*domain.User, error) {
	// Check if user exists by email
	if _, err := s.userRepo.GetByEmail(ctx, user.Email); err == nil {
		return nil, fmt.Errorf("email %w", ErrCredsAlreadyExist)
	}

	// Check if user exists by username
	if _, err := s.userRepo.GetByUsername(ctx, user.Username); err == nil {
		return nil, fmt.Errorf("username %w", ErrCredsAlreadyExist)
	}

	// Generate UUID
	newID, err := uuid.NewV7FromReader(bytes.NewReader([]byte(userNamespace)))
	if err != nil {
		newID = uuid.New() // Fallback
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password")
	}

	// Create user
	dbUser := &domain.User{
		ID:       newID,
		Email:    user.Email,
		Username: user.Username,
		Password: string(hashedPassword),
		Role:     tokenutil.ROLE_USER,
		Active:   true,
	}

	// Store in database
	if err := s.userRepo.Create(ctx, dbUser); err != nil {
		return nil, fmt.Errorf("failed to create user")
	}

	return dbUser, nil
}
