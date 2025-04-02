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
	// ErrCredsAlreadyExist to wrap existing credentials on creation
	ErrCredsAlreadyExist = fmt.Errorf("already exist")
	// UUID namespace for generating user IDs, 16 bytes
	userNamespace = "user-uuid-gen-01"
)

var _ domain.UserServicer = (*userService)(nil)

// userService implements UserServicer
type userService struct {
	userRepo domain.UserStore
	env      *bootstrap.Env
}

// NewUserService creates a new UserServicer instance
func NewUserService(userRepo domain.UserStore, env *bootstrap.Env) domain.UserServicer {
	return &userService{
		userRepo: userRepo,
		env:      env,
	}
}

// GetByID returns a user by an ID, which is normally extracted from a JWT
func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// GetByEmail returns a user by an email, which is normally extracted from OAuth
func (s *userService) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// Login authenticates a user by email and password
func (s *userService) Login(ctx context.Context, user domain.LoginRequest) (*domain.User, error) {
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
func (s *userService) Register(ctx context.Context, user domain.SignUpRequest) (*domain.User, error) {
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
		Role:     tokenutil.RoleUser,
		Active:   true,
	}

	// Store in database
	if err := s.userRepo.Create(ctx, dbUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w, %v", err, dbUser)
	}

	return dbUser, nil
}

// Helper function to perform concurrent checks using errgroup
func (s *userService) checkUserExistence(ctx context.Context, user domain.SignUpRequest) error {
	g, ctx := errgroup.WithContext(ctx)

	// Check if user exists by email
	g.Go(func() error {
		if _, err := s.userRepo.GetByEmail(ctx, user.Email); err == nil {
			return fmt.Errorf("email %w", ErrCredsAlreadyExist)
		}
		return nil
	})

	// Check if user exists by username
	if user.Username != "" {
		g.Go(func() error {
			if _, err := s.userRepo.GetByUsername(ctx, user.Username); err == nil {
				return fmt.Errorf("username %w", ErrCredsAlreadyExist)
			}
			return nil
		})
	}

	// Wait for all goroutines to finish and return the first error encountered, if any
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
