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
	"golang.org/x/sync/errgroup"
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

// GetByID returns a user by an ID, which is normally extracted from a JWT
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
	if err := s.checkUserExistence(ctx, user); err != nil {
		return nil, err
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

// Helper function to perform concurrent checks using errgroup
func (s *userServiceImpl) checkUserExistence(ctx context.Context, user domain.SignUpRequest) error {
	g, ctx := errgroup.WithContext(ctx)

	// Check if user exists by email
	g.Go(func() error {
		if _, err := s.userRepo.GetByEmail(ctx, user.Email); err == nil {
			return fmt.Errorf("email %w", ErrCredsAlreadyExist)
		}
		return nil
	})

	// Check if user exists by username
	g.Go(func() error {
		if _, err := s.userRepo.GetByUsername(ctx, user.Username); err == nil {
			return fmt.Errorf("username %w", ErrCredsAlreadyExist)
		}
		return nil
	})

	// Wait for all goroutines to finish and return the first error encountered, if any
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
