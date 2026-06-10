package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"comics/domain"
	"comics/internal/tokenutil"

	"github.com/google/uuid"
)

type fakeUserService struct {
	user *domain.User
	err  error
}

func (s fakeUserService) Login(context.Context, domain.LoginRequest) (*domain.User, error) {
	return s.user, s.err
}

func (s fakeUserService) Register(context.Context, domain.SignUpRequest) (*domain.User, error) {
	return s.user, s.err
}

func (s fakeUserService) Update(context.Context, *domain.User, domain.UpdateRequest) error {
	return s.err
}

func (s fakeUserService) GetByID(context.Context, uuid.UUID) (*domain.User, error) {
	return s.user, s.err
}

func (s fakeUserService) GetByEmail(context.Context, string) (*domain.User, error) {
	return s.user, s.err
}

func testAuthConfig() AuthConfig {
	return AuthConfig{
		AccessTokenExpiryHour:  time.Hour,
		RefreshTokenExpiryHour: 24 * time.Hour,
		AccessTokenSecret:      "access-secret",
		RefreshTokenSecret:     "refresh-secret",
	}
}

func TestAuthServiceTokenLogin(t *testing.T) {
	ctx := context.Background()
	user := &domain.User{ID: uuid.New(), Role: tokenutil.RoleUser}
	token, err := tokenutil.GenerateTokenWithRole(user.ID, []byte(testAuthConfig().AccessTokenSecret), time.Hour, user.Role)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewAuthService(fakeUserService{user: user}, testAuthConfig())

	resp, err := svc.Login(ctx, "Bearer "+token, domain.LoginRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Data == nil || resp.Data.UserID != user.ID.String() || resp.Data.AccessToken != token {
		t.Fatalf("expected token login response, got %#v", resp)
	}
}

func TestAuthServicePasswordLoginInvalidCredentials(t *testing.T) {
	ctx := context.Background()
	svc := NewAuthService(fakeUserService{err: domain.ErrInvalidCredentials}, testAuthConfig())

	resp, err := svc.Login(ctx, "", domain.LoginRequest{Email: "a@example.com", Password: "bad"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
	if resp.Status != 401 {
		t.Fatalf("expected 401 response, got %#v", resp)
	}
}

func TestAuthServiceRefreshTokenRoleMismatch(t *testing.T) {
	userID := uuid.New()
	token, err := tokenutil.GenerateTokenWithRole(
		userID,
		[]byte(testAuthConfig().RefreshTokenSecret),
		time.Hour,
		tokenutil.RoleUser,
	)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewAuthService(fakeUserService{}, testAuthConfig())

	resp, err := svc.RefreshToken(context.Background(), "Bearer "+token, tokenutil.RoleAdmin)
	if err == nil {
		t.Fatal("expected role mismatch error")
	}
	if resp.Status != 401 || resp.Message != "invalid role" {
		t.Fatalf("expected invalid role response, got %#v", resp)
	}
}

func TestAuthServiceRegisterConflict(t *testing.T) {
	svc := NewAuthService(fakeUserService{err: domain.ErrCredentialsAlreadyExist}, testAuthConfig())

	resp, err := svc.Register(context.Background(), domain.SignUpRequest{Email: "a@example.com"})
	if !errors.Is(err, domain.ErrCredentialsAlreadyExist) {
		t.Fatalf("expected credentials conflict, got %v", err)
	}
	if resp.Status != 409 {
		t.Fatalf("expected 409 response, got %#v", resp)
	}
}
