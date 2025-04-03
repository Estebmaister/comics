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

// Update updates a user's information
// Email and username can't match other users to guarantee uniqueness
func (s *userService) Update(ctx context.Context, dbUser *domain.User, user domain.UpdateRequest) error {
	g, groupCtx := errgroup.WithContext(ctx)

	// Apply updates if provided
	if user.Email != "" && user.Email != dbUser.Email {
		// Check if email already exists
		g.Go(func() error {
			existingUser, err := s.userRepo.GetByEmail(groupCtx, user.Email)
			if err == nil && existingUser.ID != dbUser.ID {
				return fmt.Errorf("email %w", ErrCredsAlreadyExist)
			}
			dbUser.Email = user.Email
			return nil
		})
	}

	if user.Username != "" && user.Username != dbUser.Username {
		// Check if username already exists
		g.Go(func() error {
			existingUser, err := s.userRepo.GetByUsername(groupCtx, user.Username)
			if err == nil && existingUser.ID != dbUser.ID {
				return fmt.Errorf("username %w", ErrCredsAlreadyExist)
			}
			dbUser.Username = user.Username
			return nil
		})
	}

	if user.Password != "" {
		// Hash new password
		g.Go(func() error {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("error hashing password: %w", err)
			}
			dbUser.Password = string(hashedPassword)
			return nil
		})
	}

	// Wait for all goroutines to finish and return the first error encountered, if any
	if err := g.Wait(); err != nil {
		return err
	}

	// Update user in the repository
	if err := s.userRepo.Update(ctx, dbUser); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
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
		return nil, fmt.Errorf("error hashing password: %w", err)
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

// checkUserExistence ia a helper function to perform concurrent checks of
// username and email using errgroup
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
