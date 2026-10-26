# Authentication Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden Mongo-backed account authentication, Google OAuth, JWT verification, refresh sessions, cookies, CORS, frontend role handling, secret rotation, and logout without adding Redis or splitting the Go modular monolith.

**Architecture:** Keep MongoDB as the user/auth store and add two Mongo collections owned by the existing auth repository: `auth_sessions` for one-time rotating refresh sessions and `oauth_states` for one-time, expiring OAuth state/PKCE records. JWT access tokens remain short-lived bearer credentials, but authorization always reloads the active user and current role from Mongo; refresh tokens rotate against an atomic Mongo session record, and the browser receives refresh tokens only through an HttpOnly cookie.

**Tech Stack:** Go 1.27.3, Gin 1.12, MongoDB Go Driver v2.9.2, `golang-jwt/jwt/v4` v4.5.2, `golang.org/x/oauth2` v0.37.0, React 19, TypeScript 6, Vitest 3, MongoDB 7/Testcontainers.

## Global Constraints

- Keep the Go modular monolith as the production service.
- Keep React + REST JSON as the production client contract.
- Keep MongoDB as the user/auth store.
- Do not introduce Redis or another runtime service.
- Public catalog reads remain public; this workstream supplies the authenticated identity and trusted role contract used by protected/admin routes.
- A role received from a browser header, request body, session storage, or decoded client-side JWT is never an authorization source.
- Existing sessions may be signed out once when the claim/session model is deployed; logout must still clear both legacy and new browser credentials.
- All auth datastore indexes are created by Go code for identical behavior on Atlas, local Docker, and tests. `go_server/scripts/mongo-init-local.js` continues to own only the local Mongo principal and is deliberately not given a second copy of index definitions.
- Use TDD for every behavior change. Run the focused failing test before implementation, then the focused passing test, then the wider package tests.
- Never log JWTs, OAuth authorization codes, OAuth access tokens, passwords, cookie values, signing secrets, or complete Mongo connection strings.
- When Workstream 1 route grouping lands first, preserve its public-read/admin-write groups and replace only their authenticator construction with the `domain.AccessAuthenticator` contract in this plan.
- When Workstream 4 lands first, preserve `route.Setup` returning the application readiness checker and preserve the lifecycle manager's shutdown ordering; this plan adds setup failure reporting without discarding that return value.
- Workstream 1's access-token cookie fallback is transitional. The final
  contract in this workstream is bearer-only for `/protected`, dashboard,
  catalog-admin, and scrape APIs; only the rotating refresh token remains in an
  HttpOnly cookie.
- When Workstream 5 lands first, this workstream must rewrite its scrape
  route-policy tests while replacing the middleware: use the Mongo-backed
  authenticator fake and current stored roles; no test or production path may
  retain role-in-JWT authorization after Workstream 6.

---

## Security Decisions

1. **Mongo credential uniqueness:** `email` gets a case-insensitive unique index. Non-empty `username` gets a case-insensitive partial unique index because historical OAuth rows may have an empty username. A read-only duplicate/missing-email preflight runs before index creation; it never chooses, deletes, or merges an account.
2. **Canonical credentials:** new and updated email addresses are trimmed and lowercased; usernames are trimmed but preserve display case. Mongo collation strength 2 prevents case-only duplicates. Duplicate-key write errors are mapped to `domain.ErrCredentialsAlreadyExist`, closing the race in the current check-then-insert flow.
3. **Active users:** every password login, OAuth login, access-token authentication, and refresh loads the user from Mongo and rejects `active=false`. Middleware sets the request role from that Mongo user, not from a token claim.
4. **Token claims:** new JWTs use `iss=comic-auth-service`, `aud=comics-api`, standard `sub` set to the user UUID, standard `jti` set to a random UUID, `iat`, `nbf`, and `exp`, plus `token_type` (`access` or `refresh`) and `sid` (session UUID). Tokens carry no role.
5. **Signing keys:** new tokens include a `kid`. Each access/refresh key ring supports one current key and one previous verification key, enough for bounded zero-downtime secret rotation in the current single-service deployment. Access and refresh secrets must be distinct and at least 32 bytes in production.
6. **Refresh rotation/revocation:** Mongo stores only session ID, user ID, the current refresh `jti`, timestamps, expiry, and revocation state—never a raw refresh token. Refresh atomically replaces the expected current `jti`; reuse or concurrent replay revokes that session. Session expiry is absolute, not sliding. Revocation prevents future refreshes; an already-issued access token remains valid for at most its 15-minute TTL.
7. **OAuth state:** every `/auth/google` request creates independent random state, browser nonce, and PKCE verifier material. Mongo stores hashes of state and browser nonce plus the verifier for ten minutes. Callback consumption is atomic and one-time.
8. **Google identity:** user info is fetched with the OAuth-authenticated HTTP client, response status/body size are checked, and login/registration proceeds only when `verified_email=true` and email is non-empty.
9. **Browser token storage:** access tokens remain in `sessionStorage` for bearer requests. Refresh tokens are omitted from JSON and live only in a host-only HttpOnly cookie. The old access cookie is no longer issued and is cleared on refresh/logout.
10. **Cookie/CORS/CSRF posture:** production refresh cookies are `Secure; HttpOnly; SameSite=None; Path=/` for the GitHub Pages/API cross-site deployment. OAuth nonce cookies are `Secure; HttpOnly; SameSite=Lax`. Credential-bearing browser POSTs require an allowed `Origin` and `X-CSRF-Protection: 1`; CLI requests without `Origin` remain supported.
11. **Client role handling:** the frontend obtains `user.role` from `/protected/profile`, exposes `isAdmin`, and never sends a `Role` header or decodes role from a JWT. UI role checks are presentation only; the server remains authoritative.
12. **Logout compatibility:** `POST /logout` is idempotent. It revokes a new-format session when possible, accepts a cookie or bearer refresh token, returns `204` for missing/expired/legacy credentials, and always expires `refresh_token`, `access_token`, and the OAuth nonce cookie. The frontend clears session storage before making the best-effort server call.

## File and Responsibility Map

### New files

- `go_server/domain/auth_session.go` — refresh-session and OAuth-state entities, storage interfaces, and typed session errors.
- `go_server/internal/repo/mongo/auth_indexes.go` — credential preflight, named Mongo index definitions, apply, and rollback.
- `go_server/internal/repo/mongo/auth_state.go` — atomic refresh-session and one-time OAuth-state persistence.
- `go_server/cmd/auth_migrate/main.go` — `audit`, `apply`, and `rollback-indexes` operations using the same repository code as startup.
- `go_server/bootstrap/env_test.go` — focused JWT key-ring, frontend URL, OAuth, and production-secret validation tests.
- `go_server/internal/usecase/user_test.go` — credential canonicalization, duplicate-key mapping, and inactive-login tests.
- `go_server/api/route/cookies.go` — one implementation for setting, reading, and clearing auth/OAuth cookies.
- `go_server/api/route/cookies_test.go` — cookie flags, legacy clearing, and token-source tests.
- `go_server/api/route/oauth2_test.go` — unique/expiring/replayed state, PKCE, user-info, and verified-email route tests.
- `src/frontend/util/AuthHelpers.test.ts` — refresh deduplication, role-header removal, CSRF header, and logout tests.
- `src/frontend/context/AuthContext.test.tsx` — bootstrap, trusted profile role, refresh fallback, and local-first logout tests.

### Modified files

- `go_server/internal/tokenutil/jwt.go` and `jwt_test.go` — typed claims, `kid` key rings, strict HS256/issuer/audience/type verification.
- `go_server/bootstrap/env.go` and `sanitize_test.go` — key IDs/previous keys, OAuth/session settings, validation, and redaction coverage.
- `go_server/local.env` — safe development examples and all new auth settings.
- `go_server/internal/repo/db.go` — auth collection names and `repo.ErrConflict`.
- `go_server/internal/repo/mongo/collection.go` — `Aggregate`, `UpdateOne`, `FindOneAndUpdate`, and `FindOneAndDelete` adapters needed by auth persistence.
- `go_server/internal/repo/mongo/user_repo.go` and `user_repo_test.go` — three auth collections, index initialization, collation-aware lookups, and duplicate-key mapping.
- `go_server/bootstrap/db.go`, `app.go`, and `go_server/cmd/server/main.go` — aggregate auth store wiring and fail-fast index initialization.
- `go_server/domain/auth.go` and `user.go` — hardened use-case contracts, OAuth identity, response shape, and active-user errors.
- `go_server/internal/usecase/auth.go`, `auth_test.go`, and `user.go` — session issuance/rotation/logout and active-user enforcement.
- `go_server/api/controller/auth.go` — route adapter signatures without client role input.
- `go_server/api/middleware/auth.go` and `auth_test.go` — active-user authenticator and trusted request role.
- `go_server/api/route/route.go`, `auth_route_test.go`, and `user_profile.go` — token-manager construction, refresh cookie flow, logout, origin guard, and middleware wiring.
- `go_server/api/route/oauth2.go` — request-scoped state, PKCE, authenticated user-info request, verified email, and frontend redirect.
- `go_server/Makefile` — auth migration target.
- `src/frontend/util/AuthHelpers.ts` — cookie-backed refresh/logout, refresh request deduplication, role type, and OAuth URL.
- `src/frontend/context/AuthContext.tsx` — bootstrap state, refresh fallback, `isAdmin`, and async logout.
- `src/frontend/components/Auth/LoginPage.tsx`, `AuthNav.tsx`, and `ProfilePage.tsx` — Google entry and async logout call sites.
- `go_server/README.md` and `docs/deploy-render.md` — auth model, exact rollout, secret rotation, migration, and rollback.

### Deliberately unchanged

- `go_server/scripts/mongo-init-local.js` — still creates the local database principal only. Index ownership stays in `internal/repo/mongo/auth_indexes.go`.
- Catalog storage and scraper packages — this workstream only supplies the authenticated identity contract consumed by their protected routes.

## Interface Contract Used Across Tasks

Implement these names exactly so tasks remain independently reviewable:

```go
// go_server/domain/auth_session.go
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSessionNotFound = errors.New("auth session not found")
	ErrSessionReplay   = errors.New("refresh token replay detected")
	ErrOAuthState      = errors.New("invalid or expired oauth state")
	ErrUserInactive    = errors.New("user is inactive")
)

type AuthSession struct {
	ID               uuid.UUID  `bson:"_id"`
	UserID           uuid.UUID  `bson:"user_id"`
	CurrentRefreshID string     `bson:"current_refresh_id"`
	CreatedAt        time.Time  `bson:"created_at"`
	UpdatedAt        time.Time  `bson:"updated_at"`
	ExpiresAt        time.Time  `bson:"expires_at"`
	RevokedAt        *time.Time `bson:"revoked_at,omitempty"`
}

type OAuthState struct {
	StateHash        string    `bson:"_id"`
	BrowserNonceHash string    `bson:"browser_nonce_hash"`
	PKCEVerifier     string    `bson:"pkce_verifier"`
	CreatedAt        time.Time `bson:"created_at"`
	ExpiresAt        time.Time `bson:"expires_at"`
}

type AuthSessionStore interface {
	CreateSession(context.Context, *AuthSession) error
	RotateSession(context.Context, uuid.UUID, string, string, time.Time) error
	RevokeSession(context.Context, uuid.UUID, time.Time) error
}

type OAuthStateStore interface {
	CreateOAuthState(context.Context, *OAuthState) error
	ConsumeOAuthState(context.Context, string, string, time.Time) (*OAuthState, error)
}

type AuthStore interface {
	UserStore
	AuthSessionStore
	OAuthStateStore
}
```

```go
// go_server/internal/tokenutil/jwt.go
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Claims struct {
	TokenType TokenType `json:"token_type"`
	SessionID uuid.UUID `json:"sid"`
	jwt.RegisteredClaims
}

type SigningKey struct {
	ID     string
	Secret []byte
}

type KeyRing struct {
	Current  SigningKey
	Previous *SigningKey
}

type ManagerConfig struct {
	Issuer       string
	Audience     string
	AccessKeys   KeyRing
	RefreshKeys  KeyRing
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
	ClockSkew    time.Duration
	Now          func() time.Time
}

type Manager struct {
	issuer      string
	audience    string
	accessKeys  KeyRing
	refreshKeys KeyRing
	accessTTL   time.Duration
	refreshTTL  time.Duration
	clockSkew   time.Duration
	now         func() time.Time
}

func NewManager(ManagerConfig) (*Manager, error)
func (m *Manager) NewSessionTokens(uuid.UUID, uuid.UUID, string, time.Time) (string, string, error)
func (m *Manager) NewAccessToken(uuid.UUID, uuid.UUID) (string, error)
func (m *Manager) NewRefreshToken(uuid.UUID, uuid.UUID, string, time.Time) (string, error)
func (m *Manager) VerifyAccess(string) (*Claims, error)
func (m *Manager) VerifyRefresh(string) (*Claims, error)
func (m *Manager) AccessTTL() time.Duration
func (m *Manager) RefreshTTL() time.Duration
func (c *Claims) UserID() (uuid.UUID, error)
```

`NewSessionTokens` takes `(userID, sessionID, refreshJTI, sessionExpiresAt)`. `NewAccessToken` generates its own `jti`; `NewRefreshToken` uses the supplied `refreshJTI` so Mongo and the token agree.

---

### Task 1: Strict JWT Claims and Signing-Key Rotation

**Files:**
- Modify: `go_server/internal/tokenutil/jwt.go:1-85`
- Modify: `go_server/internal/tokenutil/jwt_test.go:1-150`
- Modify: `go_server/bootstrap/env.go:39-370`
- Create: `go_server/bootstrap/env_test.go`
- Modify: `go_server/bootstrap/sanitize_test.go:1-83`
- Modify: `go_server/local.env:46-55`

**Interfaces:**
- Produces: the `tokenutil.Manager`, `ManagerConfig`, `Claims`, `SigningKey`, and `KeyRing` contract shown above.
- Produces: `bootstrap.JWTConfig.AccessTokenKeyID`, `AccessTokenPreviousKeyID`, `AccessTokenPreviousSecret`, `RefreshTokenKeyID`, `RefreshTokenPreviousKeyID`, and `RefreshTokenPreviousSecret`.
- Consumes: only standard `time`, UUID, and `jwt/v4`; no datastore.

- [ ] **Step 1: Replace token tests with the hardened claim/key-ring matrix**

Write table-driven tests in `go_server/internal/tokenutil/jwt_test.go` with a fixed clock `2026-10-26T14:00:00Z`. The tests must assert:

```go
func testManager(t *testing.T) *Manager {
	t.Helper()
	now := time.Date(2026, 10, 26, 14, 0, 0, 0, time.UTC)
	manager, err := NewManager(ManagerConfig{
		Issuer:   "comic-auth-service",
		Audience: "comics-api",
		AccessKeys: KeyRing{Current: SigningKey{
			ID: "access-2026-10", Secret: []byte("0123456789abcdef0123456789abcdef"),
		}},
		RefreshKeys: KeyRing{Current: SigningKey{
			ID: "refresh-2026-10", Secret: []byte("abcdef0123456789abcdef0123456789"),
		}},
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
		ClockSkew:  30 * time.Second,
		Now:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}
```

- Access claims have `token_type=access`, `aud=comics-api`, `iss=comic-auth-service`, `sub` equal to the user UUID, non-empty `jti`, the expected `sid`, and a 15-minute expiry.
- Refresh claims have `token_type=refresh`, the supplied refresh `jti`, the same `sid`, and the absolute session expiry.
- `VerifyAccess` rejects refresh tokens and `VerifyRefresh` rejects access tokens.
- Parsing rejects `alg=none`, RSA, missing `exp`, wrong issuer, wrong audience, unknown `kid`, expired tokens, and malformed UUID `sub`/`sid`.
- A token signed by `Previous` verifies while that key is configured and fails after it is removed.
- `NewManager` rejects blank key IDs, secrets shorter than 32 bytes, equal access/refresh current secrets, incomplete previous-key pairs, and non-positive TTLs.

- [ ] **Step 2: Run the token tests and verify the new API is missing**

Run:

```bash
cd go_server
go test ./internal/tokenutil -run 'Test(Manager|Claims|KeyRing)' -count=1
```

Expected: `FAIL` with compile errors including `undefined: NewManager`, `undefined: ManagerConfig`, and `undefined: TokenTypeAccess`.

- [ ] **Step 3: Replace the token helper with the typed manager**

Implement the interface contract above in `go_server/internal/tokenutil/jwt.go`. Use:

```go
const (
	defaultIssuer   = "comic-auth-service"
	defaultAudience = "comics-api"
)

func (m *Manager) sign(
	tokenType TokenType,
	userID uuid.UUID,
	sessionID uuid.UUID,
	tokenID string,
	expiresAt time.Time,
	ring KeyRing,
) (string, error) {
	now := m.now()
	claims := Claims{
		TokenType: tokenType,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{m.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now.Add(-m.clockSkew)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        tokenID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header[jwt.HeaderKeyID] = ring.Current.ID
	return token.SignedString(ring.Current.Secret)
}
```

Verification must select a key by `kid`, accept only `HS256`, require expiration, issuer, and audience, validate issued-at with configured skew, call `Claims.UserID()`, reject a nil session UUID or empty `jti`, and compare `token_type` with the requested type. A token without `kid` is invalid in the new claim model; the rollout intentionally performs a one-time sign-in reset.

Keep `RoleAdmin`, `RoleUser`, and the current `GenerateToken`, `GenerateTokenWithRole`, `VerifyToken`, `RefreshToken`, and `RefreshTokenWithRole` API in this compatibility type until Task 4 migrates the use case and middleware in one atomic task:

```go
type LegacyClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}
```

New code must use `Manager`. `TestLegacyJWTCompatibilityUntilHTTPMigration` generates/verifies one legacy user token so intermediate commits remain buildable. Delete the compatibility functions, test, and `LegacyClaims` at the end of Task 4.

- [ ] **Step 4: Add environment tests before environment implementation**

In `go_server/bootstrap/env_test.go`, test `Env.Validate()` directly:

```go
func validAuthEnv() *Env {
	return &Env{
		AppEnv:             Production,
		AddressHTTP:        "0.0.0.0",
		PortHTTP:           "8081",
		HostURL:            "api.example.com",
		FrontendURL:        "https://estebmaister.github.io/comics",
		CORSAllowedOrigins: "https://estebmaister.github.io",
		ComicsDBDriver:     "sqlite",
		ComicsSQLitePath:   "../db/comics.db",
		DBConfig: &repo.DBConfig{
			Addr: "mongodb://localhost:27017", Name: "comics",
		},
		JWTConfig: &JWTConfig{
			AccessTokenExpiryHour:  15 * time.Minute,
			RefreshTokenExpiryHour: 168 * time.Hour,
			AccessTokenKeyID:       "access-2026-10",
			AccessTokenSecret:      "0123456789abcdef0123456789abcdef",
			RefreshTokenKeyID:      "refresh-2026-10",
			RefreshTokenSecret:     "abcdef0123456789abcdef0123456789",
		},
	}
}
```

Assert production rejects short/equal current secrets, one-sided previous-key configuration, duplicate current/previous key IDs, malformed `FrontendURL`, wildcard CORS with credentials, and only one Google credential. Assert development accepts the committed development secrets. Extend `sanitize_test.go` to assert every current and previous secret is redacted and key IDs remain visible.

- [ ] **Step 5: Run environment tests and verify the fields are missing**

Run:

```bash
go test ./bootstrap -run 'Test(ValidateAuth|SanitizeJWT)' -count=1
```

Expected: `FAIL` with compile errors for `FrontendURL` and the new JWT key fields.

- [ ] **Step 6: Extend and validate auth environment configuration**

Add these exact fields:

```go
FrontendURL   string        `mapstructure:"FRONTEND_URL"`
OAuthStateTTL time.Duration `mapstructure:"OAUTH_STATE_TTL"`

type JWTConfig struct {
	AccessTokenExpiryHour      time.Duration `mapstructure:"ACCESS_TOKEN_EXPIRY_HOUR"`
	RefreshTokenExpiryHour     time.Duration `mapstructure:"REFRESH_TOKEN_EXPIRY_HOUR"`
	AccessTokenKeyID           string        `mapstructure:"ACCESS_TOKEN_KEY_ID"`
	AccessTokenSecret          string        `mapstructure:"ACCESS_TOKEN_SECRET"`
	AccessTokenPreviousKeyID   string        `mapstructure:"ACCESS_TOKEN_PREVIOUS_KEY_ID"`
	AccessTokenPreviousSecret  string        `mapstructure:"ACCESS_TOKEN_PREVIOUS_SECRET"`
	RefreshTokenKeyID          string        `mapstructure:"REFRESH_TOKEN_KEY_ID"`
	RefreshTokenSecret         string        `mapstructure:"REFRESH_TOKEN_SECRET"`
	RefreshTokenPreviousKeyID  string        `mapstructure:"REFRESH_TOKEN_PREVIOUS_KEY_ID"`
	RefreshTokenPreviousSecret string        `mapstructure:"REFRESH_TOKEN_PREVIOUS_SECRET"`
}
```

Defaults:

```go
viper.SetDefault("FRONTEND_URL", "http://localhost:3000/comics")
viper.SetDefault("OAUTH_STATE_TTL", 10*time.Minute)
viper.SetDefault("ACCESS_TOKEN_KEY_ID", "access-2026-10")
viper.SetDefault("REFRESH_TOKEN_KEY_ID", "refresh-2026-10")
viper.SetDefault("ACCESS_TOKEN_EXPIRY_HOUR", 15*time.Minute)
viper.SetDefault("REFRESH_TOKEN_EXPIRY_HOUR", 168*time.Hour)
```

Bind all six key ID/previous-secret variables, `FRONTEND_URL`, and `OAUTH_STATE_TTL`. Preserve `applyLegacyJWTEnvAliases()` for one release, but log only the legacy variable name when it is used. Production validation requires 32-byte distinct current secrets; previous ID and secret must be both set or both blank; previous IDs must differ from current IDs; OAuth state TTL must be from 5 through 15 minutes; `FRONTEND_URL` must be an absolute HTTP(S) URL whose origin appears exactly in `CORS_ALLOWED_ORIGINS`; and Google client ID/secret must be both set or both blank.

Update `go_server/local.env` with:

```dotenv
FRONTEND_URL=https://localhost:3000/comics
OAUTH_STATE_TTL=10m
ACCESS_TOKEN_EXPIRY_HOUR=15m
REFRESH_TOKEN_EXPIRY_HOUR=168h
ACCESS_TOKEN_KEY_ID=access-2026-10
ACCESS_TOKEN_SECRET=dev-access-0123456789abcdef0123456789abcdef
ACCESS_TOKEN_PREVIOUS_KEY_ID=
ACCESS_TOKEN_PREVIOUS_SECRET=
REFRESH_TOKEN_KEY_ID=refresh-2026-10
REFRESH_TOKEN_SECRET=dev-refresh-abcdef0123456789abcdef0123456789
REFRESH_TOKEN_PREVIOUS_KEY_ID=
REFRESH_TOKEN_PREVIOUS_SECRET=
```

- [ ] **Step 7: Run focused and full package tests**

Run:

```bash
go test ./internal/tokenutil ./bootstrap -count=1
go test ./... -count=1
```

Expected: focused and full Go suites print `ok` or `[no test files]`; token tests prove strict claim/type checking, bootstrap tests prove production secret validation/redaction, and the compatibility test keeps existing callers compiling until Task 4.

- [ ] **Step 8: Commit the token and configuration foundation**

```bash
git add go_server/internal/tokenutil/jwt.go go_server/internal/tokenutil/jwt_test.go go_server/bootstrap/env.go go_server/bootstrap/env_test.go go_server/bootstrap/sanitize_test.go go_server/local.env
git commit -m "feat(auth): define typed JWT claims and key rotation"
```

---

### Task 2: Mongo Credential Preflight and Unique Indexes

**Files:**
- Modify: `go_server/internal/repo/db.go:8-36`
- Modify: `go_server/internal/repo/mongo/collection.go:1-84`
- Modify: `go_server/internal/repo/mongo/user_repo.go:33-289`
- Create: `go_server/internal/repo/mongo/auth_indexes.go`
- Modify: `go_server/internal/repo/mongo/user_repo_test.go:1-309`
- Create: `go_server/cmd/auth_migrate/main.go`
- Modify: `go_server/bootstrap/db.go:19-38`
- Modify: `go_server/bootstrap/app.go:21-84`
- Modify: `go_server/Makefile:100-140`

**Interfaces:**
- Produces: `(*UserRepo).AuditAuthIndexes(context.Context) (AuthIndexAudit, error)`.
- Produces: `(*UserRepo).EnsureAuthIndexes(context.Context) error`.
- Produces: `(*UserRepo).DropAuthIndexes(context.Context) error`.
- Produces: named indexes `users_email_unique_ci`, `users_username_unique_ci`, `auth_sessions_refresh_id_unique`, `auth_sessions_expires_ttl`, `auth_sessions_user_active`, and `oauth_states_expires_ttl`.
- Consumes: Task 1 environment validation and existing `UserRepo`.

- [ ] **Step 1: Add integration tests for preflight, indexes, and write races**

Extend the integration-tagged Mongo suite with fresh database names per test. Cover:

```go
func TestEnsureAuthIndexesRejectsCaseInsensitiveEmailDuplicates(t *testing.T)
func TestEnsureAuthIndexesRejectsCaseInsensitiveUsernameDuplicates(t *testing.T)
func TestEnsureAuthIndexesRejectsMissingEmail(t *testing.T)
func TestEnsureAuthIndexesAllowsMultipleEmptyUsernames(t *testing.T)
func TestEnsureAuthIndexesCreatesAllNamedIndexes(t *testing.T)
func TestUserRepoCreateMapsDuplicateKeyToConflict(t *testing.T)
func TestUserRepoUpdateMapsDuplicateKeyToConflict(t *testing.T)
```

The duplicate audit JSON must be deterministic:

```go
type CredentialDuplicate struct {
	Field   string      `json:"field"`
	Value   string      `json:"value"`
	UserIDs []uuid.UUID `json:"user_ids"`
}

type AuthIndexAudit struct {
	Duplicates      []CredentialDuplicate `json:"duplicates"`
	MissingEmailIDs []uuid.UUID           `json:"missing_email_ids"`
}

func (a AuthIndexAudit) Clean() bool {
	return len(a.Duplicates) == 0 && len(a.MissingEmailIDs) == 0
}
```

- [ ] **Step 2: Run the index tests and verify they fail**

Run:

```bash
cd go_server
go test -tags=integration ./internal/repo/mongo -run 'Test(EnsureAuthIndexes|UserRepo.*DuplicateKey)' -count=1
```

Expected: `FAIL` with missing `EnsureAuthIndexes`/`AuditAuthIndexes`, or duplicate writes succeeding because no unique index exists.

- [ ] **Step 3: Add repository configuration and Mongo adapters**

Add `repo.ErrConflict = errors.New("record conflicts with existing data")`, plus:

```go
TableAuthSessions string `mapstructure:"DB_TABLE_AUTH_SESSIONS" default:"auth_sessions"`
TableOAuthStates  string `mapstructure:"DB_TABLE_OAUTH_STATES" default:"oauth_states"`
```

Set and bind defaults in `bootstrap/env.go`:

```go
viper.SetDefault("DB_TABLE_AUTH_SESSIONS", "auth_sessions")
viper.SetDefault("DB_TABLE_OAUTH_STATES", "oauth_states")
```

Extend `Collection` and `mongoCollection` with the exact v2 driver signatures required by Tasks 2 and 3:

```go
Aggregate(context.Context, any, ...options.Lister[options.AggregateOptions]) (*mongo.Cursor, error)
UpdateOne(context.Context, any, any, ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error)
FindOneAndUpdate(context.Context, any, any, ...options.Lister[options.FindOneAndUpdateOptions]) *mongo.SingleResult
FindOneAndDelete(context.Context, any, ...options.Lister[options.FindOneAndDeleteOptions]) *mongo.SingleResult
```

Add `sessions` and `oauthStates` collection fields to `UserRepo` and initialize them from the new DB config names. Add `repo.ErrConflict` and `mongo.IsDuplicateKeyError(err)` to the permanent/no-retry path. Map duplicate keys from both `Create` and `Update` to `repo.ErrConflict`.

- [ ] **Step 4: Implement deterministic preflight and named indexes**

In `auth_indexes.go`, run two aggregation pipelines with `options.Aggregate().SetCollation(&options.Collation{Locale: "en", Strength: 2})`:

```go
func duplicatePipeline(field string, excludeBlank bool) mongo.Pipeline {
	match := bson.D{{Key: field, Value: bson.D{{Key: "$type", Value: "string"}}}}
	if excludeBlank {
		match = bson.D{{Key: field, Value: bson.D{
			{Key: "$type", Value: "string"},
			{Key: "$gt", Value: ""},
		}}}
	}
	return mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$" + field},
			{Key: "user_ids", Value: bson.D{{Key: "$push", Value: "$_id"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$match", Value: bson.D{{Key: "count", Value: bson.D{{Key: "$gt", Value: 1}}}}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
}
```

Audit email with `excludeBlank=false`, username with `excludeBlank=true`, and separately find users whose email is missing, non-string, or empty. Sort every UUID list before returning. `EnsureAuthIndexes` must call the audit first and return an error that includes only counts and the command `make go-auth-migrate ARGS=audit`; do not emit PII in startup logs.

Create these models:

```go
emailIndex := mongo.IndexModel{
	Keys: bson.D{{Key: "email", Value: 1}},
	Options: options.Index().
		SetName("users_email_unique_ci").
		SetUnique(true).
		SetCollation(&options.Collation{Locale: "en", Strength: 2}),
}

usernameIndex := mongo.IndexModel{
	Keys: bson.D{{Key: "username", Value: 1}},
	Options: options.Index().
		SetName("users_username_unique_ci").
		SetUnique(true).
		SetCollation(&options.Collation{Locale: "en", Strength: 2}).
		SetPartialFilterExpression(bson.D{{Key: "username", Value: bson.D{
			{Key: "$type", Value: "string"},
			{Key: "$gt", Value: ""},
		}}}),
}
```

Task 3 will use the session/state models; define all six named indexes now so one migration owns the complete auth schema. TTL models use `SetExpireAfterSeconds(0)` on `expires_at`; `auth_sessions_refresh_id_unique` is unique on `current_refresh_id`; `auth_sessions_user_active` is `{user_id: 1, revoked_at: 1}`.

- [ ] **Step 5: Make lookups index-compatible and canonicalize repository errors**

Use `options.FindOne().SetCollation(&options.Collation{Locale: "en", Strength: 2})` in `GetByEmail` and `GetByUsername`. Update `mapUserRepoError` in `internal/usecase/user.go`:

```go
if errors.Is(err, repo.ErrConflict) {
	return domain.ErrCredentialsAlreadyExist
}
```

This preserves preflight-friendly `409` behavior while the unique index becomes the final concurrency authority.

- [ ] **Step 6: Add the auth migration command and startup gate**

`go_server/cmd/auth_migrate/main.go` accepts exactly one operation:

```text
audit
apply
rollback-indexes --confirm
```

Behavior:

- `audit`: print indented `AuthIndexAudit` JSON; exit 0 when clean and 2 when duplicates/missing emails exist.
- `apply`: audit, refuse dirty data with exit 2, then create all named indexes and print `auth indexes applied`.
- `rollback-indexes --confirm`: drop only the six named indexes; preserve users, sessions, and OAuth state documents; print `auth indexes removed`.
- Any other invocation: print `usage: auth_migrate audit|apply|rollback-indexes --confirm` to stderr and exit 64.

The command constructs `UserRepo` from `bootstrap.MustLoadEnv`, pings it, defers close, and never starts HTTP or scraping.

Add:

```make
## @database:go-auth-migrate Audit/apply auth Mongo indexes (ARGS="audit" by default)
go-auth-migrate:
	@go run ./cmd/auth_migrate $(if $(strip $(ARGS)),$(ARGS),audit)
```

Extend the bootstrap repository interface with `EnsureAuthIndexes(context.Context) error`. In `MustLoadApp`, call it after `Ping` and before initializing the catalog store. A dirty production database must fail before serving traffic.

- [ ] **Step 7: Run integration, unit, and migration usage tests**

Run:

```bash
go test -tags=integration ./internal/repo/mongo -run 'Test(EnsureAuthIndexes|UserRepo.*DuplicateKey)' -count=1
go test ./internal/usecase ./bootstrap -count=1
go run ./cmd/auth_migrate
```

Expected:

- Integration and unit packages print `ok`.
- The command without an operation exits 64 and prints the exact usage string.

- [ ] **Step 8: Commit the Mongo uniqueness boundary**

```bash
git add go_server/internal/repo/db.go go_server/internal/repo/mongo/collection.go go_server/internal/repo/mongo/user_repo.go go_server/internal/repo/mongo/auth_indexes.go go_server/internal/repo/mongo/user_repo_test.go go_server/cmd/auth_migrate/main.go go_server/bootstrap/env.go go_server/bootstrap/db.go go_server/bootstrap/app.go go_server/Makefile
git commit -m "feat(auth): enforce Mongo credential indexes"
```

---

### Task 3: Mongo Refresh Sessions and OAuth State

**Files:**
- Create: `go_server/domain/auth_session.go`
- Modify: `go_server/domain/user.go:27-52`
- Create: `go_server/internal/repo/mongo/auth_state.go`
- Modify: `go_server/internal/repo/mongo/user_repo.go:37-92`
- Modify: `go_server/internal/repo/mongo/user_repo_test.go`
- Modify: `go_server/bootstrap/app.go:21-84`

**Interfaces:**
- Produces: `domain.AuthSessionStore`, `domain.OAuthStateStore`, and `domain.AuthStore` exactly as defined in the shared interface block.
- Produces: atomic session rotation and one-time state consumption for Tasks 4 and 6.
- Consumes: Task 2 collection adapters and named TTL/uniqueness indexes.

- [ ] **Step 1: Write integration tests for all auth-state transitions**

Add these integration tests:

```go
func TestAuthSessionLifecycle(t *testing.T)
func TestRotateSessionRejectsWrongCurrentTokenAndRevokesOnReplay(t *testing.T)
func TestRotateSessionRejectsExpiredOrRevokedSession(t *testing.T)
func TestRevokeSessionIsIdempotent(t *testing.T)
func TestOAuthStateConsumeIsOneTime(t *testing.T)
func TestOAuthStateConsumeRejectsWrongBrowserNonce(t *testing.T)
func TestOAuthStateConsumeRejectsExpiredState(t *testing.T)
```

Assert the stored session never contains a token string, only `current_refresh_id`. For replay, first rotate `jti-1` to `jti-2`, then attempt `jti-1` again; the second call returns `domain.ErrSessionReplay` and the session becomes revoked.

- [ ] **Step 2: Run the focused integration tests and verify missing methods**

Run:

```bash
cd go_server
go test -tags=integration ./internal/repo/mongo -run 'Test(AuthSession|RotateSession|RevokeSession|OAuthState)' -count=1
```

Expected: `FAIL` with missing domain/store types and repository methods.

- [ ] **Step 3: Add the domain state model**

Create `go_server/domain/auth_session.go` with the complete shared interface block at the top of this plan. Add the concrete compile-time assertion beside the existing user-store assertion in `user_repo.go`:

```go
var _ domain.AuthStore = (*UserRepo)(nil)
```

Do not add raw token, role, user agent, or IP fields. They are not needed to enforce rotation and would add retention/privacy obligations.

- [ ] **Step 4: Implement atomic session persistence**

In `auth_state.go`:

```go
func (r *UserRepo) RotateSession(
	ctx context.Context,
	sessionID uuid.UUID,
	expectedRefreshID string,
	nextRefreshID string,
	now time.Time,
) error {
	filter := bson.D{
		{Key: "_id", Value: sessionID},
		{Key: "current_refresh_id", Value: expectedRefreshID},
		{Key: "revoked_at", Value: bson.D{{Key: "$exists", Value: false}}},
		{Key: "expires_at", Value: bson.D{{Key: "$gt", Value: now}}},
	}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "current_refresh_id", Value: nextRefreshID},
		{Key: "updated_at", Value: now},
	}}}
	err := r.sessions.FindOneAndUpdate(ctx, filter, update).Err()
	if err == nil {
		return nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}
	if revokeErr := r.RevokeSession(ctx, sessionID, now); revokeErr != nil {
		return errors.Join(domain.ErrSessionReplay, revokeErr)
	}
	return domain.ErrSessionReplay
}
```

`CreateSession` validates non-nil IDs, non-empty current JTI, and `ExpiresAt.After(CreatedAt)`. `RevokeSession` sets `revoked_at`/`updated_at` only when not already revoked and returns nil when no record matches, making logout idempotent.

- [ ] **Step 5: Implement hashed, one-time OAuth state**

`CreateOAuthState` validates a 64-character lowercase SHA-256 hex state hash, a 64-character browser nonce hash, non-empty verifier, and future expiry before insertion.

`ConsumeOAuthState` uses `FindOneAndDelete` with all three constraints:

```go
filter := bson.D{
	{Key: "_id", Value: stateHash},
	{Key: "browser_nonce_hash", Value: browserNonceHash},
	{Key: "expires_at", Value: bson.D{{Key: "$gt", Value: now}}},
}
```

Map `mongo.ErrNoDocuments` to `domain.ErrOAuthState`. Because deletion and return are one atomic Mongo operation, callback replay cannot reuse the verifier.

- [ ] **Step 6: Promote the bootstrap store to the aggregate auth contract**

Change the bootstrap interface and field names:

```go
type ClosableAuthStore interface {
	Closable
	domain.AuthStore
	EnsureAuthIndexes(context.Context) error
}

type Application struct {
	Env          *Env
	AuthStore    ClosableAuthStore
	ComicService domain.ComicUseCase
	ScrapeCoord  *scrape.Coordinator
	Shutters     []func(context.Context) error
}
```

Update `newRepo` to return `ClosableAuthStore`, return the same Mongo `UserRepo`, and update `cmd/server/main.go`/route wiring references from `app.UserRepo` to `app.AuthStore`. This is one Mongo client and one modular-monolith repository object, not an additional service.

- [ ] **Step 7: Run focused and complete repository tests**

Run:

```bash
go test -tags=integration ./internal/repo/mongo -run 'Test(AuthSession|RotateSession|RevokeSession|OAuthState)' -count=1
go test ./domain ./bootstrap ./cmd/server -count=1
```

Expected: all listed packages print `ok` or `[no test files]`; race/replay tests prove only one refresh rotation succeeds.

- [ ] **Step 8: Commit auth-state persistence**

```bash
git add go_server/domain/auth_session.go go_server/domain/user.go go_server/internal/repo/mongo/auth_state.go go_server/internal/repo/mongo/user_repo.go go_server/internal/repo/mongo/user_repo_test.go go_server/bootstrap/app.go go_server/bootstrap/db.go go_server/cmd/server/main.go
git commit -m "feat(auth): persist refresh sessions and OAuth state"
```

---

### Task 4: Active-User Sessions and HTTP Security Boundary

**Files:**
- Modify: `go_server/domain/auth.go:10-83`
- Modify: `go_server/domain/user.go:15-52`
- Modify: `go_server/internal/usecase/user.go:1-153`
- Create: `go_server/internal/usecase/user_test.go`
- Modify: `go_server/internal/usecase/auth.go:1-244`
- Modify: `go_server/internal/usecase/auth_test.go:1-113`
- Modify: `go_server/api/controller/auth.go:1-60`
- Modify: `go_server/api/middleware/auth.go:1-104`
- Modify: `go_server/api/middleware/auth_test.go:1-160`
- Create: `go_server/api/route/cookies.go`
- Create: `go_server/api/route/cookies_test.go`
- Modify: `go_server/api/route/route.go:27-333`
- Modify: `go_server/api/route/auth_route_test.go:1-222`
- Modify: `go_server/api/route/user_profile.go:28-99`
- Modify: `go_server/cmd/server/main.go:45-56`
- Modify: `go_server/internal/tokenutil/jwt.go`
- Modify: `go_server/internal/tokenutil/jwt_test.go`

**Interfaces:**
- Consumes: Task 1 `tokenutil.Manager` and Task 3 auth stores.
- Produces: `domain.AccessAuthenticator` for middleware.
- Produces: `middleware.BearerAuthenticationMiddleware(domain.AccessAuthenticator) gin.HandlerFunc` as the single access-token authenticator for every protected API group.
- Produces: Gin context `middleware.KeyUserID` as `uuid.UUID`, `middleware.KeyUser` as `*domain.User`, and `middleware.KeyRole` from the current Mongo user.
- Produces: idempotent `POST /logout`; `POST /refresh-token` no longer accepts `Role`.
- Produces: use-case methods with no browser-supplied role:

```go
type AccessAuthenticator interface {
	AuthenticateAccessToken(context.Context, string) (*User, error)
}

type AuthUseCase interface {
	AccessAuthenticator
	GetAccessTokenExpirySeconds() int
	GetRefreshTokenExpirySeconds() int
	Login(context.Context, LoginRequest) (*AuthResponse, error)
	LoginByOAuth(context.Context, OAuthIdentity) (*AuthResponse, error)
	UpdateProfile(context.Context, *User, UpdateRequest) (*AuthResponse, error)
	Register(context.Context, SignUpRequest) (*AuthResponse, error)
	RefreshToken(context.Context, string) (*AuthResponse, error)
	Logout(context.Context, string) error
}
```

- [ ] **Step 1: Write user-service tests for canonicalization and active accounts**

Create a fake `domain.UserStore` that records arguments. Test:

- login lowercases/trims email before lookup;
- password login returns `domain.ErrInvalidCredentials` for inactive users without revealing account status;
- registration lowercases/trims email and trims username;
- update does the same;
- `repo.ErrConflict` maps to `domain.ErrCredentialsAlreadyExist`;
- OAuth registration creates an active `user` with no password and a deterministic non-empty username.

Use this exact OAuth identity:

```go
domain.OAuthIdentity{
	Provider:      "google",
	Subject:       "google-subject-123",
	Email:         " Reader@Example.COM ",
	EmailVerified: true,
	DisplayName:   "Reader",
}
```

The generated username must be `reader-7efafae7feee`, where the suffix is the first twelve lowercase hex characters of SHA-256 over `google:google-subject-123`. Compute and assert the expected value from the helper in the test so a collision-resistant deterministic suffix, not the display name, defines uniqueness.

- [ ] **Step 2: Write auth-service tests for the session state machine**

Replace the current role-mismatch refresh test with:

```go
func TestAuthServiceLoginCreatesSession(t *testing.T)
func TestAuthServiceRejectsInactivePasswordAndOAuthUsers(t *testing.T)
func TestAuthenticateAccessTokenReloadsActiveUser(t *testing.T)
func TestAuthenticateAccessTokenRejectsInactiveUser(t *testing.T)
func TestRefreshRotatesJTIAndUsesCurrentDatabaseUser(t *testing.T)
func TestRefreshReplayRevokesSession(t *testing.T)
func TestRefreshRejectsInactiveUserAndRevokesSession(t *testing.T)
func TestLogoutRevokesNewSession(t *testing.T)
func TestLogoutAcceptsMissingMalformedAndLegacyToken(t *testing.T)
```

Use a fake clock, fake session store, and Task 1 manager. Assert refresh returns a different refresh token, old refresh replay returns 401, and role changes do not depend on any request header or token field.

- [ ] **Step 3: Run use-case tests and verify old signatures fail**

Run:

```bash
cd go_server
go test ./internal/usecase -run 'Test(AuthService|UserService)' -count=1
```

Expected: `FAIL` because `Login` still accepts an access token, `RefreshToken` still accepts a role, and session methods are not wired.

- [ ] **Step 4: Harden the user service**

Add:

```go
func canonicalEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func canonicalUsername(value string) string {
	return strings.TrimSpace(value)
}
```

Apply them before every email/username lookup and write. After password comparison, reject inactive users as invalid credentials. Add `RegisterOAuth(context.Context, OAuthIdentity)` to `UserServicer`; it rejects unverified/blank email, reuses `Register`-style UUID creation, stores no password hash, and generates a username by joining the canonical email local part, a hyphen, and the first 12 SHA-256 hex characters of `provider:subject`. Replace the current `errgroup` mutations in update/existence checks with sequential lookups: propagate datastore errors other than `ErrUserNotFound`, then mutate the user only after all checks pass. Keep prechecks for friendly errors, but rely on Task 2 unique indexes for race safety.

- [ ] **Step 5: Replace auth-service construction and issuance**

Use:

```go
type AuthService struct {
	users    domain.UserServicer
	sessions domain.AuthSessionStore
	tokens   *tokenutil.Manager
	now      func() time.Time
}

func NewAuthService(
	users domain.UserServicer,
	sessions domain.AuthSessionStore,
	tokens *tokenutil.Manager,
	now func() time.Time,
) *AuthService {
	if now == nil {
		now = time.Now
	}
	return &AuthService{users: users, sessions: sessions, tokens: tokens, now: now}
}
```

Centralize login/register/OAuth issuance:

```go
func (s *AuthService) startSession(ctx context.Context, user *domain.User) (*domain.AuthData, error) {
	if user == nil || !user.Active {
		return nil, domain.ErrInvalidCredentials
	}
	now := s.now().UTC()
	sessionID := uuid.New()
	refreshID := uuid.NewString()
	expiresAt := now.Add(s.tokens.RefreshTTL())
	session := &domain.AuthSession{
		ID:               sessionID,
		UserID:           user.ID,
		CurrentRefreshID: refreshID,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        expiresAt,
	}
	if err := s.sessions.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create auth session: %w", err)
	}
	access, refresh, err := s.tokens.NewSessionTokens(user.ID, sessionID, refreshID, expiresAt)
	if err != nil {
		_ = s.sessions.RevokeSession(ctx, sessionID, now)
		return nil, fmt.Errorf("issue auth tokens: %w", err)
	}
	return &domain.AuthData{
		UserID:       user.ID.String(),
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}
```

Add `RefreshTTL() time.Duration` to `tokenutil.Manager`. Keep `AuthData.RefreshToken` for route-to-cookie transfer but change its tag to `json:"-"`. Remove the “login with an existing access token” shortcut.

- [ ] **Step 6: Implement access authentication, rotation, and logout**

`AuthenticateAccessToken` trims an optional `Bearer ` prefix, verifies access type, extracts `sub`, loads the user by ID, and returns unauthorized when missing/inactive.

`UpdateProfile` accepts the already-authenticated `*domain.User`, rejects nil/inactive users, and delegates canonicalized field changes to `UserServicer.Update`; it never verifies the bearer token a second time.

Refresh order is fixed:

1. verify refresh type/signature/claims;
2. reject missing session ID;
3. load the user and revoke the session if absent/inactive;
4. generate a next refresh JTI;
5. atomically call `RotateSession(sessionID, oldJTI, nextJTI, now)`;
6. issue access + refresh with the existing absolute `exp`;
7. return both through `AuthData`, with refresh omitted from JSON.

If rotation returns `ErrSessionReplay`, return 401 and never issue a token. `Logout` verifies a refresh token when possible, revokes by `sid`, and treats missing/malformed/new-claim-incompatible tokens as already logged out. Datastore failures after a valid token must be returned so the route can report 503 after clearing cookies.

- [ ] **Step 7: Make OAuth identity and public errors explicit**

Add:

```go
type OAuthIdentity struct {
	Provider      string
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
}
```

`LoginByOAuth` must reject unverified email before any lookup, load an existing user by canonical email, register through `RegisterOAuth` only on `ErrUserNotFound`, reject inactive users, and start a session. Public auth responses remain generic:

- bad password/inactive account: `401 invalid credentials`;
- invalid/replayed/old-format refresh: `401 session expired; sign in again`;
- duplicate signup/update: `409 already exist`;
- auth persistence failure: `503 authentication service unavailable`.

- [ ] **Step 8: Update the controller adapter and run tests**

Remove role/access-token shortcut parameters:

```go
func (ac *AuthControl) Login(ctx context.Context, user domain.LoginRequest) (*domain.AuthResponse, error)
func (ac *AuthControl) LoginByOAuth(ctx context.Context, identity domain.OAuthIdentity) (*domain.AuthResponse, error)
func (ac *AuthControl) UpdateProfile(ctx context.Context, user *domain.User, update domain.UpdateRequest) (*domain.AuthResponse, error)
func (ac *AuthControl) RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error)
func (ac *AuthControl) Logout(ctx context.Context, refreshToken string) error
func (ac *AuthControl) AuthenticateAccessToken(ctx context.Context, token string) (*domain.User, error)
```

Keep only one temporary controller adapter so the existing OAuth route compiles until Task 5:

```go
func (ac *AuthControl) LoginByOAuthEmail(ctx context.Context, email string) (*domain.AuthResponse, error) {
	return ac.auth.LoginByOAuth(ctx, domain.OAuthIdentity{
		Provider:      "google",
		Subject:       strings.ToLower(strings.TrimSpace(email)),
		Email:         email,
		EmailVerified: true,
	})
}
```

Task 5 must delete this adapter when the callback begins passing the verified provider identity itself. Do not retain compatibility overloads for login, refresh, profile update, or role.

Run:

```bash
go test ./internal/usecase ./api/controller -count=1
```

Expected: `ok` for use cases and `[no test files]` or `ok` for controller; active-user and replay tests pass.

- [ ] **Step 9: Rewrite middleware tests around an authenticator fake**

The fake returns a `domain.User`, not claims. Test:

- bearer extraction with no access-cookie fallback;
- missing/malformed tokens;
- authenticator failure;
- inactive-user failure from the authenticator;
- `KeyUser`, `KeyUserID`, and `KeyRole` equal the returned Mongo user;
- admin role middleware allows only context role;
- a token string that visually contains `admin` cannot affect the role.

The constructor is:

```go
func BearerAuthenticationMiddleware(auth domain.AccessAuthenticator) gin.HandlerFunc
```

- [ ] **Step 10: Add route tests for cookie and origin policy**

In `cookies_test.go` and `auth_route_test.go`, assert:

- production refresh cookie is `HttpOnly`, `Secure`, `SameSite=None`, `Path=/`, with configured max age;
- development refresh cookie is `HttpOnly`, not secure on HTTP, and `SameSite=Lax`;
- OAuth nonce cookie is always `SameSite=Lax`;
- logout expires `refresh_token`, `access_token`, and `oauth_session` with `Max-Age=0` or a past expiry;
- refresh reads canonical cookie first, then `Authorization: Bearer` for non-browser clients;
- refresh ignores/removes the old `Role` header contract;
- allowed browser origin + `X-CSRF-Protection: 1` succeeds;
- disallowed origin returns 403 before the handler;
- missing CSRF header on credential-bearing browser POST returns 403;
- CLI POST with no `Origin` is allowed;
- logout returns 204 for no cookie, malformed token, and repeated calls;
- login/signup/refresh set only the refresh cookie and return access token JSON without `refresh_token`.

- [ ] **Step 11: Run middleware/route tests and verify old constructors fail**

Run:

```bash
cd go_server
go test ./api/middleware ./api/route -run 'Test(BearerAuthenticationMiddleware|RoleMiddleware|AuthCookie|OriginGuard|AuthRefresh|AuthLogout)' -count=1
```

Expected: `FAIL` because middleware still accepts a secret string, cookies use Gin defaults, and `/logout` does not exist.

- [ ] **Step 12: Make authentication middleware datastore-aware**

Replace token parsing in middleware:

```go
func BearerAuthenticationMiddleware(auth domain.AccessAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(KeyAuthorization)
		if token == "" {
			abortUnauthorized(c, "Missing authentication token")
			return
		}
		if !strings.HasPrefix(token, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(token, "Bearer ")) == "" {
			abortUnauthorized(c, "Invalid structure token")
			return
		}
		user, err := auth.AuthenticateAccessToken(c.Request.Context(), token)
		if err != nil || user == nil || !user.Active {
			abortUnauthorized(c, "Invalid authentication token")
			return
		}
		c.Set(KeyUser, user)
		c.Set(KeyUserID, user.ID)
		c.Set(KeyRole, user.Role)
		c.Next()
	}
}
```

Delete `ExtractCookieAccessToken`; new protected requests require the bearer header. Keep the legacy `access_token` cookie name only in `clearAuthCookies`, so a one-time sign-in reset and logout remove it. Do not parse JWTs or trust claims in middleware.

- [ ] **Step 13: Centralize cookie policy**

Create:

```go
type authCookiePolicy struct {
	Secure          bool
	RefreshSameSite http.SameSite
}

func newAuthCookiePolicy(env *bootstrap.Env) authCookiePolicy {
	secure := env.AppEnv.IsProduction() || env.HasHTTPSTLSFiles()
	sameSite := http.SameSiteLaxMode
	if secure && env.AppEnv.IsProduction() {
		sameSite = http.SameSiteNoneMode
	}
	return authCookiePolicy{Secure: secure, RefreshSameSite: sameSite}
}
```

Use `http.SetCookie`, not `gin.Context.SetCookie`, so `SameSite` is explicit. Cookie names remain `refresh_token`, `access_token`, and `oauth_session`. `setRefreshCookie` sets only `refresh_token`; `clearAuthCookies` expires all three names using the same host-only domain and `/` path.

- [ ] **Step 14: Add exact-origin and CSRF protection**

Refactor parsing into one `allowedOriginSet` shared by CORS and:

```go
func browserMutationGuard(origins allowedOriginSet) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" {
			c.Next()
			return
		}
		if !origins.Contains(origin) || c.GetHeader("X-CSRF-Protection") != "1" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden origin"})
			return
		}
		c.Next()
	}
}
```

Add `X-CSRF-Protection` to `Access-Control-Allow-Headers`. Reject disallowed preflights with 403 instead of returning 204 without CORS headers. Preserve exact-origin reflection and `Vary: Origin`; never use `*` with credentials.

- [ ] **Step 15: Wire the token manager, refresh cookie, and logout routes**

Build the `tokenutil.Manager` in `route.Setup` from validated env key rings and return an error if construction fails. Change `Setup` to:

```go
func Setup(
	env *bootstrap.Env,
	authStore domain.AuthStore,
	comics domain.ComicUseCase,
	g *gin.Engine,
) (*health.Checker, error)
```

Construct one `UserService`, one `AuthService`, and one controller. Preserve
Workstream 1's public `catalogRead` group without middleware. Mount catalog
mutations and every scrape route on the existing `catalogAdmin` group with
`BearerAuthenticationMiddleware(authService)` followed by
`RoleMiddleware(tokenutil.RoleAdmin)`. `/protected` and dashboard groups also
use `BearerAuthenticationMiddleware(authService)`; the dashboard adds the
admin role middleware. This deliberately removes Workstream 1's temporary
access-cookie fallback while preserving all route paths. Update catalog,
profile, dashboard, and scrape policy tests to inject an authenticator fake
that returns active `user` or `admin` Mongo users rather than
`GenerateTokenWithRole` tokens. `cmd/server/main.go` handles setup failure
before serving:

```go
ready, err := route.Setup(app.Env, app.AuthStore, app.ComicService, g)
if err != nil {
	log.Fatal().Err(err).Msg("Failed to configure routes")
}
```

Pass `ready` unchanged into Workstream 4's lifecycle manager.

Login/signup call the simplified controller, set only the refresh cookie from `resp.Data.RefreshToken`, and serialize `AuthData` without that field.

Refresh token source order is cookie, then bearer header. It never reads `Role`. On success it rotates the refresh cookie and returns access token JSON. On failure it clears cookies and returns the use-case status.

Add:

```go
func logoutRouter(auth *controller.AuthControl, cookies authCookiePolicy, group *gin.RouterGroup) {
	group.POST("/logout", func(c *gin.Context) {
		refresh := extractRefreshToken(c)
		clearAuthCookies(c, cookies)
		if err := auth.Logout(c.Request.Context(), refresh); err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
```

Apply `browserMutationGuard` to POST `/signup`, `/login`, `/refresh-token`,
and `/logout`, and to the existing `catalogAdmin` group before its
authentication/role middleware. The safe-method branch leaves scrape
status/history GETs unchanged, while catalog mutations and `POST /scrape`
require the CSRF header for browser origins. Add
`middleware.KeyUser = "user"` and set the authenticated `*domain.User` in
middleware. `getProfile` returns that context user; `putProfile` passes it to
`AuthControl.UpdateProfile`. Neither profile handler reparses a token or
performs a second lookup.

- [ ] **Step 16: Run route policy and full Go unit tests**

Run:

```bash
go test ./api/middleware ./api/route ./cmd/server -count=1
go test ./... -count=1
```

Expected: all packages print `ok` or `[no test files]`; auth route tests prove the refresh JSON has no token and logout is idempotent.

After every use-case and middleware caller uses `tokenutil.Manager`, delete `LegacyClaims` and the five compatibility token functions retained in Task 1. Preserve only `RoleAdmin` and `RoleUser` until a separate role-constant cleanup moves all callers.

- [ ] **Step 17: Commit the complete auth/HTTP security boundary**

```bash
git add go_server/domain/auth.go go_server/domain/user.go go_server/internal/usecase/user.go go_server/internal/usecase/user_test.go go_server/internal/usecase/auth.go go_server/internal/usecase/auth_test.go go_server/api/controller/auth.go go_server/api/middleware/auth.go go_server/api/middleware/auth_test.go go_server/api/route/cookies.go go_server/api/route/cookies_test.go go_server/api/route/route.go go_server/api/route/auth_route_test.go go_server/api/route/user_profile.go go_server/cmd/server/main.go go_server/internal/tokenutil/jwt.go go_server/internal/tokenutil/jwt_test.go
git commit -m "feat(auth): rotate and enforce browser sessions"
```

---

### Task 5: Per-Session Google OAuth State and Verified Email

**Files:**
- Modify: `go_server/api/route/oauth2.go:1-135`
- Create: `go_server/api/route/oauth2_test.go`
- Modify: `go_server/api/route/route.go:72-82`
- Modify: `go_server/api/controller/auth.go`
- Modify: `go_server/domain/auth.go`
- Modify: `go_server/internal/usecase/auth_test.go`

**Interfaces:**
- Consumes: Task 3 `OAuthStateStore` and Task 4 `LoginByOAuth`/cookie policy.
- Produces: `oauthHandler` with injectable OAuth endpoints/user-info URL/clock for deterministic tests.

- [ ] **Step 1: Write route tests before replacing global OAuth state**

Use fake `OAuthStateStore`, fake auth use case, and `httptest.Server` OAuth token/user-info endpoints. Cover:

```go
func TestGoogleStartCreatesUniqueStateNonceAndPKCE(t *testing.T)
func TestGoogleCallbackRejectsMissingMismatchedExpiredAndReplayedState(t *testing.T)
func TestGoogleCallbackSendsPKCEVerifierToExchange(t *testing.T)
func TestGoogleCallbackUsesAuthenticatedUserInfoClient(t *testing.T)
func TestGoogleCallbackRejectsNon200AndOversizedUserInfo(t *testing.T)
func TestGoogleCallbackRejectsUnverifiedOrBlankEmail(t *testing.T)
func TestGoogleCallbackLogsInVerifiedEmailAndSetsRefreshCookie(t *testing.T)
```

For two start requests, assert different query `state`, different `oauth_session` cookies, and stored hashes—not raw values. For a valid callback assert the use case receives:

```go
domain.OAuthIdentity{
	Provider:      "google",
	Subject:       "google-user-123",
	Email:         "reader@example.com",
	EmailVerified: true,
	DisplayName:   "Reader Example",
}
```

- [ ] **Step 2: Run OAuth route tests and verify the global-state flow fails**

Run:

```bash
cd go_server
go test ./api/route -run 'TestGoogle(Start|Callback)' -count=1
```

Expected: `FAIL`; current requests reuse one process-global state, omit PKCE, accept `verified_email=false`, and put the OAuth token in a query string.

- [ ] **Step 3: Replace mutable package globals with an injected handler**

Use:

```go
type oauthHandler struct {
	config      *oauth2.Config
	states      domain.OAuthStateStore
	auth        *controller.AuthControl
	cookies     authCookiePolicy
	frontendURL string
	userInfoURL string
	stateTTL    time.Duration
	now         func() time.Time
	randomBytes func(int) ([]byte, error)
}
```

`newOAuthHandler` copies, rather than mutates, an `oauth2.Config` built from env. Keep the Google scopes and endpoint. Build redirect URL from `env.ServerScheme() + "://" + env.HostURL + "/auth/callback"`. The production platform terminates TLS, so `ServerScheme()` must return `https` for production even when the in-process listener is HTTP; add a focused bootstrap test and update that method accordingly.

- [ ] **Step 4: Implement one-time state, browser nonce, and PKCE**

On `/auth/google`:

1. generate 32 random bytes each for state and browser nonce with `crypto/rand`;
2. base64url-encode without padding;
3. call `oauth2.GenerateVerifier()`;
4. SHA-256 hash state and nonce to lowercase hex;
5. store `domain.OAuthState` with `CreatedAt=now`, `ExpiresAt=now+env.OAuthStateTTL`, and verifier;
6. set the nonce in `oauth_session` as HttpOnly, host-only, SameSite Lax, production Secure, path `/`, max-age equal to state TTL;
7. redirect to `config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))`.

Never log state, nonce, verifier, code, or provider token.

- [ ] **Step 5: Consume state before code exchange**

On callback, require non-empty query state/code and nonce cookie. Hash state and nonce, call:

```go
record, err := h.states.ConsumeOAuthState(ctx, stateHash, nonceHash, h.now().UTC())
```

Clear `oauth_session` immediately, whether consumption succeeds or fails. Exchange with:

```go
token, err := h.config.Exchange(ctx, code, oauth2.VerifierOption(record.PKCEVerifier))
```

An invalid/expired/replayed state redirects to:

```text
$FRONTEND_URL/login?oauth_error=state
```

The callback never falls back to a process-global value.

- [ ] **Step 6: Fetch and validate Google user info safely**

Use `h.config.Client(ctx, token).Do` with a context-bound GET to the fixed HTTPS user-info endpoint. Do not concatenate the access token into the URL. Require status 200, read with `io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))`, reject a body longer than `1<<20`, and decode:

```go
type googleDTO struct {
	Email         string `json:"email"`
	FamilyName    string `json:"family_name"`
	GivenName     string `json:"given_name"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}
```

Require `ID`, canonical email, and `VerifiedEmail`. Convert to `domain.OAuthIdentity`, call `LoginByOAuth`, set only the refresh cookie, and redirect success to `strings.TrimRight(h.frontendURL, "/") + "/"`. User-info/email/account failures redirect to `strings.TrimRight(h.frontendURL, "/") + "/login?oauth_error=identity"`; internal error detail stays in structured server logs without token material.

- [ ] **Step 7: Wire OAuth only when configured**

When both Google values are blank, register `/auth/google` and callback handlers that return `404 {"error":"Google OAuth is not configured"}` instead of creating an invalid provider URL. When both are present, inject the same auth store, controller, and cookie policy used by password auth.

Delete the temporary `LoginByOAuthEmail` controller adapter from Task 4. The route now constructs a complete `domain.OAuthIdentity` only after state, PKCE exchange, and `verified_email` checks succeed.

- [ ] **Step 8: Run OAuth, route, and bootstrap tests**

Run:

```bash
go test ./api/route ./internal/usecase ./bootstrap -run 'Test(Google|AuthService.*OAuth|ServerScheme)' -count=1
```

Expected: all selected tests print `PASS`; replayed state is rejected and unverified email never reaches the auth use case.

- [ ] **Step 9: Commit OAuth hardening**

```bash
git add go_server/api/route/oauth2.go go_server/api/route/oauth2_test.go go_server/api/route/route.go go_server/api/controller/auth.go go_server/domain/auth.go go_server/internal/usecase/auth_test.go go_server/bootstrap/env.go go_server/bootstrap/env_test.go
git commit -m "feat(auth): secure Google OAuth sessions"
```

---

### Task 6: Frontend Session Bootstrap, Trusted Role, and Logout

**Files:**
- Modify: `src/frontend/util/AuthHelpers.ts:1-109`
- Create: `src/frontend/util/AuthHelpers.test.ts`
- Modify: `src/frontend/context/AuthContext.tsx:1-99`
- Create: `src/frontend/context/AuthContext.test.tsx`
- Modify: `src/frontend/components/Auth/LoginPage.tsx:1-48`
- Modify: `src/frontend/components/Auth/AuthNav.tsx:1-27`
- Modify: `src/frontend/components/Auth/ProfilePage.tsx:1-80`

**Interfaces:**
- Consumes: Task 4 response/cookie/logout contract.
- Produces: `AuthContextValue.status`, `isAuthenticated`, and `isAdmin`, with role sourced only from profile JSON.
- Produces: one deduplicated cookie refresh request, safe under React Strict Mode and serialized across tabs with the Web Locks API when available.

- [ ] **Step 1: Write helper tests for the browser wire contract**

Mock `global.fetch` and assert:

- login/signup/refresh/logout use `credentials: "include"`;
- POSTs send `X-CSRF-Protection: 1`;
- refresh sends no `Role` header and no refresh token body;
- two simultaneous `refreshAuth()` calls result in one fetch and resolve to the same response;
- refresh requests run inside `navigator.locks.request("comics-auth-refresh", ...)` when Web Locks is available, preventing two tabs from replaying the same rotating cookie;
- environments without Web Locks retain same-tab module-promise deduplication;
- logout accepts 204 without trying to parse JSON;
- profile includes the bearer access token;
- `AuthUser.role` accepts only `"user" | "admin"` and includes `active`;
- `AuthData` has no refresh-token field.

The exact types are:

```ts
export type AuthRole = 'user' | 'admin';

export type AuthData = {
  user_id: string;
  access_token?: string;
};

export type AuthUser = {
  id: string;
  username: string;
  email: string;
  role: AuthRole;
  active: boolean;
};
```

- [ ] **Step 2: Write AuthContext tests before implementation**

Test:

- an access token in session storage loads profile and sets `status="authenticated"`;
- no access token triggers one cookie refresh, stores the returned access token, then loads profile;
- failed refresh sets `status="anonymous"` without a render-time unhandled rejection;
- an invalid stored access token is cleared, refreshed once, and profile is retried;
- signup stores the access token from the signup response and does not create a second session by calling login;
- `isAdmin` is true only when profile JSON role is `admin`;
- a token payload containing an `admin` string with profile role `user` leaves `isAdmin=false`;
- logout clears session storage and context synchronously before the mocked server logout resolves;
- React Strict Mode does not send two rotation requests.

- [ ] **Step 3: Run frontend auth tests and verify missing behavior**

Run:

```bash
npm test -- src/frontend/util/AuthHelpers.test.ts src/frontend/context/AuthContext.test.tsx
```

Expected: `FAIL`; current helper sends `Role: user`, refresh is not deduplicated, context does not bootstrap from the cookie, and logout never calls the server.

- [ ] **Step 4: Harden `AuthHelpers.ts`**

Keep one request helper and add JSON parsing that tolerates 204 and non-JSON errors. Add the CSRF header only for non-GET methods. Implement module-level refresh deduplication:

```ts
let refreshInFlight: Promise<AuthResponse> | null = null;

const performRefresh = (): Promise<AuthResponse> => (
  authFetch('/refresh-token', { method: 'POST' }).then(parseAuthResponse)
);

export const refreshAuth = (): Promise<AuthResponse> => {
  if (refreshInFlight) return refreshInFlight;
  const lockedRefresh = navigator.locks
    ? navigator.locks.request('comics-auth-refresh', performRefresh)
    : performRefresh();
  refreshInFlight = lockedRefresh
    .finally(() => {
      refreshInFlight = null;
    });
  return refreshInFlight;
};

export const logoutSession = async (): Promise<void> => {
  const response = await authFetch('/logout', { method: 'POST' });
  if (!response.ok && response.status !== 401) {
    throw new Error('Logout unavailable');
  }
};

export const googleLoginURL = (): string => `${SERVER}/auth/google`;
```

Delete the `Role` header and `refresh_token` type. Continue storing only `comics_access_token` in session storage.

- [ ] **Step 5: Implement deterministic context bootstrap**

Use:

```ts
export type AuthStatus = 'loading' | 'authenticated' | 'anonymous';

type AuthContextValue = {
  user: AuthUser | null;
  accessToken?: string;
  status: AuthStatus;
  isAuthenticated: boolean;
  isAdmin: boolean;
  login: (email: string, password: string) => Promise<string | null>;
  signup: (email: string, username: string, password: string) => Promise<string | null>;
  logout: () => Promise<void>;
  refreshProfile: () => Promise<void>;
};
```

Bootstrap algorithm:

1. mark loading;
2. try the stored access token and profile;
3. if missing/unauthorized, clear local token and call deduplicated `refreshAuth`;
4. store a returned access token and load profile once;
5. authenticate only if profile is active;
6. otherwise clear local state and mark anonymous.

Derive `isAdmin` from `user?.role === "admin"`. Never inspect JWT payloads. On logout, clear local state before awaiting `logoutSession`; swallow the network error because local logout must remain compatible/offline.

Signup follows the same response path as login: store `response.data.access_token`, load profile, and return. It must not call `login(email, password)`, because the signup response already created a refresh session.

- [ ] **Step 6: Update auth components**

In `LoginPage`, display a normal anchor:

```tsx
<a className="auth-link" href={googleLoginURL()}>
  Continue with Google
</a>
```

If `oauth_error` is present in `useSearchParams`, display `Google sign-in failed. Please try again.` without echoing the query value.

Update logout buttons:

```tsx
<button type="button" onClick={() => void logout()}>Sign out</button>
```

While `status === "loading"`, Profile renders `Loading profile…`; it must not flash a sign-in prompt before cookie refresh completes.

- [ ] **Step 7: Run focused and full frontend checks**

Run:

```bash
npm test -- src/frontend/util/AuthHelpers.test.ts src/frontend/context/AuthContext.test.tsx
npm test
npm run build
```

Expected:

- focused tests: 2 files passed;
- full Vitest run: all test files passed;
- build: TypeScript exits 0 and Vite prints `built in`.

- [ ] **Step 8: Commit the browser session contract**

```bash
git add src/frontend/util/AuthHelpers.ts src/frontend/util/AuthHelpers.test.ts src/frontend/context/AuthContext.tsx src/frontend/context/AuthContext.test.tsx src/frontend/components/Auth/LoginPage.tsx src/frontend/components/Auth/AuthNav.tsx src/frontend/components/Auth/ProfilePage.tsx
git commit -m "feat(frontend): harden auth session handling"
```

---

### Task 7: Migration Rehearsal, Secret Rotation, Documentation, and Rollback

**Files:**
- Modify: `go_server/README.md:119-139`
- Modify: `docs/deploy-render.md:39-119`
- Modify: `render.yaml`
- Modify: `go_server/local.env`
- Modify: `go_server/Makefile`

**Interfaces:**
- Consumes: all prior tasks.
- Produces: an executable rollout/rollback runbook with no destructive automatic account merge.

- [ ] **Step 1: Add a release-path test that covers the complete auth lifecycle**

Extend `go_server/api/route/auth_route_test.go` to execute:

1. signup;
2. profile with bearer access;
3. refresh using only the cookie;
4. assert refresh cookie/token changed;
5. replay the old refresh and assert 401 plus cleared cookie;
6. login again;
7. logout and assert 204 plus cleared legacy/new cookies;
8. refresh after logout and assert 401;
9. mark the user inactive in the fake store and assert old access, login, and refresh all fail.

Do not inspect or assert secret/token values beyond inequality and required claim fields.

- [ ] **Step 2: Run the lifecycle test and all release checks**

Run:

```bash
cd go_server
go test ./api/route -run TestAuthLifecycleRotationReplayLogoutAndInactive -count=1
go test ./... -count=1
go vet ./...
cd ..
npm test
npm run build
```

Expected: lifecycle test and all Go/frontend suites pass; vet and build exit 0.

- [ ] **Step 3: Rehearse the Mongo migration against local Docker**

Use the repository-supported local Mongo:

```bash
make -C go_server mongo-up
make -C go_server mongo-init
run_local_auth_migrate() (
  cd go_server
  set -a
  . ./local.env
  set +a
  DB_ADDR=mongodb://localhost:27017 DB_USER=esteb DB_PASS=localdev go run ./cmd/auth_migrate "$@"
)
run_local_auth_migrate audit
run_local_auth_migrate apply
run_local_auth_migrate audit
```

Expected:

- first audit prints JSON with empty `duplicates` and `missing_email_ids` for a clean local database;
- apply prints `auth indexes applied`;
- second audit exits 0 with the same clean report.

Then verify named indexes:

```bash
mongosh --quiet mongodb://esteb:localdev@localhost:27017/comics?authSource=admin --eval \
'db.users.getIndexes().map(i => i.name).sort(); db.auth_sessions.getIndexes().map(i => i.name).sort(); db.oauth_states.getIndexes().map(i => i.name).sort()'
```

Expected names include all six names from Task 2 plus Mongo `_id_` indexes.

- [ ] **Step 4: Document dirty-preflight handling**

Document that `audit` exit 2 is a hard stop. The JSON report provides exact field/value/user IDs for operator review. The command does not merge/delete because two accounts with the same credential may have different ownership. The safe sequence is:

```bash
set +e
(cd go_server && go run ./cmd/auth_migrate audit) > auth-index-preflight.json
audit_status=$?
set -e
jq . auth-index-preflight.json
test "$audit_status" -eq 0
```

If `duplicates` or `missing_email_ids` is non-empty, keep the current release running, repair ownership through an approved account-support decision, rerun `audit`, and deploy only after it exits 0. No automated “keep lowest UUID” rule is allowed for user accounts.

- [ ] **Step 5: Document the first hardened deployment**

Add the exact order:

1. Confirm the existing secrets are distinct and at least 32 bytes. If either is weak, generate replacements:

   ```bash
   openssl rand -hex 32
   openssl rand -hex 32
   ```

2. Keep strong current production secret values for the first hardened deploy; otherwise use the generated replacements. Set `ACCESS_TOKEN_KEY_ID=access-2026-10` and `REFRESH_TOKEN_KEY_ID=refresh-2026-10`; leave previous key pairs blank.
3. Set `FRONTEND_URL=https://estebmaister.github.io/comics`,
   `OAUTH_STATE_TTL=10m`, `DB_TABLE_AUTH_SESSIONS=auth_sessions`, and
   `DB_TABLE_OAUTH_STATES=oauth_states`. Update `render.yaml` with those
   non-secret values plus `ACCESS_TOKEN_KEY_ID` and `REFRESH_TOKEN_KEY_ID`;
   declare each current/previous signing secret with `sync: false` and leave
   previous key pairs unset for the first deployment.
4. Run `go-auth-migrate audit`, verify it exits 0, then run `apply`. This migration creates indexes and collections but never rewrites or deletes a user; any separate duplicate-account repair requires its own approved backup and recovery procedure.
5. Deploy the API, then frontend.
6. Expect a one-time sign-in reset because old tokens have no `kid`, `aud`, `token_type`, or `sid`.
7. Verify signup/login/profile/refresh/logout/Google OAuth and an admin-only route. UI `isAdmin` is not evidence of server authorization; verify the HTTP status.

- [ ] **Step 6: Document bounded secret rotation**

For access and refresh independently:

1. Move the current ID/secret into its matching previous variables.
2. Generate a new 32-byte-or-longer secret and a new monotonic ID, such as `access-2026-11` or `refresh-2026-11`.
3. Deploy once with current + previous. New tokens use current; in-flight tokens verify with previous.
4. Remove the previous access key only after 15 minutes plus 30 seconds clock skew.
5. Remove the previous refresh key only after 168 hours plus 30 seconds clock skew.
6. Never swap access and refresh secrets, reuse IDs with different bytes, or log either value.

The environment variable names are:

```text
ACCESS_TOKEN_KEY_ID
ACCESS_TOKEN_SECRET
ACCESS_TOKEN_PREVIOUS_KEY_ID
ACCESS_TOKEN_PREVIOUS_SECRET
REFRESH_TOKEN_KEY_ID
REFRESH_TOKEN_SECRET
REFRESH_TOKEN_PREVIOUS_KEY_ID
REFRESH_TOKEN_PREVIOUS_SECRET
```

- [ ] **Step 7: Document rollback**

Application rollback is non-destructive:

1. Restore the prior API/frontend image and the prior current signing-secret values.
2. Expect users holding new-format tokens to sign in again; the old binary does not understand `sid`/`token_type`.
3. Leave `auth_sessions` and `oauth_states` collections in place; the old binary ignores them and TTL cleanup remains safe.
4. Keep user unique indexes unless index creation caused the rollback. To remove only this migration’s indexes:

   ```bash
   make -C go_server go-auth-migrate ARGS='rollback-indexes --confirm'
   ```

5. Confirm users were preserved:

   ```bash
   make -C go_server go-auth-migrate ARGS=audit
   ```

6. Roll forward by deploying the hardened binary and rerunning `apply`; index creation is idempotent.

Cookie/logout rollback compatibility is explicit: both frontend versions clear `comics_access_token`; the new logout endpoint clears legacy `access_token` and `refresh_token`; malformed or old tokens do not prevent a 204 local logout.

Logout revokes refresh capability immediately. An access token already issued before logout can remain valid for at most 15 minutes; reducing that bound requires shortening `ACCESS_TOKEN_EXPIRY_HOUR`, not querying session state on every authorization.

- [ ] **Step 8: Run the complete project check**

Run:

```bash
make check
```

Expected: Go unit/contract/vet/build checks and frontend tests/build all exit
0. Mongo integration tests remain the explicit Docker command:

```bash
make -C go_server go-test-integration
```

Expected: `ok comics/internal/repo/mongo`.

- [ ] **Step 9: Commit docs and release verification**

```bash
git add go_server/api/route/auth_route_test.go go_server/README.md docs/deploy-render.md render.yaml go_server/local.env go_server/Makefile
git commit -m "docs(auth): document hardened auth rollout"
```

---

## Migration Acceptance Checklist

- [ ] `go-auth-migrate audit` reports no duplicate case-insensitive email or non-empty username groups and no missing emails.
- [ ] All six named indexes exist; user, session, and OAuth-state documents remain intact.
- [ ] Password login, OAuth login, token middleware, and refresh all reject inactive users.
- [ ] Access and refresh tokens reject cross-use, unknown `kid`, wrong issuer/audience, missing expiry, and non-HS256 algorithms.
- [ ] Refresh rotation changes JTI/token, replay revokes the session, logout revokes it idempotently, and TTL uses absolute expiry.
- [ ] No endpoint accepts a client role header for authorization; middleware role is loaded from Mongo.
- [ ] OAuth start requests have unique state/nonce/PKCE records; expired, mismatched, and replayed callbacks fail.
- [ ] Google `verified_email=false` and blank email never log in or register.
- [ ] Refresh tokens are absent from JSON/session storage/logs and present only in the configured HttpOnly cookie.
- [ ] Production cookie, exact-origin CORS, CSRF header, and frontend cross-site refresh work on the GitHub Pages/Render topology.
- [ ] Frontend Strict Mode produces only one refresh rotation; `isAdmin` comes only from profile JSON.
- [ ] Logout clears local storage first and expires both legacy and new server cookies even when revocation cannot occur.
- [ ] Current/previous secret rotation and one-time session reset are documented and rehearsed.
- [ ] Rollback preserves users and does not require dropping auth collections.
