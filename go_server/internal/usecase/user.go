package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"comics/domain"
	"comics/internal/repo"
	"comics/internal/tokenutil"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

var userNamespace = "user-uuid-gen-01"

var _ domain.UserServicer = (*UserService)(nil)

type UserService struct {
	userRepo domain.UserStore
}

func NewUserService(userRepo domain.UserStore) domain.UserServicer {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	return user, mapUserRepoError(err)
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	return user, mapUserRepoError(err)
}

func (s *UserService) Login(ctx context.Context, user domain.LoginRequest) (*domain.User, error) {
	dbUser, err := s.GetByEmail(ctx, user.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidCredentials, err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password)); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidCredentials, err)
	}
	return dbUser, nil
}

func (s *UserService) Update(ctx context.Context, dbUser *domain.User, user domain.UpdateRequest) error {
	g, groupCtx := errgroup.WithContext(ctx)

	if user.Email != "" && user.Email != dbUser.Email {
		g.Go(func() error {
			existingUser, err := s.userRepo.GetByEmail(groupCtx, user.Email)
			if err == nil && existingUser.ID != dbUser.ID {
				return fmt.Errorf("email %w", domain.ErrCredentialsAlreadyExist)
			}
			dbUser.Email = user.Email
			return nil
		})
	}

	if user.Username != "" && user.Username != dbUser.Username {
		g.Go(func() error {
			existingUser, err := s.userRepo.GetByUsername(groupCtx, user.Username)
			if err == nil && existingUser.ID != dbUser.ID {
				return fmt.Errorf("username %w", domain.ErrCredentialsAlreadyExist)
			}
			dbUser.Username = user.Username
			return nil
		})
	}

	if user.Password != "" {
		g.Go(func() error {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("error hashing password: %w", err)
			}
			dbUser.Password = string(hashedPassword)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}
	if err := s.userRepo.Update(ctx, dbUser); err != nil {
		return fmt.Errorf("failed to update user: %w", mapUserRepoError(err))
	}
	return nil
}

func (s *UserService) Register(ctx context.Context, user domain.SignUpRequest) (*domain.User, error) {
	if err := s.checkUserExistence(ctx, user); err != nil {
		return nil, err
	}

	newID, err := uuid.NewV7FromReader(bytes.NewReader([]byte(userNamespace)))
	if err != nil {
		newID = uuid.New()
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	dbUser := &domain.User{
		ID:       newID,
		Email:    user.Email,
		Username: user.Username,
		Password: string(hashedPassword),
		Role:     tokenutil.RoleUser,
		Active:   true,
	}
	if err := s.userRepo.Create(ctx, dbUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", mapUserRepoError(err))
	}
	return dbUser, nil
}

func (s *UserService) checkUserExistence(ctx context.Context, user domain.SignUpRequest) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if _, err := s.userRepo.GetByEmail(ctx, user.Email); err == nil {
			return fmt.Errorf("email %w", domain.ErrCredentialsAlreadyExist)
		}
		return nil
	})

	if user.Username != "" {
		g.Go(func() error {
			if _, err := s.userRepo.GetByUsername(ctx, user.Username); err == nil {
				return fmt.Errorf("username %w", domain.ErrCredentialsAlreadyExist)
			}
			return nil
		})
	}

	return g.Wait()
}

func mapUserRepoError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repo.ErrNotFound) {
		return domain.ErrUserNotFound
	}
	return err
}
