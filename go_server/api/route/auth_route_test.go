package route

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"comics/api/controller"
	"comics/api/middleware"
	"comics/domain"
	"comics/internal/tokenutil"
	"comics/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type memoryUserStore struct {
	mu    sync.Mutex
	users map[uuid.UUID]*domain.User
}

func newMemoryUserStore() *memoryUserStore {
	return &memoryUserStore{users: map[uuid.UUID]*domain.User{}}
}

func (s *memoryUserStore) Ping(context.Context) error { return nil }
func (s *memoryUserStore) GetStats() map[string]string {
	return map[string]string{"users": "0"}
}
func (s *memoryUserStore) Tx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
func (s *memoryUserStore) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}
func (s *memoryUserStore) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, user := range s.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}
func (s *memoryUserStore) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, user := range s.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}
func (s *memoryUserStore) List(context.Context, int, int) ([]*domain.User, int64, error) {
	return nil, 0, nil
}
func (s *memoryUserStore) FindActiveUsersByRole(context.Context, string) ([]*domain.User, error) {
	return nil, nil
}
func (s *memoryUserStore) Create(_ context.Context, user *domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.users {
		if existing.Email == user.Email || existing.Username == user.Username {
			return domain.ErrCredentialsAlreadyExist
		}
	}
	s.users[user.ID] = user
	return nil
}
func (s *memoryUserStore) Update(_ context.Context, user *domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.ID] = user
	return nil
}
func (s *memoryUserStore) Delete(context.Context, uuid.UUID) error { return nil }

func newAuthTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := newMemoryUserStore()
	userService := usecase.NewUserService(store)
	authController := controller.NewAuthControl(usecase.NewAuthService(userService, usecase.AuthConfig{
		AccessTokenExpiryHour:  time.Hour,
		RefreshTokenExpiryHour: 24 * time.Hour,
		AccessTokenSecret:      "access-secret",
		RefreshTokenSecret:     "refresh-secret",
	}))
	router := gin.New()
	group := router.Group("/")
	signUpRouter(authController, group)
	loginRouter(authController, group)
	refreshTokenRouter(authController, group)
	protected := router.Group("/protected")
	protected.Use(middleware.AuthenticationMiddleware("access-secret"))
	protected.GET("/profile", getProfile(authController))
	protected.PUT("/profile", putProfile(authController))
	return router
}

func TestAuthSignupLoginAndProfile(t *testing.T) {
	router := newAuthTestRouter(t)

	signupReq := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(`{
		"email":"reader@example.com","username":"reader","password":"secret123"
	}`))
	signupReq.Header.Set("Content-Type", "application/json")
	signupRes := httptest.NewRecorder()
	router.ServeHTTP(signupRes, signupReq)
	if signupRes.Code != http.StatusCreated {
		t.Fatalf("signup expected 201, got %d: %s", signupRes.Code, signupRes.Body.String())
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{
		"email":"reader@example.com","password":"secret123"
	}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes := httptest.NewRecorder()
	router.ServeHTTP(loginRes, loginReq)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d: %s", loginRes.Code, loginRes.Body.String())
	}
	token := loginRes.Header().Get(keyAuthorization)
	if token == "" {
		t.Fatal("expected Authorization header")
	}

	profileReq := httptest.NewRequest(http.MethodGet, "/protected/profile", nil)
	profileReq.Header.Set(keyAuthorization, token)
	profileReq.Header.Set(keyAccept, contentTypeJSON)
	profileRes := httptest.NewRecorder()
	router.ServeHTTP(profileRes, profileReq)
	if profileRes.Code != http.StatusOK {
		t.Fatalf("profile expected 200, got %d: %s", profileRes.Code, profileRes.Body.String())
	}
}

func TestAuthSignupRejectsDuplicateEmail(t *testing.T) {
	store := newMemoryUserStore()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	id := uuid.New()
	_ = store.Create(context.Background(), &domain.User{
		ID: id, Email: "dup@example.com", Username: "first", Password: string(hashed), Role: tokenutil.RoleUser,
	})
	userService := usecase.NewUserService(store)
	authController := controller.NewAuthControl(usecase.NewAuthService(userService, usecase.AuthConfig{
		AccessTokenExpiryHour:  time.Hour,
		RefreshTokenExpiryHour: 24 * time.Hour,
		AccessTokenSecret:      "access-secret",
		RefreshTokenSecret:     "refresh-secret",
	}))
	router := gin.New()
	signUpRouter(authController, router.Group("/"))

	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(`{
		"email":"dup@example.com","username":"second","password":"secret123"
	}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", res.Code, res.Body.String())
	}
}

func TestAuthRefreshTokenAfterLogin(t *testing.T) {
	router := newAuthTestRouter(t)

	signupReq := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(`{
		"email":"refresh@example.com","username":"refresh","password":"secret123"
	}`))
	signupReq.Header.Set("Content-Type", "application/json")
	signupRes := httptest.NewRecorder()
	router.ServeHTTP(signupRes, signupReq)
	if signupRes.Code != http.StatusCreated {
		t.Fatalf("signup expected 201, got %d", signupRes.Code)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{
		"email":"refresh@example.com","password":"secret123"
	}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes := httptest.NewRecorder()
	router.ServeHTTP(loginRes, loginReq)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d", loginRes.Code)
	}
	refresh := loginRes.Result().Cookies()
	var refreshToken string
	for _, cookie := range refresh {
		if cookie.Name == cookieRefreshToken {
			refreshToken = cookie.Value
		}
	}
	if refreshToken == "" {
		t.Fatal("expected refresh token cookie")
	}

	req := httptest.NewRequest(http.MethodPost, "/refresh-token", nil)
	req.Header.Set(keyAuthorization, "Bearer "+refreshToken)
	req.Header.Set(keyRole, tokenutil.RoleUser)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("refresh expected 200, got %d: %s", res.Code, res.Body.String())
	}
}
