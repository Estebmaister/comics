# Security Boundary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep catalog list/get/search public while requiring an authenticated admin bearer JWT for every catalog mutation, merge, cover update, manual scrape trigger, and operational scrape-status request, with bearer-aware and role-aware React behavior.

**Architecture:** Split Gin catalog registration into public-read and admin-operation routers while preserving every existing URL and HTTP method. Admin API routes use a new bearer-only authentication middleware followed by the existing JWT-role middleware; browser profile/dashboard routes retain cookie fallback. The React app gains a small authenticated request primitive, hydrates the user role from `/protected/profile`, sends the session access token on every admin request, clears expired sessions on `401`, and never renders or initiates catalog-management behavior for non-admin users.

**Tech Stack:** Go 1.27+, Gin, HMAC JWT (`github.com/golang-jwt/jwt/v4`), Testify, React 19, TypeScript 6, Fetch API, React Testing Library, Vitest 3, MongoDB users/auth, SQLite catalog by default, optional PostgreSQL catalog.

## Global Constraints

- Keep the Go modular monolith as the production service.
- Keep React + REST JSON as the production client contract.
- Keep SQLite as the default catalog store and MongoDB as the user/auth store; PostgreSQL remains an optional catalog backend and scale-up path.
- Do not introduce Redis or split the application into microservices.
- Public routes remain `GET /comics`, `GET /comics/:id`, and `GET /comics/search/:title`.
- Require an admin JWT for create, update, delete, merge, cover mutation, manual scrape, and operational scrape status.
- Preserve the existing mutation and scrape URLs, HTTP methods, request bodies, response bodies, and status codes in this workstream. In particular, `GET /scrape` remains the compatibility contract until the scrape-reliability workstream deliberately versions or replaces it.
- Browser profile and dashboard routes may continue accepting the existing HttpOnly access-token cookie; catalog administration routes accept only an `Authorization` bearer value containing the session access JWT.
- The backend remains the authorization boundary. Hiding controls in React is usability and defense in depth, never the enforcement mechanism.
- A `401` clears the frontend session; a `403` preserves the valid session and reports insufficient privileges.
- No SQLite, PostgreSQL, or MongoDB schema migration is part of this workstream.
- Do not change discovery, transaction, scrape-result, refresh-token, OAuth, metrics, Swagger-exposure, or readiness semantics here.

---

## Dependencies and Sequencing

- This is workstream 1 and has no dependency on another implementation plan. It depends only on the approved design at `docs/superpowers/specs/2026-10-26-architecture-review-redesign.md` and the existing JWT subject-role contract in `go_server/internal/tokenutil/jwt.go`.
- Release prerequisite: promote and verify at least one existing MongoDB application user with `role: "admin"` before deploying the protected backend. That user must sign in again after promotion because existing access and refresh JWTs retain their old subject role.
- Land this plan before `docs/superpowers/plans/2026-10-26-discovery-data-integrity.md` and `docs/superpowers/plans/2026-10-26-transactional-repository-schema-parity.md` expose additional write-path errors through REST.
- `docs/superpowers/plans/2026-10-26-scrape-reliability.md` may extend the scrape status/result schema and may replace the synchronous trigger contract, but it must retain the bearer-admin middleware established here.
- `docs/superpowers/plans/2026-10-26-auth-hardening.md` replaces both JWT-only authenticators with a Mongo-backed `domain.AccessAuthenticator`, preserves the public-read/admin-write route split, and deliberately removes the legacy access-token cookie fallback once every protected frontend request sends a bearer token. It must continue returning the persisted profile role used by `AuthContext`.
- `docs/superpowers/plans/2026-10-26-frontend-api-consistency.md` should evolve `src/frontend/api/client.ts` from the minimal bearer helper established here into the sole typed REST transport. The DOM session-expired event and direct `ServerHelpers`/`AuthHelpers` integrations in this plan are interim compatibility steps; Workstream 7 removes them after all callers use the typed client. Standard response parsing, abort handling, and list invalidation remain in that later workstream.
- `docs/superpowers/plans/2026-10-26-observability-verification.md` owns denial metrics and cross-workstream security telemetry; this plan supplies route-policy tests but no new metric series.
- Backend and frontend changes are independently testable, but deploy them in one release. An old frontend remains able to read the catalog but receives `401` for every write because it does not send a bearer token.

## File Structure

### Backend

- Modify `go_server/api/middleware/auth.go`: preserve browser cookie authentication and add a bearer-only API middleware; abort redirect paths so denied handlers cannot execute.
- Modify `go_server/api/middleware/auth_test.go`: prove cookie/browser compatibility, bearer-only enforcement, and handler non-execution on denial.
- Modify `go_server/api/route/comics.go`: split route registration into public reads and admin operations without changing handlers or wire contracts.
- Modify `go_server/api/route/route.go`: mount the public and bearer-admin catalog groups separately.
- Create `go_server/api/route/catalog_security_test.go`: route-policy matrix for anonymous, cookie-only, user-bearer, and public-read access.
- Modify `go_server/api/route/comics_test.go`: mount the production-equivalent route boundary in handler tests and authorize mutation/scrape requests as admin.
- Modify `go_server/api/route/comics_contract_test.go`: keep reads unauthenticated and send an admin bearer token for write-contract requests.

### Frontend foundation

- Create `src/frontend/api/client.ts`: attach the session bearer token, set JSON headers, emit the session-expired event on `401`, and normalize backend `message`/`error` text.
- Create `src/frontend/api/client.test.ts`: verify bearer injection and distinct `401`/`403` behavior.
- Modify `src/frontend/util/AuthHelpers.ts`: add `role` to the profile type.
- Modify `src/frontend/context/AuthContext.tsx`: hydrate a persisted session, expose `isAdmin`, and react to the session-expired event.
- Create `src/frontend/context/AuthContext.test.tsx`: verify role hydration and expired-session clearing.

### Frontend admin requests and UX

- Modify `src/frontend/util/ServerHelpers.ts`: route card mutations through the authenticated client and update local state only after a successful response.
- Create `src/frontend/util/ServerHelpers.test.ts`: prove denied card mutations do not alter local state.
- Modify `src/frontend/hooks/useComicActions.ts`: explicitly launch the now-async mutation helpers.
- Modify `src/frontend/components/Comics/Edition/CreateComic.tsx`: authenticate create requests and treat non-2xx responses as failures.
- Modify `src/frontend/components/Comics/Edition/EditComic.tsx`: authenticate edit requests and reject non-2xx payloads.
- Modify `src/frontend/components/Comics/Edition/MergeComic.tsx`: authenticate merge requests.
- Modify `src/frontend/components/Comics/Edition/ScrapeButton.tsx`: authenticate scrape trigger/status requests.
- Create `src/frontend/components/Comics/Edition/catalogMutations.test.ts`: verify create/edit/merge bearer wiring and denied-result behavior.
- Modify `src/frontend/components/Comics/Edition/ScrapeButton.test.ts`: verify bearer wiring for manual scrape.
- Modify `src/frontend/components/Comics/MainPage/MainPage.tsx`: render the floating management rail only for admins.
- Create `src/frontend/components/Comics/MainPage/MainPage.auth.test.tsx`: verify public/user/admin management-rail visibility.
- Modify `src/frontend/components/Comics/Card/ComicCard.tsx`: keep catalog data visible while hiding edit/delete/rating/tracking/checkout controls from non-admins.
- Modify `src/frontend/components/Comics/Card/ComicCover.tsx`: keep local image fallback public but report broken covers only for admins.
- Modify `src/frontend/components/Comics/Card/ComicCard.test.tsx`: verify both admin and public card behavior.

### Documentation and rollout

- Create `docs/security-boundary.md`: route policy, admin promotion, rollout, compatibility, smoke checks, and rollback.
- Modify `README.md`: summarize public-read/admin-write policy and link the runbook.
- Modify `go_server/README.md`: label every REST route public or bearer-admin.
- Modify `docs/superpowers/specs/2026-10-08-scrape-coordination.md`: record that manual scrape and status are bearer-admin operations.
- Modify `CLAUDE.md`: keep workspace route guidance aligned with the implemented boundary.

### Explicitly unchanged

- `go_server/domain/comic.go`: `ComicUseCase` signatures do not change.
- `go_server/internal/tokenutil/jwt.go`: the existing role-in-subject JWT contract is reused.
- `go_server/internal/usecase/`, `go_server/internal/repo/`, and all SQL migrations: authorization stays at the route boundary and requires no data-store change.
- `go_server/docs/swagger.yaml`, `go_server/docs/swagger.json`, and `go_server/docs/docs.go`: catalog/scrape handlers currently have no Swagger annotations, so regeneration would not document this boundary and is deferred until those operations are annotated.

### Task 1: Add Bearer-Only API Authentication

**Files:**
- Modify: `go_server/api/middleware/auth_test.go:40-160`
- Modify: `go_server/api/middleware/auth.go:39-103`

**Interfaces:**
- Consumes: `tokenutil.VerifyToken(token string, secret []byte) (*tokenutil.Claims, error)`, `ExtractCookieAccessToken(*gin.Context, *string)`, `KeyAuthorization`, and `KeyRole`.
- Produces: `BearerAuthenticationMiddleware(accessTokenSecret string) gin.HandlerFunc`.
- Preserves: `AuthenticationMiddleware(accessTokenSecret string) gin.HandlerFunc`, including cookie fallback and browser redirect for profile/dashboard routes.
- Security invariant: every denied request aborts the Gin chain before the protected handler executes.

- [ ] **Step 1: Write the failing bearer-only and redirect-abort tests**

In `go_server/api/middleware/auth_test.go`, change the existing `"No header"` case to expect the login redirect body and add a handler-execution assertion:

```go
	}, {
		name: "No header",
		args: args{secretKey: secretKey, token: "", noContentType: true},
		want: want{code: http.StatusFound, msg: "/login"},
	}, {
```

Replace the handler and final assertions inside `TestAuthenticationMiddleware` with:

```go
			handlerCalled := false
			router := gin.New()
			router.Use(AuthenticationMiddleware(tt.args.secretKey))
			router.GET("/protected", func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, accessGrantedMsg)
			})

			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			if !tt.args.noContentType {
				req.Header.Set(KeyAccept, ContentTypeJSON)
			}
			if tt.args.cookieToken != "" {
				req.AddCookie(&http.Cookie{
					Name:  KeyAccessToken,
					Value: tt.args.cookieToken,
				})
			}
			req.Header.Set(KeyAuthorization, tt.args.token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.want.code, w.Code)
			assert.Contains(t, w.Body.String(), tt.want.msg)
			assert.Equal(t, tt.want.code == http.StatusOK, handlerCalled)
```

Append this complete test:

```go
func TestBearerAuthenticationMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		headerToken string
		cookieToken string
		wantCode    int
	}{
		{
			name:        "accepts bearer header",
			headerToken: bearer + validAdminToken,
			wantCode:    http.StatusOK,
		},
		{
			name:        "rejects cookie without bearer header",
			cookieToken: validAdminToken,
			wantCode:    http.StatusUnauthorized,
		},
		{
			name:     "rejects missing bearer without redirect",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:        "rejects malformed bearer",
			headerToken: validAdminToken,
			wantCode:    http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			router := gin.New()
			router.Use(BearerAuthenticationMiddleware(secretKey))
			router.GET("/admin-api", func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, accessGrantedMsg)
			})

			req := httptest.NewRequest(http.MethodGet, "/admin-api", nil)
			req.Header.Set(KeyAuthorization, tt.headerToken)
			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{
					Name:  KeyAccessToken,
					Value: tt.cookieToken,
				})
			}

			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			assert.Equal(t, tt.wantCode, res.Code)
			assert.Equal(t, tt.wantCode == http.StatusOK, handlerCalled)
			if tt.wantCode != http.StatusOK {
				assert.NotContains(t, res.Body.String(), accessGranted)
				assert.Empty(t, res.Header().Get("Location"))
			}
		})
	}
}
```

- [ ] **Step 2: Run the focused tests and verify the red state**

Run:

```bash
cd go_server
go test ./api/middleware -run 'Test(AuthenticationMiddleware|BearerAuthenticationMiddleware)$' -count=1
```

Expected: build failure containing `undefined: BearerAuthenticationMiddleware`. The changed `"No header"` assertion also records the existing redirect-chain defect that Task 1 fixes.

- [ ] **Step 3: Implement shared authentication with an explicit cookie policy**

Replace `AuthenticationMiddleware` in `go_server/api/middleware/auth.go` and add the bearer-only entry point:

```go
// AuthenticationMiddleware validates an access JWT and permits the existing
// HttpOnly cookie fallback used by browser profile/dashboard routes.
func AuthenticationMiddleware(accessTokenSecret string) gin.HandlerFunc {
	return authenticationMiddleware(accessTokenSecret, true)
}

// BearerAuthenticationMiddleware validates only an Authorization bearer token.
// API callers never fall back to browser cookies or redirects.
func BearerAuthenticationMiddleware(accessTokenSecret string) gin.HandlerFunc {
	return authenticationMiddleware(accessTokenSecret, false)
}

func authenticationMiddleware(accessTokenSecret string, allowCookie bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader(KeyAuthorization)
		if allowCookie {
			ExtractCookieAccessToken(c, &tokenString)
		}

		if tokenString == "" {
			if allowCookie && c.GetHeader(KeyAccept) != ContentTypeJSON {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}

			c.Error(fmt.Errorf("missing authentication token")) // nolint:errcheck
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication token"})
			c.Abort()
			return
		}

		tokenParts := strings.Split(tokenString, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" || tokenParts[1] == "" {
			c.Error(fmt.Errorf("invalid authentication token structure")) // nolint:errcheck
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid structure token"})
			c.Abort()
			return
		}

		claims, err := tokenutil.VerifyToken(tokenParts[1], []byte(accessTokenSecret))
		if err != nil {
			c.Error(err) // nolint:errcheck
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
			c.Abort()
			return
		}

		c.Set(KeyUserID, claims.UserID)
		c.Set(KeyRole, claims.Subject)
		c.Next()
	}
}
```

Keep `ExtractCookieAccessToken` and `RoleMiddleware` unchanged.

- [ ] **Step 4: Run middleware tests**

Run:

```bash
cd go_server
go test ./api/middleware -count=1
```

Expected: `ok  	comics/api/middleware`; cookie auth still passes for `AuthenticationMiddleware`, cookie-only auth fails for `BearerAuthenticationMiddleware`, and denied handlers remain uncalled.

- [ ] **Step 5: Commit the middleware boundary**

```bash
git add go_server/api/middleware/auth.go go_server/api/middleware/auth_test.go
git commit -m "fix(auth): add bearer-only API middleware"
```

### Task 2: Split Public Reads from Admin Catalog Operations

**Files:**
- Create: `go_server/api/route/catalog_security_test.go`
- Modify: `go_server/api/route/comics_test.go:1-239`
- Modify: `go_server/api/route/comics_contract_test.go:1-103`
- Modify: `go_server/api/route/comics.go:36-57`
- Modify: `go_server/api/route/route.go:73-101`

**Interfaces:**
- Consumes: `middleware.BearerAuthenticationMiddleware(string) gin.HandlerFunc`, `middleware.RoleMiddleware(tokenutil.RoleAdmin) gin.HandlerFunc`, and the existing `domain.ComicUseCase`.
- Produces:
  - `catalogReadRouter(comics domain.ComicUseCase, group *gin.RouterGroup)`
  - `catalogAdminRouter(comics domain.ComicUseCase, group *gin.RouterGroup)`
- Public contract: `GET /comics`, `GET /comics/:id`, `GET /comics/search/:title`, and the existing `GET /health/db`.
- Bearer-admin contract: `POST /comics`, `PUT /comics/:id`, `DELETE /comics/:id`, `PATCH /comics/:id/cover-visibility`, `PATCH|PUT /comics/:id/:merging_id`, `GET /scrape`, and `GET /scrape/status`.

- [ ] **Step 1: Add production-equivalent route test helpers**

In `go_server/api/route/comics_test.go`, add `time`, `comics/api/middleware`, `comics/internal/tokenutil`, and `github.com/google/uuid` to the imports, then add:

```go
const catalogTestAccessSecret = "catalog-test-access-secret"

func newComicTestRouter(comics domain.ComicUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	public := router.Group("/")
	catalogReadRouter(comics, public)

	admin := router.Group("/")
	admin.Use(
		middleware.BearerAuthenticationMiddleware(catalogTestAccessSecret),
		middleware.RoleMiddleware(tokenutil.RoleAdmin),
	)
	catalogAdminRouter(comics, admin)
	return router
}

func catalogTokenForRole(t *testing.T, role string) string {
	t.Helper()
	token, err := tokenutil.GenerateTokenWithRole(
		uuid.New(),
		[]byte(catalogTestAccessSecret),
		time.Minute,
		role,
	)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func authorizeCatalogRequest(t *testing.T, req *http.Request, role string) {
	t.Helper()
	req.Header.Set(
		middleware.KeyAuthorization,
		"Bearer "+catalogTokenForRole(t, role),
	)
	req.Header.Set(middleware.KeyAccept, middleware.ContentTypeJSON)
}
```

The compile failure in the next step is intentional because the two router functions do not exist yet.

- [ ] **Step 2: Write the route-policy matrix before changing production routes**

Create `go_server/api/route/catalog_security_test.go`:

```go
package route

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"comics/api/middleware"
	"comics/internal/tokenutil"
)

func TestCatalogReadRoutesRemainPublic(t *testing.T) {
	router := newComicTestRouter(newTestComicService(t))

	tests := []struct {
		path     string
		wantCode int
	}{
		{path: "/comics?from=0&limit=1", wantCode: http.StatusOK},
		{path: "/comics/search/missing?from=0&limit=1", wantCode: http.StatusOK},
		{path: "/comics/999999999", wantCode: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set(middleware.KeyAccept, middleware.ContentTypeJSON)
			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			if res.Code != tt.wantCode {
				t.Fatalf("%s: expected %d, got %d: %s", tt.path, tt.wantCode, res.Code, res.Body.String())
			}
		})
	}
}

func TestCatalogAdminRoutesRejectAnonymousUserAndCookieOnly(t *testing.T) {
	routes := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create", method: http.MethodPost, path: "/comics", body: `{"titles":["Denied create"]}`},
		{name: "update", method: http.MethodPut, path: "/comics/1", body: `{"track":true}`},
		{name: "delete", method: http.MethodDelete, path: "/comics/1"},
		{name: "cover", method: http.MethodPatch, path: "/comics/1/cover-visibility", body: `{"cover":"cover.webp","cover_visible":false}`},
		{name: "merge patch", method: http.MethodPatch, path: "/comics/1/2"},
		{name: "merge put", method: http.MethodPut, path: "/comics/1/2"},
		{name: "scrape", method: http.MethodGet, path: "/scrape"},
		{name: "scrape status", method: http.MethodGet, path: "/scrape/status"},
	}

	principals := []struct {
		name     string
		configure func(*testing.T, *http.Request)
		wantCode int
	}{
		{
			name:      "anonymous",
			configure: func(*testing.T, *http.Request) {},
			wantCode:  http.StatusUnauthorized,
		},
		{
			name: "admin cookie without bearer",
			configure: func(t *testing.T, req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  middleware.KeyAccessToken,
					Value: catalogTokenForRole(t, tokenutil.RoleAdmin),
				})
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "user bearer",
			configure: func(t *testing.T, req *http.Request) {
				authorizeCatalogRequest(t, req, tokenutil.RoleUser)
			},
			wantCode: http.StatusForbidden,
		},
	}

	for _, route := range routes {
		for _, principal := range principals {
			t.Run(route.name+"/"+principal.name, func(t *testing.T) {
				service := &scrapeStubComicService{ComicUseCase: newTestComicService(t)}
				router := newComicTestRouter(service)
				req := httptest.NewRequest(route.method, route.path, bytes.NewBufferString(route.body))
				req.Header.Set(middleware.KeyAccept, middleware.ContentTypeJSON)
				if route.body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				principal.configure(t, req)
				res := httptest.NewRecorder()

				router.ServeHTTP(res, req)

				if res.Code != principal.wantCode {
					t.Fatalf(
						"%s %s as %s: expected %d, got %d: %s",
						route.method,
						route.path,
						principal.name,
						principal.wantCode,
						res.Code,
						res.Body.String(),
					)
				}
			})
		}
	}
}
```

- [ ] **Step 3: Run the route tests and verify the red state**

Run:

```bash
cd go_server
go test ./api/route -run 'TestCatalog(ReadRoutesRemainPublic|AdminRoutesRejectAnonymousUserAndCookieOnly)$' -count=1
```

Expected: build failure containing `undefined: catalogReadRouter` and `undefined: catalogAdminRouter`.

- [ ] **Step 4: Split route registration without changing handlers or paths**

Replace `comicsRouter` in `go_server/api/route/comics.go` with:

```go
func catalogReadRouter(comics domain.ComicUseCase, group *gin.RouterGroup) {
	group.GET("/health/db", func(c *gin.Context) {
		if err := comics.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "database unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	group.GET("/comics", listComics(comics))
	group.GET("/comics/search/:title", searchComics(comics))
	group.GET("/comics/:id", getComic(comics))
}

func catalogAdminRouter(comics domain.ComicUseCase, group *gin.RouterGroup) {
	group.GET("/scrape/status", scrapeStatus(comics))
	group.GET("/scrape", runScrape(comics))

	group.POST("/comics", createComic(comics))
	group.PUT("/comics/:id", updateComic(comics))
	group.DELETE("/comics/:id", deleteComic(comics))
	group.PATCH("/comics/:id/cover-visibility", updateCoverVisibility(comics))
	group.PATCH("/comics/:id/:merging_id", mergeComics(comics))
	group.PUT("/comics/:id/:merging_id", mergeComics(comics))
}
```

In `go_server/api/route/route.go`, replace the catalog call in the public block and add a root-path admin API group while keeping `/admin/dashboard` unchanged:

```go
	publicRouter := g.Group(basePath)
	setOAuth2(env, authController, publicRouter)
	{
		swaggerRouter(env, basePath, publicRouter)
		metricsRouter(compositePinger{userRepo, comics}, publicRouter)
		catalogReadRouter(comics, publicRouter)
		signUpRouter(authController, publicRouter)
		loginRouter(authController, publicRouter)
		refreshTokenRouter(authController, publicRouter)
	}

	catalogAdmin := g.Group(basePath)
	catalogAdmin.Use(
		middleware.BearerAuthenticationMiddleware(env.JWTConfig.AccessTokenSecret),
		middleware.RoleMiddleware(tokenutil.RoleAdmin),
	)
	{
		catalogAdminRouter(comics, catalogAdmin)
	}

	protected := g.Group("/protected")
	protected.Use(
		middleware.AuthenticationMiddleware(env.JWTConfig.AccessTokenSecret),
	)
	{
		protected.GET("/profile", getProfile(authController))
		protected.PUT("/profile", putProfile(authController))
		NewTaskRouter(env, userRepo, protected)
	}

	admin := g.Group("/admin")
	admin.Use(
		middleware.AuthenticationMiddleware(env.JWTConfig.AccessTokenSecret),
		middleware.RoleMiddleware(tokenutil.RoleAdmin),
	)
	{
		dashboardRouter(userRepo, admin)
	}
```

- [ ] **Step 5: Move route handler tests onto the real boundary**

In `go_server/api/route/comics_test.go`, replace each local router setup that calls the removed `comicsRouter` with `newComicTestRouter` and the same comic service. Add an admin token to every mutation, scrape, and status request immediately after constructing it:

```go
authorizeCatalogRequest(t, createReq, tokenutil.RoleAdmin)
authorizeCatalogRequest(t, dupReq, tokenutil.RoleAdmin)
authorizeCatalogRequest(t, updateReq, tokenutil.RoleAdmin)
authorizeCatalogRequest(t, patchReq, tokenutil.RoleAdmin)
authorizeCatalogRequest(t, req, tokenutil.RoleAdmin) // scrape and scrape status tests
```

- [ ] **Step 6: Add admin success and CORS regression tests**

Add this cover-route success test:

```go
func TestUpdateCoverVisibilityAcceptsAdminBearer(t *testing.T) {
	service := newTestComicService(t)
	router := newComicTestRouter(service)

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/comics",
		bytes.NewBufferString(`{"titles":["Cover test"],"cover":"cover.webp"}`),
	)
	createReq.Header.Set("Content-Type", "application/json")
	authorizeCatalogRequest(t, createReq, tokenutil.RoleAdmin)
	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	coverReq := httptest.NewRequest(
		http.MethodPatch,
		"/comics/1/cover-visibility",
		bytes.NewBufferString(`{"cover":"cover.webp","cover_visible":false}`),
	)
	coverReq.Header.Set("Content-Type", "application/json")
	authorizeCatalogRequest(t, coverReq, tokenutil.RoleAdmin)
	coverRes := httptest.NewRecorder()
	router.ServeHTTP(coverRes, coverReq)

	if coverRes.Code != http.StatusOK {
		t.Fatalf("cover update expected 200, got %d: %s", coverRes.Code, coverRes.Body.String())
	}
	if !strings.Contains(coverRes.Body.String(), `"cover_visible":false`) {
		t.Fatalf("expected hidden cover in response, got %s", coverRes.Body.String())
	}
}
```

Add a success test for the compatibility `PUT` merge alias (the contract suite
already exercises `PATCH`):

```go
func TestMergePutAcceptsAdminBearer(t *testing.T) {
	service := newTestComicService(t)
	for _, comic := range []domain.Comic{
		{Titles: []string{"Merge base"}, ComType: 3},
		{Titles: []string{"Merge duplicate"}, ComType: 3},
	} {
		if _, err := service.Create(context.Background(), comic); err != nil {
			t.Fatal(err)
		}
	}
	router := newComicTestRouter(service)
	req := httptest.NewRequest(http.MethodPut, "/comics/1/2", nil)
	authorizeCatalogRequest(t, req, tokenutil.RoleAdmin)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("PUT merge expected 200, got %d: %s", res.Code, res.Body.String())
	}
}
```

Extend `TestCORSMiddlewareAllowsConfiguredOrigin` so the deployed GitHub Pages
frontend can preflight bearer requests:

```go
	if got := res.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
		t.Fatalf("expected Authorization in CORS allow headers, got %q", got)
	}
	if got := res.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "PATCH") {
		t.Fatalf("expected mutation methods in CORS allow methods, got %q", got)
	}
```

- [ ] **Step 7: Move HTTP contract writes behind admin authorization**

In `go_server/api/route/comics_contract_test.go`, use `newComicTestRouter(comics)`. Leave list/search requests unauthenticated, and authorize each `dupReq`, `create`, `dupCreate`, `patchReq`, `mergeReq`, and `deleteReq`:

```go
authorizeCatalogRequest(t, dupReq, tokenutil.RoleAdmin)
authorizeCatalogRequest(t, create, tokenutil.RoleAdmin)
authorizeCatalogRequest(t, dupCreate, tokenutil.RoleAdmin)
authorizeCatalogRequest(t, patchReq, tokenutil.RoleAdmin)
authorizeCatalogRequest(t, mergeReq, tokenutil.RoleAdmin)
authorizeCatalogRequest(t, deleteReq, tokenutil.RoleAdmin)
```

Add `"comics/internal/tokenutil"` to that test file's imports.

- [ ] **Step 8: Run route and HTTP contract tests**

Run:

```bash
cd go_server
go test ./api/route -count=1
make contract-test
```

Expected:

```text
ok  	comics/api/route
ok  	comics/api/route
```

The policy matrix must cover all eight admin route shapes, public reads must return their normal `200`/`404` responses without a token, and contract writes must pass with an admin bearer.

- [ ] **Step 9: Commit the backend route boundary**

```bash
git add \
  go_server/api/route/route.go \
  go_server/api/route/comics.go \
  go_server/api/route/comics_test.go \
  go_server/api/route/comics_contract_test.go \
  go_server/api/route/catalog_security_test.go
git commit -m "feat(api): protect catalog administration routes"
```

### Task 3: Add the Frontend Bearer Client and Role State

**Files:**
- Create: `src/frontend/api/client.test.ts`
- Create: `src/frontend/api/client.ts`
- Create: `src/frontend/context/AuthContext.test.tsx`
- Modify: `src/frontend/util/AuthHelpers.ts:15-22`
- Modify: `src/frontend/context/AuthContext.tsx:1-99`

**Interfaces:**
- Consumes: `readAccessToken()`, `clearAuthStorage()`, `config.SERVER`, and the backend profile field `role`.
- Produces:
  - `AUTH_SESSION_EXPIRED_EVENT = "comics:auth-session-expired"`
  - `authorizedFetch(path: string, init?: RequestInit, server?: string): Promise<Response>`
  - `apiErrorMessage(payload: unknown, fallback: string): string`
  - `AuthContextValue.isAdmin: boolean`
- Event contract: `authorizedFetch` dispatches `AUTH_SESSION_EXPIRED_EVENT` only for `401`; `AuthProvider` clears in-memory auth state when that event fires.

- [ ] **Step 1: Write failing bearer-client tests**

Create `src/frontend/api/client.test.ts`:

```ts
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import {
  AUTH_SESSION_EXPIRED_EVENT,
  apiErrorMessage,
  authorizedFetch,
} from './client';
import { readAccessToken, storeAccessToken } from '../util/AuthHelpers';

describe('authorizedFetch', () => {
  beforeEach(() => {
    sessionStorage.clear();
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    sessionStorage.clear();
    vi.unstubAllGlobals();
  });

  test('attaches the stored bearer token and JSON headers', async () => {
    storeAccessToken('admin-access-token');
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      status: 200,
    } as Response);

    await authorizedFetch(
      '/comics/7',
      {
        method: 'PUT',
        body: JSON.stringify({ track: true }),
        headers: { 'X-Request-Test': 'yes' },
      },
      'https://api.example.test',
    );

    expect(fetch).toHaveBeenCalledTimes(1);
    const [url, init] = vi.mocked(fetch).mock.calls[0];
    const headers = new Headers(init?.headers);
    expect(url).toBe('https://api.example.test/comics/7');
    expect(headers.get('Authorization')).toBe('Bearer admin-access-token');
    expect(headers.get('Accept')).toBe('application/json');
    expect(headers.get('Content-Type')).toBe('application/json');
    expect(headers.get('X-Request-Test')).toBe('yes');
    expect(init?.credentials).not.toBe('include');
  });

  test('clears the session and emits expiration on 401', async () => {
    storeAccessToken('expired-token');
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 401,
    } as Response);
    const listener = vi.fn();
    window.addEventListener(AUTH_SESSION_EXPIRED_EVENT, listener);

    try {
      await authorizedFetch('/comics/7', { method: 'DELETE' }, 'https://api.example.test');
    } finally {
      window.removeEventListener(AUTH_SESSION_EXPIRED_EVENT, listener);
    }

    expect(readAccessToken()).toBeUndefined();
    expect(listener).toHaveBeenCalledTimes(1);
  });

  test('keeps a valid session on 403', async () => {
    storeAccessToken('user-access-token');
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 403,
    } as Response);

    await authorizedFetch('/comics/7', { method: 'DELETE' }, 'https://api.example.test');

    expect(readAccessToken()).toBe('user-access-token');
  });

  test('normalizes message and error response bodies', () => {
    expect(apiErrorMessage({ message: 'Conflict' }, 'Fallback')).toBe('Conflict');
    expect(apiErrorMessage({ error: 'Insufficient privileges' }, 'Fallback'))
      .toBe('Insufficient privileges');
    expect(apiErrorMessage('plain failure', 'Fallback')).toBe('plain failure');
    expect(apiErrorMessage(null, 'Fallback')).toBe('Fallback');
  });
});
```

- [ ] **Step 2: Write failing auth-role hydration tests**

Create `src/frontend/context/AuthContext.test.tsx`:

```tsx
import { act, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { AuthProvider, useAuth } from './AuthContext';
import { AUTH_SESSION_EXPIRED_EVENT } from '../api/client';
import { storeAccessToken } from '../util/AuthHelpers';

const AuthProbe = () => {
  const { isAuthenticated, isAdmin, user } = useAuth();
  return (
    <output>
      {`${isAuthenticated ? 'authenticated' : 'anonymous'}:${isAdmin ? 'admin' : 'not-admin'}:${user?.role ?? 'no-role'}`}
    </output>
  );
};

beforeEach(() => {
  sessionStorage.clear();
});

afterEach(() => {
  sessionStorage.clear();
  vi.unstubAllGlobals();
});

test('hydrates the persisted profile role and exposes isAdmin', async () => {
  storeAccessToken('admin-access-token');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve({
      id: 'user-1',
      username: 'catalog-admin',
      email: 'admin@example.test',
      role: 'admin',
    }),
  }));

  render(
    <AuthProvider>
      <AuthProbe />
    </AuthProvider>,
  );

  await waitFor(() => {
    expect(screen.getByText('authenticated:admin:admin')).toBeInTheDocument();
  });

  const [, init] = vi.mocked(fetch).mock.calls[0];
  expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer admin-access-token');
});

test('clears in-memory auth when an authorized request receives 401', async () => {
  storeAccessToken('admin-access-token');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve({
      id: 'user-1',
      username: 'catalog-admin',
      email: 'admin@example.test',
      role: 'admin',
    }),
  }));

  render(
    <AuthProvider>
      <AuthProbe />
    </AuthProvider>,
  );
  await waitFor(() => {
    expect(screen.getByText('authenticated:admin:admin')).toBeInTheDocument();
  });

  act(() => {
    window.dispatchEvent(new Event(AUTH_SESSION_EXPIRED_EVENT));
  });

  expect(screen.getByText('anonymous:not-admin:no-role')).toBeInTheDocument();
});
```

- [ ] **Step 3: Run the focused frontend tests and verify the red state**

Run:

```bash
npm test -- src/frontend/api/client.test.ts src/frontend/context/AuthContext.test.tsx
```

Expected: FAIL because `api/client.ts` does not exist and `AuthContextValue` has no `isAdmin`.

- [ ] **Step 4: Implement the authenticated request primitive**

Create `src/frontend/api/client.ts`:

```ts
import config from '../util/Config';
import { clearAuthStorage, readAccessToken } from '../util/AuthHelpers';

export const AUTH_SESSION_EXPIRED_EVENT = 'comics:auth-session-expired';

export const apiErrorMessage = (payload: unknown, fallback: string): string => {
  if (typeof payload === 'string' && payload.trim() !== '') {
    return payload;
  }
  if (payload !== null && typeof payload === 'object') {
    const body = payload as { message?: unknown; error?: unknown };
    if (typeof body.message === 'string' && body.message.trim() !== '') {
      return body.message;
    }
    if (typeof body.error === 'string' && body.error.trim() !== '') {
      return body.error;
    }
  }
  return fallback;
};

export const authorizedFetch = async (
  path: string,
  init: RequestInit = {},
  server = config.SERVER,
): Promise<Response> => {
  const headers = new Headers(init.headers ?? {});
  headers.set('Accept', 'application/json');
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const accessToken = readAccessToken();
  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }

  const response = await fetch(`${server}${path}`, {
    ...init,
    headers,
  });

  if (response.status === 401) {
    clearAuthStorage();
    window.dispatchEvent(new Event(AUTH_SESSION_EXPIRED_EVENT));
  }

  return response;
};
```

- [ ] **Step 5: Add profile role typing and role-aware auth state**

In `src/frontend/util/AuthHelpers.ts`, replace `AuthUser` with:

```ts
export type AuthUser = {
  id?: string;
  username?: string;
  email?: string;
  role?: string;
};
```

In `src/frontend/context/AuthContext.tsx`, add `useEffect` to the React import and import the event:

```ts
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { AUTH_SESSION_EXPIRED_EVENT } from '../api/client';
```

Add `isAdmin` to the context interface:

```ts
type AuthContextValue = {
  user: AuthUser | null;
  accessToken?: string;
  isAuthenticated: boolean;
  isAdmin: boolean;
  login: (email: string, password: string) => Promise<string | null>;
  signup: (email: string, username: string, password: string) => Promise<string | null>;
  logout: () => void;
  refreshProfile: () => Promise<void>;
};
```

Make the no-token path clear both pieces of state:

```ts
    if (!token) {
      setUser(null);
      setAccessToken(undefined);
      return;
    }
```

After `logout`, add session hydration and expiration-event effects:

```ts
  useEffect(() => {
    void refreshProfile();
  }, [refreshProfile]);

  useEffect(() => {
    const handleSessionExpired = () => {
      setUser(null);
      setAccessToken(undefined);
    };
    window.addEventListener(AUTH_SESSION_EXPIRED_EVENT, handleSessionExpired);
    return () => {
      window.removeEventListener(AUTH_SESSION_EXPIRED_EVENT, handleSessionExpired);
    };
  }, []);
```

Add the exact admin predicate to the memoized value:

```ts
  const value = useMemo(
    () => ({
      user,
      accessToken,
      isAuthenticated: Boolean(accessToken),
      isAdmin: user?.role === 'admin',
      login,
      signup,
      logout,
      refreshProfile,
    }),
    [accessToken, login, logout, refreshProfile, signup, user],
  );
```

- [ ] **Step 6: Run bearer-client and auth-context tests**

Run:

```bash
npm test -- src/frontend/api/client.test.ts src/frontend/context/AuthContext.test.tsx
```

Expected: both files PASS; `401` clears storage and context, `403` keeps the token, and a persisted admin profile produces `isAdmin === true`.

- [ ] **Step 7: Commit the frontend auth foundation**

```bash
git add \
  src/frontend/api/client.ts \
  src/frontend/api/client.test.ts \
  src/frontend/util/AuthHelpers.ts \
  src/frontend/context/AuthContext.tsx \
  src/frontend/context/AuthContext.test.tsx
git commit -m "feat(frontend): add bearer-aware admin session"
```

### Task 4: Authenticate Every Catalog Mutation and Scrape Request

**Files:**
- Create: `src/frontend/util/ServerHelpers.test.ts`
- Create: `src/frontend/components/Comics/Edition/catalogMutations.test.ts`
- Modify: `src/frontend/util/ServerHelpers.ts:1-216`
- Modify: `src/frontend/hooks/useComicActions.ts:28-44`
- Modify: `src/frontend/components/Comics/Edition/CreateComic.tsx:10-35`
- Modify: `src/frontend/components/Comics/Edition/EditComic.tsx:11-43`
- Modify: `src/frontend/components/Comics/Edition/MergeComic.tsx:13-49`
- Modify: `src/frontend/components/Comics/Edition/ScrapeButton.tsx:15-52`
- Modify: `src/frontend/components/Comics/Edition/ScrapeButton.test.ts:1-55`

**Interfaces:**
- Consumes: `authorizedFetch(path, init, server)` and `apiErrorMessage(payload, fallback)` from Task 3.
- Produces:
  - `createComic(comic: CreateComicFormState, server?: string): Promise<string>`
  - `editComic(comic: Comic, server?: string): Promise<[Comic | undefined, string]>`
  - Existing `mergeComic`, `scrape`, `fetchScrapeStatus`, and card helper signatures, now bearer-authenticated.
  - `trackComic`, `checkoutComic`, and `delComic` return `Promise<boolean>` so callers and tests can distinguish success.
- State invariant: no callback or local card-state setter runs after a network exception or non-2xx response.

- [ ] **Step 1: Write failing card-mutation denial tests**

Create `src/frontend/util/ServerHelpers.test.ts`:

```ts
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import {
  checkoutComic,
  delComic,
  rateComic,
  reportCoverVisibility,
  trackComic,
} from './ServerHelpers';
import { storeAccessToken } from './AuthHelpers';

beforeEach(() => {
  sessionStorage.clear();
  storeAccessToken('admin-access-token');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status: 403,
    json: () => Promise.resolve({ error: 'Insufficient privileges' }),
  }));
});

afterEach(() => {
  sessionStorage.clear();
  vi.unstubAllGlobals();
});

test('denied card mutations neither report success nor change local state', async () => {
  const setTrack = vi.fn();
  const setComic = vi.fn();
  const setCheck = vi.fn();
  const setViewedChap = vi.fn();
  const checkoutSuccess = vi.fn();
  const setDelete = vi.fn();
  const deleteSuccess = vi.fn();

  await expect(trackComic(false, 7, setTrack, 'https://api.example.test')).resolves.toBe(false);
  await expect(rateComic(4, 7, setComic, 'https://api.example.test')).resolves.toBe(false);
  await expect(
    checkoutComic(12, 7, setCheck, setViewedChap, checkoutSuccess, 'https://api.example.test'),
  ).resolves.toBe(false);
  await expect(
    delComic(7, setDelete, deleteSuccess, 'https://api.example.test'),
  ).resolves.toBe(false);
  await expect(
    reportCoverVisibility(7, 'cover.webp', false, 'https://api.example.test'),
  ).rejects.toThrow('Insufficient privileges');

  expect(setTrack).not.toHaveBeenCalled();
  expect(setComic).not.toHaveBeenCalled();
  expect(setCheck).not.toHaveBeenCalled();
  expect(setViewedChap).not.toHaveBeenCalled();
  expect(checkoutSuccess).not.toHaveBeenCalled();
  expect(setDelete).not.toHaveBeenCalled();
  expect(deleteSuccess).not.toHaveBeenCalled();

  for (const [, init] of vi.mocked(fetch).mock.calls) {
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer admin-access-token');
  }
});
```

- [ ] **Step 2: Write failing create/edit/merge bearer tests**

Create `src/frontend/components/Comics/Edition/catalogMutations.test.ts`:

```ts
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { storeAccessToken } from '../../../util/AuthHelpers';
import type { Comic, CreateComicFormState } from '../types';
import { createComic } from './CreateComic';
import { editComic } from './EditComic';
import { mergeComic } from './MergeComic';

const createFixture: CreateComicFormState = {
  title: 'Boundary test',
  track: false,
  current_chap: 1,
  viewed_chap: 0,
  cover: 'cover.webp',
  description: '',
  author: '',
  com_type: 3,
  status: 0,
  published_in: [],
  genres: [],
};

const comicFixture: Comic = {
  id: 7,
  titles: ['Boundary test'],
  track: false,
  current_chap: 1,
  viewed_chap: 0,
  cover: 'cover.webp',
  author: '',
  com_type: 3,
  status: 0,
  published_in: [],
  genres: [],
};

beforeEach(() => {
  sessionStorage.clear();
  storeAccessToken('admin-access-token');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status: 403,
    headers: new Headers({ 'content-type': 'application/json' }),
    json: () => Promise.resolve({ error: 'Insufficient privileges' }),
    text: () => Promise.resolve('Insufficient privileges'),
  }));
});

afterEach(() => {
  sessionStorage.clear();
  vi.unstubAllGlobals();
});

test('create edit and merge expose denied responses and send the bearer token', async () => {
  await expect(createComic(createFixture, 'https://api.example.test'))
    .resolves.toBe('Insufficient privileges');
  await expect(editComic(comicFixture, 'https://api.example.test'))
    .resolves.toEqual([undefined, 'Insufficient privileges']);
  await expect(mergeComic(7, 8, 'https://api.example.test'))
    .resolves.toBe('Insufficient privileges');

  expect(fetch).toHaveBeenCalledTimes(3);
  for (const [, init] of vi.mocked(fetch).mock.calls) {
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer admin-access-token');
  }
});
```

In `src/frontend/components/Comics/Edition/ScrapeButton.test.ts`, store a token in `beforeEach`, clear it in `afterEach`, and add this assertion to the successful-response test:

```ts
    const [, init] = vi.mocked(fetch).mock.calls[0];
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer admin-access-token');
```

- [ ] **Step 3: Run mutation tests and verify the red state**

Run:

```bash
npm test -- \
  src/frontend/util/ServerHelpers.test.ts \
  src/frontend/components/Comics/Edition/catalogMutations.test.ts \
  src/frontend/components/Comics/Edition/ScrapeButton.test.ts
```

Expected: FAIL because create/edit/merge are not exported, raw requests have no `Authorization` header, and denied card requests currently call local-state setters.

- [ ] **Step 4: Convert card mutation helpers to authenticated, success-gated promises**

In `src/frontend/util/ServerHelpers.ts`, import the client:

```ts
import { apiErrorMessage, authorizedFetch } from '../api/client';
```

Replace the five mutation helpers with:

```ts
const trackComic = async (
  tracked: boolean,
  id: number,
  setTrack: (value: boolean) => void,
  server = SERVER,
): Promise<boolean> => {
  try {
    const response = await authorizedFetch(
      `/comics/${id}`,
      {
        method: 'PUT',
        body: JSON.stringify({ track: !tracked }),
      },
      server,
    );
    await response.json().catch(() => null);
    if (!response.ok) return false;
    setTrack(!tracked);
    return true;
  } catch (error) {
    console.debug((error as Error)?.message ?? error);
    return false;
  }
};

const rateComic = async (
  rating: number,
  id: number,
  setComic: (value: SetStateAction<Comic>) => void,
  server = SERVER,
): Promise<boolean> => {
  const next = Math.max(0, Math.min(5, Number.isFinite(rating) ? rating : 0));
  try {
    const response = await authorizedFetch(
      `/comics/${id}`,
      {
        method: 'PUT',
        body: JSON.stringify({ rating: next }),
      },
      server,
    );
    await response.json().catch(() => null);
    if (!response.ok) return false;
    setComic((prev) => ({ ...prev, rating: next }));
    return true;
  } catch (error) {
    console.debug((error as Error)?.message ?? error);
    return false;
  }
};

const checkoutComic = async (
  currChap: number,
  id: number,
  setCheck: (value: SetStateAction<boolean>) => void,
  setViewedChap: (value: SetStateAction<number>) => void,
  onSuccess?: () => void,
  server = SERVER,
): Promise<boolean> => {
  try {
    const response = await authorizedFetch(
      `/comics/${id}`,
      {
        method: 'PUT',
        body: JSON.stringify({ viewed_chap: currChap }),
      },
      server,
    );
    await response.json().catch(() => null);
    if (!response.ok) return false;
    setCheck(false);
    setViewedChap(currChap);
    onSuccess?.();
    return true;
  } catch (error) {
    console.debug((error as Error)?.message ?? error);
    return false;
  }
};

const delComic = async (
  id: number,
  setDelete: (value: SetStateAction<boolean>) => void,
  onSuccess?: () => void,
  server = SERVER,
): Promise<boolean> => {
  try {
    const response = await authorizedFetch(
      `/comics/${id}`,
      { method: 'DELETE' },
      server,
    );
    await response.json().catch(() => null);
    if (!response.ok) return false;
    setDelete(true);
    onSuccess?.();
    return true;
  } catch (error) {
    console.debug((error as Error)?.message ?? error);
    return false;
  }
};

const reportCoverVisibility = async (
  id: number,
  cover: string,
  coverVisible: boolean,
  server = SERVER,
): Promise<Comic | undefined> => {
  const response = await authorizedFetch(
    `/comics/${id}/cover-visibility`,
    {
      method: 'PATCH',
      body: JSON.stringify({ cover, cover_visible: coverVisible }),
    },
    server,
  );
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    throw new Error(apiErrorMessage(
      data,
      `Cover visibility update failed (${response.status})`,
    ));
  }
  return data as Comic;
};
```

Keep `dataFetch` on raw `fetch`: list and search are intentionally public.

- [ ] **Step 5: Launch async card mutations explicitly**

In `src/frontend/hooks/useComicActions.ts`, make the three launched promises explicit:

```ts
  const handleCheckout = useCallback(() => {
    void checkoutComic(currentChap, comicId, setCheck, setViewedChap, onCheckoutSuccess);
  }, [comicId, currentChap, onCheckoutSuccess, setCheck, setViewedChap]);

  const handleTrackToggle = useCallback(() => {
    void trackComic(isTracked, comicId, setTrackComic);
  }, [comicId, isTracked, setTrackComic]);

  const handleDelete = useCallback(() => {
    void delComic(comicId, setDel, onDeleteSuccess);
  }, [comicId, onDeleteSuccess, setDel]);
```

- [ ] **Step 6: Convert create to an authenticated request**

In `src/frontend/components/Comics/Edition/CreateComic.tsx`, import `apiErrorMessage` and `authorizedFetch`, then replace `create` with:

```ts
export const createComic = async (
  comic: CreateComicFormState,
  server = SERVER,
): Promise<string> => {
  const payload = {
    ...comic,
    last_update: new Date().getTime(),
    titles: [comic.title],
  };

  try {
    const response = await authorizedFetch(
      '/comics',
      {
        method: 'POST',
        body: JSON.stringify(payload),
      },
      server,
    );
    const data = await response.json().catch(() => null);
    if (!response.ok) {
      return apiErrorMessage(data, `Create request failed (${response.status})`);
    }
    return '';
  } catch (error) {
    return error instanceof Error ? error.message : 'Unexpected network error';
  }
};
```

Change the submit call to:

```ts
    const resultMsg = await createComic(data);
```

- [ ] **Step 7: Convert edit to an authenticated request**

In `src/frontend/components/Comics/Edition/EditComic.tsx`, import the same client helpers and replace `edit` with:

```ts
export const editComic = async (
  comic: Comic,
  server = SERVER,
): Promise<[Comic | undefined, string]> => {
  const { deleted, track, ...payload } = comic;

  try {
    const response = await authorizedFetch(
      `/comics/${comic.id}`,
      {
        method: 'PUT',
        body: JSON.stringify(payload),
      },
      server,
    );
    const data = await response.json().catch(() => null);
    if (!response.ok) {
      return [
        undefined,
        apiErrorMessage(data, `Update request failed (${response.status})`),
      ];
    }
    if (data === null || typeof data !== 'object') {
      return [undefined, 'Update returned an invalid comic payload'];
    }
    return [data as Comic, ''];
  } catch (error) {
    return [
      undefined,
      error instanceof Error ? error.message : 'Unexpected network error',
    ];
  }
};
```

Change the submit call to:

```ts
    const [newData, resultMsg] = await editComic(data);
```

- [ ] **Step 8: Convert merge to an authenticated request**

In `src/frontend/components/Comics/Edition/MergeComic.tsx`, export the request helper, import the client helpers, and replace its request body with:

```ts
export const mergeComic = async (
  baseID: number,
  mergingID: number,
  server = SERVER,
): Promise<string> => {
  try {
    const response = await authorizedFetch(
      `/comics/${baseID}/${mergingID}`,
      { method: 'PATCH' },
      server,
    );
    const contentType = response.headers.get('content-type') ?? '';
    const payload = contentType.includes('application/json')
      ? await response.json()
      : await response.text();

    if (!response.ok) {
      return apiErrorMessage(payload, `Merge request failed (${response.status})`);
    }
    return '';
  } catch (error) {
    return error instanceof Error ? error.message : 'Unexpected network error';
  }
};
```

- [ ] **Step 9: Authenticate scrape trigger and operational status**

In `src/frontend/components/Comics/Edition/ScrapeButton.tsx`, import `authorizedFetch` and replace the two exported helpers with:

```ts
export const fetchScrapeStatus = async (
  server = SERVER,
): Promise<ScrapeStatusResponse | null> => {
  try {
    const response = await authorizedFetch(
      '/scrape/status',
      { method: 'GET' },
      server,
    );
    if (!response.ok) {
      return null;
    }
    return await response.json() as ScrapeStatusResponse;
  } catch {
    return null;
  }
};

export const scrape = async (
  setShowLoader: (value: SetStateAction<boolean>) => void,
  server = SERVER,
) => {
  setShowLoader(true);
  try {
    const response = await authorizedFetch(
      '/scrape',
      { method: 'GET' },
      server,
    );
    const data = await response.json().catch(() => null);
    if (response.status === 409) {
      return {
        ok: false as const,
        busy: true,
        source: data?.source as string | undefined,
      };
    }
    return { ok: response.ok, busy: false as const };
  } catch {
    return { ok: false, busy: false as const };
  } finally {
    setShowLoader(false);
  }
};
```

- [ ] **Step 10: Run mutation tests and scan for bypasses**

Run:

```bash
npm test -- \
  src/frontend/util/ServerHelpers.test.ts \
  src/frontend/components/Comics/Edition/catalogMutations.test.ts \
  src/frontend/components/Comics/Edition/ScrapeButton.test.ts
rg -n '\bfetch\(' \
  src/frontend/components/Comics \
  src/frontend/hooks/useComicActions.ts \
  src/frontend/util/ServerHelpers.ts
```

Expected: all focused tests PASS. The `rg` output contains only the public `dataFetch` call in `src/frontend/util/ServerHelpers.ts`; no edition, card-mutation, cover-mutation, merge, or scrape module calls raw `fetch`.

- [ ] **Step 11: Commit authenticated catalog calls**

```bash
git add \
  src/frontend/util/ServerHelpers.ts \
  src/frontend/util/ServerHelpers.test.ts \
  src/frontend/hooks/useComicActions.ts \
  src/frontend/components/Comics/Edition/CreateComic.tsx \
  src/frontend/components/Comics/Edition/EditComic.tsx \
  src/frontend/components/Comics/Edition/MergeComic.tsx \
  src/frontend/components/Comics/Edition/ScrapeButton.tsx \
  src/frontend/components/Comics/Edition/ScrapeButton.test.ts \
  src/frontend/components/Comics/Edition/catalogMutations.test.ts
git commit -m "feat(frontend): authenticate catalog mutations"
```

### Task 5: Make Catalog Management UI Admin-Only

**Files:**
- Create: `src/frontend/components/Comics/MainPage/MainPage.auth.test.tsx`
- Modify: `src/frontend/components/Comics/MainPage/MainPage.tsx:1-266`
- Modify: `src/frontend/components/Comics/Card/ComicCard.test.tsx:1-123`
- Modify: `src/frontend/components/Comics/Card/ComicCard.tsx:1-219`
- Modify: `src/frontend/components/Comics/Card/ComicCover.tsx:15-49`

**Interfaces:**
- Consumes: `useAuth().isAdmin` from Task 3 and authenticated mutation helpers from Task 4.
- Produces: `ComicCoverProps.canReportFailure: boolean`.
- UX invariant: public and ordinary-user visitors can read/copy catalog data but cannot see or activate create, scrape, merge, edit, delete, rating, tracking, checkout, or broken-cover-report controls.
- Hydration invariant: management controls default hidden until the persisted token has been validated and an admin profile has loaded.

- [ ] **Step 1: Write the failing main-page role test**

Create `src/frontend/components/Comics/MainPage/MainPage.auth.test.tsx`:

```tsx
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { ToastProvider } from '../../Toast/ToastProvider';
import { ComicsMainPage } from './MainPage';
import { dataFetch } from '../../../util/ServerHelpers';

const authState = vi.hoisted(() => ({
  user: null as null | { username: string; role: string },
  accessToken: undefined as string | undefined,
  isAuthenticated: false,
  isAdmin: false,
  login: vi.fn(),
  signup: vi.fn(),
  logout: vi.fn(),
  refreshProfile: vi.fn(),
}));

vi.mock('../../../context/AuthContext', () => ({
  useAuth: () => authState,
}));

vi.mock('../../../util/ServerHelpers', () => ({
  dataFetch: vi.fn(),
  ComicListFetchError: class ComicListFetchError extends Error {
    constructor(message: string, public readonly kind: 'network' | 'server' = 'network') {
      super(message);
    }
  },
  isAbortError: () => false,
}));

const renderPage = () => render(
  <ToastProvider>
    <MemoryRouter>
      <ComicsMainPage />
    </MemoryRouter>
  </ToastProvider>,
);

beforeEach(() => {
  authState.user = null;
  authState.accessToken = undefined;
  authState.isAuthenticated = false;
  authState.isAdmin = false;
  vi.mocked(dataFetch).mockResolvedValue({
    comics: [],
    pagination: { total: 0, totalPages: 1, currentPage: 1 },
  });
});

afterEach(() => {
  vi.clearAllMocks();
});

test('shows catalog utilities only after an admin role is present', async () => {
  const view = renderPage();
  await waitFor(() => expect(dataFetch).toHaveBeenCalled());
  expect(screen.queryByRole('button', { name: /open comic utilities/i })).not.toBeInTheDocument();

  authState.user = { username: 'reader', role: 'user' };
  authState.accessToken = 'user-token';
  authState.isAuthenticated = true;
  view.rerender(
    <ToastProvider>
      <MemoryRouter>
        <ComicsMainPage />
      </MemoryRouter>
    </ToastProvider>,
  );
  expect(screen.queryByRole('button', { name: /open comic utilities/i })).not.toBeInTheDocument();

  authState.user = { username: 'catalog-admin', role: 'admin' };
  authState.accessToken = 'admin-token';
  authState.isAdmin = true;
  view.rerender(
    <ToastProvider>
      <MemoryRouter>
        <ComicsMainPage />
      </MemoryRouter>
    </ToastProvider>,
  );
  expect(screen.getByRole('button', { name: /open comic utilities/i })).toBeInTheDocument();
});
```

- [ ] **Step 2: Extend card tests for public and admin behavior**

At the top of `src/frontend/components/Comics/Card/ComicCard.test.tsx`, add:

```ts
const authState = vi.hoisted(() => ({
  isAdmin: true,
}));

vi.mock('../../../context/AuthContext', () => ({
  useAuth: () => authState,
}));
```

In the existing `beforeEach`, set an access token and reset the role:

```ts
  authState.isAdmin = true;
  sessionStorage.setItem('comics_access_token', 'admin-access-token');
```

Append this test:

```tsx
test('keeps catalog data public but hides every mutation control for non-admins', async () => {
  authState.isAdmin = false;
  render(
    <ToastProvider>
      <ComicCard comic={comicFixture} />
    </ToastProvider>,
  );

  expect(screen.getByText('Rise from the bottom')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: /copy comic id 15816/i })).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /delete rise from the bottom/i })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /edit/i })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /checkout/i })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /untrack/i })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /set rating/i })).not.toBeInTheDocument();

  fireEvent.error(screen.getByRole('img', { name: /rise from the bottom/i }));
  expect(screen.getByTestId('poster-fallback')).toBeInTheDocument();
  await waitFor(() => {
    expect(global.fetch).not.toHaveBeenCalled();
  });
});
```

In the existing admin broken-cover assertion, also verify the bearer:

```ts
    const [, init] = vi.mocked(global.fetch).mock.calls[0];
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer admin-access-token');
```

- [ ] **Step 3: Run role-aware UI tests and verify the red state**

Run:

```bash
npm test -- \
  src/frontend/components/Comics/MainPage/MainPage.auth.test.tsx \
  src/frontend/components/Comics/Card/ComicCard.test.tsx
```

Expected: FAIL because `ComicsMainPage` always renders the rail, `ComicCard` always renders mutation controls, and `ComicCover` always reports failures.

- [ ] **Step 4: Gate the floating management rail**

In `src/frontend/components/Comics/MainPage/MainPage.tsx`, import `useAuth`, read `isAdmin` at the start of `ComicsMainPage`, and replace the unconditional rail with:

```tsx
  const { isAdmin } = useAuth();
```

```tsx
        {isAdmin ? (
          <FloatingActionRail>
            <ScrapeButton onSuccess={requestRefresh} />
            <MergeComic onSuccess={requestRefresh} />
            <CreateComic onSuccess={requestRefresh} />
          </FloatingActionRail>
        ) : null}
```

- [ ] **Step 5: Gate the card overlay and checkout predicate**

In `src/frontend/components/Comics/Card/ComicCard.tsx`, import `useAuth`, read `isAdmin`, and make checkout admin-only:

```tsx
  const { isAdmin } = useAuth();
  const [comic, setComic] = useState(initialComic);
  const { id, current_chap } = comic;
  const [viewedChap, setViewedChap] = useState<number>(comic.viewed_chap);
  const [check, setCheck] = useState(current_chap > viewedChap);
  const [del, setDel] = useState(false);
  const showCheckout = useMemo(
    () => isAdmin && comic.track && check,
    [comic.track, check, isAdmin],
  );
```

Pass authorization intent to the cover and conditionally render overlay mutations:

```tsx
          <ComicCover
            comic={comic}
            canReportFailure={isAdmin}
            onCoverVisibilityChange={(coverVisible) => {
              setComic((prev) => ({ ...prev, cover_visible: coverVisible }));
            }}
          >
            {isAdmin ? (
              <div className={styles.overlayActions}>
                <button
                  className={`${styles.overlayButton} ${styles.overlayButtonDanger}`}
                  onClick={handleDelete}
                  aria-label={`Delete ${title}`}
                >
                  Delete
                </button>
                <EditComic className={styles.overlayButton}>Edit</EditComic>
              </div>
            ) : null}

            <CopyableSpan
              textToCopy={id}
              textToShow={`ID ${id}`}
              className={styles.idChip}
              ariaLabel={`Copy comic ID ${id}`}
            />
          </ComicCover>
```

- [ ] **Step 6: Keep rating visible but make it admin-editable**

Replace the rating chip with a read-only visual for non-admins:

```tsx
                <div
                  className={styles.ratingChip}
                  role={isAdmin ? 'group' : 'img'}
                  aria-label={
                    isAdmin
                      ? `Rating for ${title}`
                      : `${title} rating: ${rating} of 5`
                  }
                >
                  <span className={styles.ratingDots}>
                    {Array.from({ length: 5 }, (_, index) => {
                      const value = index + 1;
                      const active = value <= rating;
                      const dot = (
                        <span
                          className={`${styles.ratingDot}${active ? ` ${styles.ratingDotActive}` : ''}`}
                        />
                      );
                      return isAdmin ? (
                        <button
                          key={value}
                          type="button"
                          className={styles.ratingPip}
                          onClick={() => handleRate(rating === value ? 0 : value)}
                          aria-label={`Set rating to ${value} of 5`}
                          aria-pressed={active}
                          title={`${value}/5`}
                        >
                          {dot}
                        </button>
                      ) : (
                        <span
                          key={value}
                          className={styles.ratingPip}
                          aria-hidden="true"
                        >
                          {dot}
                        </span>
                      );
                    })}
                  </span>
                </div>
```

- [ ] **Step 7: Gate tracking and checkout controls**

Render the footer action lane only for admins:

```tsx
              {isAdmin ? (
                <div className={styles.actionsColumn} data-testid="comic-footer-actions">
                  {showCheckout ? (
                    <button
                      className={`${styles.actionButton} ${styles.checkoutButton} basic-button`}
                      onClick={handleCheckout}
                    >
                      Checkout
                    </button>
                  ) : (
                    <span className={styles.actionButtonPlaceholder} aria-hidden="true" />
                  )}
                  <button
                    className={`${styles.actionButton} basic-button${comic.track ? ' reverse-button' : ''}`}
                    onClick={handleTrackToggle}
                  >
                    {comic.track ? 'Untrack' : 'Track'}
                  </button>
                </div>
              ) : null}
```

- [ ] **Step 8: Prevent public cover failures from issuing writes**

In `src/frontend/components/Comics/Card/ComicCover.tsx`, extend the props and default to read-only:

```tsx
interface ComicCoverProps {
  comic: Comic;
  children: ReactNode;
  canReportFailure?: boolean;
  onCoverVisibilityChange?: (coverVisible: boolean) => void;
}

export const ComicCover = ({
  comic,
  children,
  canReportFailure = false,
  onCoverVisibilityChange,
}: ComicCoverProps) => {
```

Replace `handleCoverError` with:

```ts
  const handleCoverError = () => {
    setImageFailed(true);
    if (!canReportFailure) return;

    const failedCover = comic.cover;
    const failureKey = `${comic.id}:${failedCover}`;
    if (!failedCover || reportedFailuresRef.current.has(failureKey)) return;

    reportedFailuresRef.current.add(failureKey);
    void reportCoverVisibility(comic.id, failedCover, false)
      .then((updatedComic) => {
        if (updatedComic?.cover === failedCover) {
          onCoverVisibilityChange?.(updatedComic.cover_visible !== false);
        }
      })
      .catch((error) => {
        console.debug(error);
      });
  };
```

- [ ] **Step 9: Run all role-aware UI tests**

Run:

```bash
npm test -- \
  src/frontend/components/Comics/MainPage/MainPage.auth.test.tsx \
  src/frontend/components/Comics/MainPage/MainPage.test.tsx \
  src/frontend/components/Comics/Card/ComicCard.test.tsx
```

Expected: PASS. Public and user roles have no management rail or card mutation controls; admins retain all existing controls; both roles retain catalog content and cover fallback; only admins report broken covers.

- [ ] **Step 10: Commit the role-aware UX**

```bash
git add \
  src/frontend/components/Comics/MainPage/MainPage.tsx \
  src/frontend/components/Comics/MainPage/MainPage.auth.test.tsx \
  src/frontend/components/Comics/Card/ComicCard.tsx \
  src/frontend/components/Comics/Card/ComicCover.tsx \
  src/frontend/components/Comics/Card/ComicCard.test.tsx
git commit -m "feat(ui): show catalog controls to admins"
```

### Task 6: Document Migration, Compatibility, Verification, and Rollback

**Files:**
- Create: `docs/security-boundary.md`
- Modify: `README.md:45-49`
- Modify: `go_server/README.md:43-61,117-154`
- Modify: `docs/superpowers/specs/2026-10-08-scrape-coordination.md:6-25,63-70,86-89`
- Modify: `CLAUDE.md:145`

**Interfaces:**
- Consumes: route policy and frontend behavior completed in Tasks 1-5.
- Produces: an operator runbook with an exact Mongo role migration, fresh-login requirement, compatibility matrix, local smoke checks, and safe rollback order.
- Migration contract: no schema changes; one existing application user document is promoted to `role: "admin"` and `active: true`.
- Rollback contract: frontend may roll back independently while the protected backend remains; backend rollback must not re-expose public mutations without an external deny rule.

- [ ] **Step 1: Run the documentation contract and verify it fails**

Run:

```bash
test -f docs/security-boundary.md &&
rg -q 'Bearer-only admin operations' docs/security-boundary.md &&
rg -q 'role: "admin"' docs/security-boundary.md
```

Expected: non-zero exit because `docs/security-boundary.md` does not exist.

- [ ] **Step 2: Write the security-boundary runbook**

Create `docs/security-boundary.md` with:

````markdown
# Catalog Security Boundary

Catalog data is publicly readable. Catalog administration and scrape operations
require an access JWT in the `Authorization` header whose subject role is
`admin`.

## Route policy

Public reads:

- `GET /comics`
- `GET /comics/:id`
- `GET /comics/search/:title`

Bearer-only admin operations:

- `POST /comics`
- `PUT /comics/:id`
- `DELETE /comics/:id`
- `PATCH /comics/:id/cover-visibility`
- `PATCH /comics/:id/:merging_id`
- `PUT /comics/:id/:merging_id`
- `GET /scrape`
- `GET /scrape/status`

Admin API requests must send `Accept: application/json` and an `Authorization`
header whose value is the word `Bearer`, one space, and the access JWT returned
by login.

The HttpOnly access-token cookie remains supported for browser profile and
dashboard routes. It is intentionally not accepted by catalog administration
routes. The React app reads its access JWT from `sessionStorage`, sends the
bearer header for admin operations, clears the session on `401`, and keeps a
valid session on `403`.

## Admin role migration

No SQL or MongoDB schema migration is required. Before deploying the protected
backend, promote one existing application user. Run these commands from the
repository root with a configured `go_server/.env`; the checked-in
`go_server/local.env` is only the local-development fallback:

```bash
: "${ENV_FILE:=go_server/.env}"
test -f "$ENV_FILE" || {
  echo "Create $ENV_FILE from go_server/local.env and set the target Mongo values" >&2
  exit 2
}
set -a
source "$ENV_FILE"
set +a
: "${ADMIN_EMAIL:?Set ADMIN_EMAIL to an existing application account email}"
export ADMIN_EMAIL

mongosh "$DB_ADDR" \
  --username "$DB_USER" \
  --password "$DB_PASS" \
  --authenticationDatabase admin \
  --quiet \
  --eval '
const databaseName = process.env.DB_NAME || "comics";
const collectionName = process.env.DB_TABLE_USERS || "users";
const email = process.env.ADMIN_EMAIL;
const users = db.getSiblingDB(databaseName).getCollection(collectionName);
const result = users.updateOne(
  { email },
  { $set: { role: "admin", active: true } }
);
if (result.matchedCount !== 1) {
  print(`expected one application user for ${email}; matched ${result.matchedCount}`);
  quit(2);
}
printjson(users.findOne(
  { email },
  { projection: { _id: 0, email: 1, role: 1, active: 1 } }
));
'
```

Expected output contains the selected email, `"role" : "admin"`, and
`"active" : true`. The promoted user must sign out and sign in again. Existing
access and refresh JWTs retain the role that was present when they were issued.

## Deployment order

1. Promote and verify one application admin.
2. Deploy the backend and frontend from the same security-boundary release.
3. Sign in again as the promoted admin.
4. Run the smoke checks below.
5. Keep the previous application image available until the checks pass.

There is no catalog downtime and no SQLite/PostgreSQL migration. Public read
URLs, query parameters, pagination headers, and payloads are unchanged.

## Compatibility

- Existing read-only clients continue to work without authentication.
- Existing write clients receive `401` until they send an admin bearer JWT.
- Authenticated non-admin clients receive `403`; their session remains valid.
- `GET /scrape` and `GET /scrape/status` keep their existing methods and
  response bodies but now require an admin bearer JWT.
- Browser cookie authentication remains compatible with `/protected/*` and
  `/admin/dashboard`.
- SQLite and PostgreSQL catalog behavior is unchanged.

## Smoke checks

Run against a local or staging deployment:

The commands require `curl`, `jq`, and `mongosh`.

```bash
: "${API_URL:=https://localhost:8081}"
: "${ADMIN_EMAIL:?Set ADMIN_EMAIL to the promoted application account email}"
: "${ADMIN_PASSWORD:?Set ADMIN_PASSWORD to that account password}"

TOKEN="$(
  curl -skS "$API_URL/login" \
    -H 'Accept: application/json' \
    -H 'Content-Type: application/json' \
    --data "$(jq -nc \
      --arg email "$ADMIN_EMAIL" \
      --arg password "$ADMIN_PASSWORD" \
      '{email:$email,password:$password}')" |
  jq -er '.data.access_token'
)"

test "$(
  curl -skS -o /dev/null -w '%{http_code}' \
    "$API_URL/comics?from=0&limit=1"
)" = 200

test "$(
  curl -skS -o /dev/null -w '%{http_code}' \
    -X PUT "$API_URL/comics/2147483647" \
    -H 'Accept: application/json' \
    -H 'Content-Type: application/json' \
    --data '{"track":false}'
)" = 401

test "$(
  curl -skS -o /dev/null -w '%{http_code}' \
    "$API_URL/scrape/status" \
    -H 'Accept: application/json' \
    -H "Authorization: Bearer $TOKEN"
)" = 200

test "$(
  curl -skS -o /dev/null -w '%{http_code}' \
    -X PUT "$API_URL/comics/2147483647" \
    -H 'Accept: application/json' \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $TOKEN" \
    --data '{"track":false}'
)" = 404
```

The final `404` is expected: it proves the admin request passed both middleware
checks and reached the comic handler without changing a real record.

## Rollback

Prefer a frontend-only rollback while retaining the protected backend. Public
catalog reads continue to work; management remains unavailable until a
bearer-aware frontend is restored.

Rolling back the backend reintroduces public writes and public scrape triggers.
Before deploying an older backend, deny external `POST`, `PUT`, `PATCH`, and
`DELETE` requests under `/comics` and deny `GET /scrape` plus
`GET /scrape/status` at the ingress or reverse proxy. Remove those external
denies only after the protected backend is restored.

If the promoted account must be demoted after a full rollback, run:

```bash
: "${ENV_FILE:=go_server/.env}"
test -f "$ENV_FILE" || {
  echo "Create $ENV_FILE from go_server/local.env and set the target Mongo values" >&2
  exit 2
}
set -a
source "$ENV_FILE"
set +a
: "${ADMIN_EMAIL:?Set ADMIN_EMAIL to the account being demoted}"
export ADMIN_EMAIL

mongosh "$DB_ADDR" \
  --username "$DB_USER" \
  --password "$DB_PASS" \
  --authenticationDatabase admin \
  --quiet \
  --eval '
const databaseName = process.env.DB_NAME || "comics";
const collectionName = process.env.DB_TABLE_USERS || "users";
const email = process.env.ADMIN_EMAIL;
const result = db
  .getSiblingDB(databaseName)
  .getCollection(collectionName)
  .updateOne({ email }, { $set: { role: "user" } });
if (result.matchedCount !== 1) {
  print(`expected one application user for ${email}; matched ${result.matchedCount}`);
  quit(2);
}
printjson({ email, role: "user" });
'
```

No catalog backup or restore is needed for this authorization-only release.
````

- [ ] **Step 3: Update route summaries and scrape coordination**

Add this concise section after the Go-server overview in `README.md`:

```markdown
### Catalog security

Catalog list/get/search routes are public. Catalog create/update/delete,
merge, cover updates, manual scrape, and scrape status require an admin access
JWT sent as a bearer token. See
[`docs/security-boundary.md`](docs/security-boundary.md) for the route policy,
admin promotion, smoke checks, compatibility, and rollback.
```

Replace the route list in `go_server/README.md` with:

```markdown
## Comics REST Routes

Public:

- `GET /comics`
- `GET /comics/:id`
- `GET /comics/search/:title`

Admin bearer JWT required:

- `POST /comics`
- `PUT /comics/:id`
- `DELETE /comics/:id`
- `PATCH /comics/:id/cover-visibility`
- `PATCH /comics/:base_id/:merging_id`
- `PUT /comics/:base_id/:merging_id`
- `GET /scrape`
- `GET /scrape/status`

The merge route is registered in Gin as `/comics/:id/:merging_id` to avoid a
wildcard conflict with `/comics/:id`; the first segment is still the base comic
ID on the wire. Admin API calls require an `Authorization` bearer value
containing the access JWT; the browser cookie fallback is limited to
profile/dashboard routes. Deployment, admin promotion, compatibility, and rollback are documented in
[`../docs/security-boundary.md`](../docs/security-boundary.md).
```

In `docs/superpowers/specs/2026-10-08-scrape-coordination.md`, retain the
coordinator behavior and add this exact security contract to the goals and HTTP
contract sections:

```markdown
Manual `GET /scrape` and operational `GET /scrape/status` require an
`Authorization` bearer JWT with subject role `admin`. Automatic and CLI scrape
sources do not pass through HTTP middleware. An anonymous or cookie-only request
returns `401`; an authenticated non-admin returns `403`.
```

Update the completed Python-retirement row to say:

```markdown
| **Done** | Flask `/scrape` removed; use bearer-admin Go `GET /scrape` and `GET /scrape/status`. |
```

Replace the periodic-scrape sentence in `CLAUDE.md` with:

```markdown
- Periodic scraping: `SCRAPE_INTERVAL` on `make go-run`; bearer-admin manual `GET /scrape` and bearer-admin `GET /scrape/status` share one coordinator (see `docs/superpowers/specs/2026-10-08-scrape-coordination.md`).
```

- [ ] **Step 4: Verify documentation and full release checks**

Run:

```bash
test -f docs/security-boundary.md
rg -q 'Bearer-only admin operations' docs/security-boundary.md
rg -q 'role: "admin"' docs/security-boundary.md
rg -q 'Admin bearer JWT required' go_server/README.md
rg -q 'bearer-admin manual `GET /scrape`' CLAUDE.md
git diff --check
make check
```

Expected:

- Every `test`/`rg` command exits `0`.
- `git diff --check` prints nothing.
- `make check` ends with `check passed`.
- Go route tests prove public reads, `401` anonymous/cookie-only denials, `403` user denials, and successful admin operations.
- Frontend tests prove bearer injection, `401` session clearing, `403` session preservation, admin-only controls, and no state updates after denied writes.

- [ ] **Step 5: Commit documentation and rollout guidance**

```bash
git add \
  docs/security-boundary.md \
  README.md \
  go_server/README.md \
  docs/superpowers/specs/2026-10-08-scrape-coordination.md \
  CLAUDE.md
git commit -m "docs: document catalog security rollout"
```

## Completion Criteria

- Public list/get/search requests succeed without credentials.
- Every catalog mutation, both merge methods, cover mutation, manual scrape, and scrape status reject anonymous and cookie-only requests with `401`.
- The same admin routes reject a valid user-role bearer with `403`.
- An admin-role bearer reaches every existing handler without changing its success/error contract.
- Browser profile/dashboard cookie behavior remains intact, and redirect denials cannot execute protected handlers.
- Every React catalog write uses `authorizedFetch`; public list/search remain unauthenticated.
- Persisted auth hydrates the profile role without flashing management controls.
- Public/user UI exposes catalog content but no mutation behavior, including automatic broken-cover reporting.
- An expired token clears the session on `401`; a non-admin `403` does not sign the user out.
- The operator runbook covers admin promotion, fresh login, coordinated rollout, compatibility, smoke checks, safe rollback, and confirms there is no schema migration.
- `make check` passes.
