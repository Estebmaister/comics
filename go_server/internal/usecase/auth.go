package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"comics/domain"
	"comics/internal/tokenutil"
)

type AuthConfig struct {
	AccessTokenExpiryHour  time.Duration
	RefreshTokenExpiryHour time.Duration
	AccessTokenSecret      string
	RefreshTokenSecret     string
}

type AuthService struct {
	userService domain.UserServicer
	config      AuthConfig
}

func NewAuthService(userService domain.UserServicer, config AuthConfig) *AuthService {
	return &AuthService{userService: userService, config: config}
}

func (s *AuthService) GetAccessTokenExpirySeconds() int {
	return int(s.config.AccessTokenExpiryHour.Seconds())
}

func (s *AuthService) GetRefreshTokenExpirySeconds() int {
	return int(s.config.RefreshTokenExpiryHour.Seconds())
}

func (s *AuthService) GetUserByJWT(ctx context.Context, accessToken string) (*domain.User, error) {
	secretKey, _, err := s.secretKeys()
	if err != nil {
		return nil, err
	}
	token := strings.TrimPrefix(accessToken, "Bearer ")
	claims, err := tokenutil.VerifyToken(token, secretKey)
	if err != nil {
		return nil, err
	}
	return s.userService.GetByID(ctx, claims.UserID)
}

func (s *AuthService) Login(ctx context.Context, accessToken string, user domain.LoginRequest) (*domain.AuthResponse, error) {
	secretKey, _, err := s.secretKeys()
	if err != nil {
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	if accessToken != "" {
		token := strings.TrimPrefix(accessToken, "Bearer ")
		claims, err := tokenutil.VerifyToken(token, secretKey)
		if err == nil {
			return &domain.AuthResponse{
				Status:  http.StatusOK,
				Message: "Authenticated with token",
				Data: &domain.AuthData{
					UserID:      claims.UserID.String(),
					AccessToken: token,
				},
			}, nil
		}
	}

	if user.Email == "" || user.Password == "" {
		err := fmt.Errorf("login failed: missing fields, email and password required")
		return &domain.AuthResponse{Status: http.StatusBadRequest, Message: "Invalid data, missing fields"}, err
	}

	dbUser, err := s.userService.Login(ctx, user)
	if err != nil {
		err := fmt.Errorf("%w: %v", domain.ErrInvalidCredentials, err)
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: domain.ErrInvalidCredentials.Error()}, err
	}

	accessToken, refreshToken, err := s.generateJWTs(dbUser)
	if err != nil {
		err := fmt.Errorf("error generating token: %w", err)
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: "error generating token"}, err
	}

	return &domain.AuthResponse{
		Status:  http.StatusOK,
		Message: "Login successful",
		Data: &domain.AuthData{
			UserID:       dbUser.ID.String(),
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}, nil
}

func (s *AuthService) LoginByOAuthEmail(ctx context.Context, email string) (*domain.AuthResponse, error) {
	dbUser, err := s.userService.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: domain.ErrInvalidCredentials.Error()}, err
		}
		err := fmt.Errorf("%w: %v", domain.ErrInvalidCredentials, err)
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: domain.ErrInvalidCredentials.Error()}, err
	}

	accessToken, refreshToken, err := s.generateJWTs(dbUser)
	if err != nil {
		err := fmt.Errorf("error generating token: %w", err)
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: "error generating token"}, err
	}

	return &domain.AuthResponse{
		Status:  http.StatusOK,
		Message: "Login successful",
		Data: &domain.AuthData{
			UserID:       dbUser.ID.String(),
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}, nil
}

func (s *AuthService) UpdateProfile(
	ctx context.Context,
	accessToken string,
	user domain.UpdateRequest,
) (*domain.AuthResponse, error) {
	dbUser, err := s.GetUserByJWT(ctx, accessToken)
	if err != nil {
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: err.Error()}, err
	}

	if err := s.userService.Update(ctx, dbUser, user); err != nil {
		if errors.Is(err, domain.ErrCredentialsAlreadyExist) {
			return &domain.AuthResponse{Status: http.StatusConflict, Message: err.Error()}, err
		}
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	return &domain.AuthResponse{Status: http.StatusOK, Message: "Update successful"}, nil
}

func (s *AuthService) Register(ctx context.Context, user domain.SignUpRequest) (*domain.AuthResponse, error) {
	dbUser, err := s.userService.Register(ctx, user)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialsAlreadyExist) {
			return &domain.AuthResponse{Status: http.StatusConflict, Message: err.Error()}, err
		}
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	accessToken, refreshToken, err := s.generateJWTs(dbUser)
	if err != nil {
		err := fmt.Errorf("error generating token: %w", err)
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: "error generating token"}, err
	}

	return &domain.AuthResponse{
		Status:  http.StatusCreated,
		Message: "User registered successfully",
		Data: &domain.AuthData{
			UserID:       dbUser.ID.String(),
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}, nil
}

func (s *AuthService) RefreshToken(_ context.Context, refreshToken string, role string) (*domain.AuthResponse, error) {
	secretKey, refreshSecretKey, err := s.secretKeys()
	if err != nil {
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	if refreshToken == "" {
		err := fmt.Errorf("no access token provided")
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: err.Error()}, err
	}
	token := strings.TrimPrefix(refreshToken, "Bearer ")

	claims, err := tokenutil.VerifyToken(token, refreshSecretKey)
	if err != nil {
		err := fmt.Errorf("invalid refresh token")
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: err.Error()}, err
	}
	if claims.Subject != role {
		err := fmt.Errorf("invalid role")
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: err.Error()}, err
	}

	accessToken, err := tokenutil.GenerateToken(claims.UserID, secretKey, s.config.AccessTokenExpiryHour)
	if err != nil {
		err := fmt.Errorf("error generating token")
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	return &domain.AuthResponse{
		Status: http.StatusOK,
		Data: &domain.AuthData{
			UserID:       claims.UserID.String(),
			AccessToken:  accessToken,
			RefreshToken: token,
		},
		Message: "Authenticated with token",
	}, nil
}

func (s *AuthService) secretKeys() (secretKey []byte, refreshSecretKey []byte, err error) {
	secretKey = []byte(s.config.AccessTokenSecret)
	refreshSecretKey = []byte(s.config.RefreshTokenSecret)
	if len(secretKey) == 0 || len(refreshSecretKey) == 0 {
		err = fmt.Errorf("JWT secret keys not set")
	}
	return
}

func (s *AuthService) generateJWTs(user *domain.User) (string, string, error) {
	secretKey, refreshSecretKey, err := s.secretKeys()
	if err != nil {
		return "", "", err
	}
	accessToken, err := tokenutil.GenerateTokenWithRole(
		user.ID,
		secretKey,
		s.config.AccessTokenExpiryHour,
		user.Role,
	)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := tokenutil.GenerateTokenWithRole(
		user.ID,
		refreshSecretKey,
		s.config.RefreshTokenExpiryHour,
		user.Role,
	)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}
