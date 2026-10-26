# Lifecycle and Operations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Go service drain safely, report real datastore readiness, enforce SQLite's single-process deployment contract, and deploy only after CI with verified backup, smoke, and rollback stages.

**Architecture:** The Gin router will run behind an owned `http.Server`; one lifecycle manager will mark readiness down, stop the periodic scheduler, cancel and bound active scrape work, drain HTTP, and only then close repositories. SQLite mode will hold an OS file lock for the process lifetime, while Docker, Render, deployment scripts, and documentation will all enforce one API replica and readiness-based rollout. Production operations will use SQLite's online backup API through `sqlite3`, an explicitly stopped restore path, persistent or external MongoDB, and CI-triggered deployment with automatic code-and-data rollback.

**Tech Stack:** Go 1.27.3, Gin, `net/http`, `golang.org/x/sys/unix`, SQLite, MongoDB 7/Atlas, Bash, Docker Compose, Render Blueprint, GitHub Actions.

## Global Constraints

- Keep the Go modular monolith as the production service.
- Keep SQLite as the default catalog store and MongoDB as the user/auth store.
- Keep PostgreSQL as an optional catalog backend and scale-up path.
- Do not introduce Redis, a queue, or another production service.
- SQLite mode runs exactly one API replica and one periodic scheduler.
- PostgreSQL mode must also remain at one scheduler until a database-backed scrape lease exists.
- `/health` is process liveness and never depends on a datastore.
- `/ready` checks both MongoDB and the configured catalog store and gates traffic and deployment success.
- Shutdown stops new traffic, cancels scheduling, bounds scrape drain, drains HTTP, and closes repositories last.
- A verified SQLite backup must exist before a destructive schema or data operation.
- SQLite restore must refuse to run unless the API is stopped.
- SSH production mounts `/home/dev/comics-data`; the tracked `db/comics.db` is a seed and local-development artifact, not the mutable production file.
- Production secrets live in Render secret environment values or the mode-`0600` host file `go_server/.env.production`; they are never committed or echoed.
- Keep public catalog reads and the admin mutation policy defined by Workstream 1 unchanged.
- Render's persistent disk is runtime-only: build jobs, pre-deploy commands, and one-off jobs cannot access `/data`.
- Render services with a persistent disk have a fixed 30-second shutdown window, so the planned 15-second drain plus 10-second repository-close budget must remain below 30 seconds.

## Dependencies and Boundaries

- Land Workstream 1 route authorization before using an authenticated mutation as any later smoke test.
- Land Workstreams 2 and 3 before deploying destructive discovery/schema repair; this plan's verified backup and rollback tooling must be present first.
- Workstream 5 may enrich scrape outcomes later, but it must preserve the idempotent `Coordinator.Shutdown(context.Context) error` lifecycle entry point and `ErrCoordinatorClosed`.
- Workstream 6 later replaces `Application.UserRepo` with the aggregate
  `Application.AuthStore`. If implementing or rebasing this plan after that
  change, substitute `authStore`/`app.AuthStore` anywhere Task 1 or Task 4
  shows `userRepo`/`app.UserRepo`; the readiness and close-order contracts are
  unchanged.
- Workstream 9 may add readiness and shutdown metrics later, but it must preserve the endpoint semantics and lifecycle ordering established here.
- Do not change scraper result semantics, auth policy, catalog transaction semantics, or PostgreSQL scaling behavior in this workstream.

## File Map

Create:

- `go_server/internal/health/health_test.go` — liveness/readiness behavior.
- `go_server/internal/singleinstance/lock.go` — Unix advisory lock for SQLite mode.
- `go_server/internal/singleinstance/lock_test.go` — lock exclusion and release tests.
- `go_server/bootstrap/app_test.go` — resource-close ordering and error aggregation.
- `go_server/internal/lifecycle/shutdown.go` — bounded shutdown orchestration.
- `go_server/internal/lifecycle/shutdown_test.go` — real HTTP drain and timeout tests.
- `go_server/cmd/server/main_test.go` — HTTP server configuration test.
- `ops/sqlite-backup.sh` — verified SQLite `.backup`.
- `ops/sqlite-restore.sh` — stopped-service, atomic restore with a pre-restore safety copy.
- `ops/start-server.sh` — optional one-shot restore before the Go process opens SQLite.
- `ops/postdeploy-smoke.sh` — retrying liveness, readiness, and public catalog smoke.
- `ops/deploy-production.sh` — stopped backup, candidate start, smoke, and automatic rollback.
- `ops/tests/operations_test.sh` — executable backup/restore/startup-restore tests.
- `ops/tests/runtime_config_test.sh` — Docker/Render/dev-stack policy assertions.
- `ops/tests/deploy_config_test.sh` — CI gate and deploy-script assertions.
- `render.yaml` — one-instance, readiness-gated Render service declaration.
- `docs/operations.md` — canonical production runbook.

Modify:

- `go_server/internal/health/health.go` — synchronous datastore readiness and drain state.
- `go_server/api/route/route.go` — return the readiness checker to the process lifecycle.
- `go_server/grpc/config.go` and `go_server/grpc/metrics.go` — compile against the non-background health checker.
- `go_server/bootstrap/env.go` and `go_server/bootstrap/sanitize_test.go` — readiness/shutdown timeout configuration.
- `go_server/internal/scrape/coordinator.go` and `coordinator_test.go` — reject new work, cancel active work, and wait.
- `go_server/internal/scrape/periodic.go` and `periodic_test.go` — stoppable scheduler with a completion signal.
- `go_server/bootstrap/app.go` — hold the SQLite lock and return close errors.
- `go_server/cmd/scrape/main.go` — signal-aware one-shot cleanup.
- `go_server/cmd/server/main.go` — owned `http.Server` and lifecycle manager.
- `go_server/go.mod` and `go_server/go.sum` — make `golang.org/x/sys` a direct dependency.
- `Makefile` — operational tests, verified backup/restore, smoke, and Docker env selection.
- `Dockerfile` and `go_server/Dockerfile` — readiness health checks and root-image recovery tools.
- `docker-compose.yaml` and `go_server/docker-compose.yaml` — readiness, shutdown grace, Mongo volume, and one-instance defaults.
- `go_server/.air.toml` and `go_server/.air.plain.toml` — preserve the shutdown window during hot reload.
- `dev/stack.sh` and `dev/README.md` — wait on `/ready`, not a fixed sleep.
- `.github/workflows/ci.yml` — contract and operational gates.
- `.github/workflows/ci-cd.yml` — deploy only after successful `main` CI.
- `.gitignore` — ignore production env and generated database backups.
- `docs/deploy-render.md`, `README.md`, and `go_server/README.md` — operational contract and recovery links.

---

### Task 1: Make Liveness and Readiness Semantically Correct

**Files:**

- Create: `go_server/internal/health/health_test.go`
- Modify: `go_server/internal/health/health.go`
- Modify: `go_server/bootstrap/env.go`
- Modify: `go_server/api/route/route.go`
- Modify: `go_server/grpc/config.go`
- Modify: `go_server/grpc/metrics.go`

**Interfaces:**

- Produces: `health.NewChecker(timeout time.Duration, dependencies ...health.Pinger) *health.Checker`
- Produces: `(*health.Checker).SetDraining()`
- Produces: `(*health.Checker).Ready(context.Context) error`
- Produces: `route.Setup(env *bootstrap.Env, userRepo domain.UserStore, comics domain.ComicUseCase, g *gin.Engine) (*health.Checker, error)`
- Produces: `bootstrap.Env.ReadinessTimeout time.Duration`

- [ ] **Step 1: Write failing liveness/readiness tests**

Create `go_server/internal/health/health_test.go`:

```go
package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type pingerFunc func(context.Context) error

func (f pingerFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

func TestLivenessDoesNotTouchDependencies(t *testing.T) {
	var calls atomic.Int32
	checker := NewChecker(20*time.Millisecond, pingerFunc(func(context.Context) error {
		calls.Add(1)
		return errors.New("database unavailable")
	}))

	recorder := httptest.NewRecorder()
	checker.LivenessHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if recorder.Body.String() != `{"status":"UP"}` {
		t.Fatalf("unexpected body %q", recorder.Body.String())
	}
	if calls.Load() != 0 {
		t.Fatalf("liveness pinged dependencies %d times", calls.Load())
	}
}

func TestReadinessChecksEveryDependency(t *testing.T) {
	var calls atomic.Int32
	checker := NewChecker(
		20*time.Millisecond,
		pingerFunc(func(context.Context) error { calls.Add(1); return nil }),
		pingerFunc(func(context.Context) error { calls.Add(1); return nil }),
	)

	recorder := httptest.NewRecorder()
	checker.ReadinessHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected two pings, got %d", calls.Load())
	}
}

func TestReadinessFailsWhenDependencyFails(t *testing.T) {
	checker := NewChecker(20*time.Millisecond, pingerFunc(func(context.Context) error {
		return errors.New("catalog unavailable")
	}))

	recorder := httptest.NewRecorder()
	checker.ReadinessHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
	if recorder.Body.String() != `{"status":"DOWN"}` {
		t.Fatalf("unexpected body %q", recorder.Body.String())
	}
}

func TestReadinessFailsImmediatelyWhileDraining(t *testing.T) {
	var calls atomic.Int32
	checker := NewChecker(20*time.Millisecond, pingerFunc(func(context.Context) error {
		calls.Add(1)
		return nil
	}))
	checker.SetDraining()

	recorder := httptest.NewRecorder()
	checker.ReadinessHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
	if calls.Load() != 0 {
		t.Fatalf("draining readiness pinged dependencies %d times", calls.Load())
	}
}

func TestReadinessBoundsDependencyPing(t *testing.T) {
	checker := NewChecker(20*time.Millisecond, pingerFunc(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}))

	started := time.Now()
	recorder := httptest.NewRecorder()
	checker.ReadinessHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
	if elapsed := time.Since(started); elapsed > 200*time.Millisecond {
		t.Fatalf("readiness exceeded bound: %s", elapsed)
	}
}
```

- [ ] **Step 2: Run the tests and verify the old checker fails the new contract**

Run:

```bash
cd go_server
go test ./internal/health -run 'Test(Liveness|Readiness)' -count=1
```

Expected: FAIL because `NewChecker` and `SetDraining` do not exist and the current checker starts ready before any datastore ping.

- [ ] **Step 3: Replace the background cache with synchronous bounded readiness**

Replace `go_server/internal/health/health.go` with:

```go
package health

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

const defaultReadinessTimeout = 2 * time.Second

var (
	statusUP    = []byte(`{"status":"UP"}`)
	statusDOWN  = []byte(`{"status":"DOWN"}`)
	ErrDraining = errors.New("service is draining")
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Checker struct {
	dependencies []Pinger
	timeout      time.Duration
	draining     atomic.Bool
}

func NewChecker(timeout time.Duration, dependencies ...Pinger) *Checker {
	if timeout <= 0 {
		timeout = defaultReadinessTimeout
	}
	return &Checker{
		dependencies: dependencies,
		timeout:      timeout,
	}
}

func (h *Checker) SetDraining() {
	h.draining.Store(true)
}

func (h *Checker) Ready(parent context.Context) error {
	if h.draining.Load() {
		return ErrDraining
	}
	ctx, cancel := context.WithTimeout(parent, h.timeout)
	defer cancel()
	for index, dependency := range h.dependencies {
		if err := dependency.Ping(ctx); err != nil {
			return fmt.Errorf("readiness dependency %d: %w", index+1, err)
		}
	}
	return nil
}

func (h *Checker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeStatus(w, http.StatusOK, statusUP)
	}
}

func (h *Checker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		if err := h.Ready(request.Context()); err != nil {
			log.Warn().Err(err).Msg("readiness check failed")
			w.Header().Set("Retry-After", "1")
			writeStatus(w, http.StatusServiceUnavailable, statusDOWN)
			return
		}
		writeStatus(w, http.StatusOK, statusUP)
	}
}

func writeStatus(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		log.Error().Err(err).Msg("failed to write health response")
	}
}
```

- [ ] **Step 4: Make the readiness timeout explicit configuration**

In `go_server/bootstrap/env.go`, add:

```go
ReadinessTimeout time.Duration `mapstructure:"READINESS_TIMEOUT"`
```

next to `InitCtxTimeout`, add:

```go
viper.SetDefault("READINESS_TIMEOUT", 2*time.Second)
```

in `setDefaults`, and add `"READINESS_TIMEOUT"` to `bindEnvKeys`.

Change `route.Setup` to return a checker, construct it with both stores, and pass it to `metricsRouter`:

```go
func Setup(
	env *bootstrap.Env,
	userRepo domain.UserStore,
	comics domain.ComicUseCase,
	g *gin.Engine,
) (*health.Checker, error) {
	ready := health.NewChecker(env.ReadinessTimeout, userRepo, comics)
```

Replace the current call with:

```go
metricsRouter(ready, publicRouter)
```

Add this as the final statement in `Setup`:

```go
return ready, nil
```

Replace `metricsRouter` and delete `compositePinger`:

```go
func metricsRouter(ready *health.Checker, group *gin.RouterGroup) {
	prometheus.MustRegister(collectors.NewBuildInfoCollector())
	group.GET("/metrics", gin.WrapH(promhttp.Handler()))
	group.GET("/health", gin.WrapH(ready.LivenessHandler()))
	group.GET("/ready", gin.WrapH(ready.ReadinessHandler()))
}
```

Remove the now-unused `context` import from `route.go`.

- [ ] **Step 5: Keep the non-production gRPC package compiling**

In `go_server/grpc/config.go`, replace the constructor with:

```go
healthChecker := health.NewChecker(2*time.Second, comicsRepo.Client())
```

Delete `s.healthChecker.Stop()` from `Shutdown`.

In `go_server/grpc/metrics.go`, delete the `s.healthChecker.Start()` block. The handlers remain unchanged.

- [ ] **Step 6: Run focused and package tests**

Run:

```bash
cd go_server
gofmt -w internal/health/health.go internal/health/health_test.go bootstrap/env.go api/route/route.go grpc/config.go grpc/metrics.go
go test ./internal/health ./api/route ./grpc/... -count=1
```

Expected: PASS. `/health` stays 200 when stores fail; `/ready` returns 503 on Mongo/catalog failure or drain.

- [ ] **Step 7: Commit**

```bash
git add go_server/internal/health/health.go go_server/internal/health/health_test.go \
  go_server/bootstrap/env.go go_server/api/route/route.go \
  go_server/grpc/config.go go_server/grpc/metrics.go
git commit -m "feat: make readiness datastore aware"
```

---

### Task 2: Add Bounded Scheduler and Scrape Shutdown

**Files:**

- Modify: `go_server/internal/scrape/coordinator.go`
- Modify: `go_server/internal/scrape/coordinator_test.go`
- Modify: `go_server/internal/scrape/periodic.go`
- Modify: `go_server/internal/scrape/periodic_test.go`

**Interfaces:**

- Produces: `scrape.ErrCoordinatorClosed`
- Produces: idempotent `(*scrape.Coordinator).Shutdown(context.Context) error`
- Produces: `(*scrape.PeriodicScheduler).Stop(context.Context) error`

- [ ] **Step 1: Write failing coordinator shutdown tests**

Append to `coordinator_test.go`:

```go
func TestCoordinatorStopCancelsActiveRunAndRejectsNewWork(t *testing.T) {
	started := make(chan struct{})
	runErr := make(chan error, 1)
	coord := NewCoordinatorWithRunner(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})

	go func() {
		runErr <- coord.Run(context.Background(), SourceAutomatic)
	}()
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := coord.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := <-runErr; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled run, got %v", err)
	}
	if err := coord.Run(context.Background(), SourceManual); !errors.Is(err, ErrCoordinatorClosed) {
		t.Fatalf("expected closed error, got %v", err)
	}
}

func TestCoordinatorStopIsBoundedWhenRunnerIgnoresCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	coord := NewCoordinatorWithRunner(func(context.Context) error {
		close(started)
		<-release
		return nil
	})
	go func() {
		_ = coord.Run(context.Background(), SourceAutomatic)
	}()
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := coord.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	close(release)
}
```

- [ ] **Step 2: Run and confirm failure**

```bash
cd go_server
go test ./internal/scrape -run 'TestCoordinatorStop' -count=1
```

Expected: FAIL because the coordinator cannot reject or cancel work during shutdown.

- [ ] **Step 3: Track the active run and implement stop**

Add to `Coordinator`:

```go
accepting    bool
activeCancel context.CancelFunc
activeDone   chan struct{}
```

Add the sentinel:

```go
var ErrCoordinatorClosed = errors.New("scrape coordinator is shutting down")
```

Construct coordinators as accepting:

```go
func NewCoordinatorWithRunner(run func(context.Context) error) *Coordinator {
	return &Coordinator{run: run, accepting: true}
}
```

Replace `Run`, `begin`, and `end` with:

```go
func (c *Coordinator) Run(parent context.Context, source Source) error {
	ctx, err := c.begin(parent, source)
	if err != nil {
		return err
	}
	defer c.end(source)
	return c.run(ctx)
}

func (c *Coordinator) begin(parent context.Context, source Source) (context.Context, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.accepting {
		return nil, ErrCoordinatorClosed
	}
	if c.busy {
		return nil, &ErrScrapeInProgress{Running: c.busySource}
	}
	ctx, cancel := context.WithCancel(parent)
	c.busy = true
	c.busySource = source
	c.activeCancel = cancel
	c.activeDone = make(chan struct{})
	return ctx, nil
}

func (c *Coordinator) end(source Source) {
	c.mu.Lock()
	cancel := c.activeCancel
	done := c.activeDone
	c.busy = false
	c.busySource = ""
	c.activeCancel = nil
	c.activeDone = nil
	c.lastCompletedAt = time.Now()
	c.lastCompletedSource = source
	c.mu.Unlock()
	cancel()
	close(done)
}
```

Replace polling `WaitUntilIdle` and add idempotent `Shutdown`:

```go
func (c *Coordinator) WaitUntilIdle(ctx context.Context) error {
	c.mu.Lock()
	done := c.activeDone
	c.mu.Unlock()
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Coordinator) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	c.accepting = false
	cancel := c.activeCancel
	done := c.activeDone
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
```

- [ ] **Step 4: Write failing scheduler stop tests**

Append to `periodic_test.go`:

```go
func TestPeriodicSchedulerStopCancelsAndWaits(t *testing.T) {
	started := make(chan struct{})
	coord := NewCoordinatorWithRunner(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	scheduler := NewPeriodicScheduler(PeriodicConfig{Interval: time.Hour}, coord)
	scheduler.Start(context.Background())
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := scheduler.Stop(ctx); err != nil {
		t.Fatalf("stop scheduler: %v", err)
	}
}

func TestPeriodicSchedulerStopIsIdempotent(t *testing.T) {
	coord := NewCoordinatorWithRunner(func(context.Context) error { return nil })
	scheduler := NewPeriodicScheduler(PeriodicConfig{Interval: 0}, coord)
	scheduler.Start(context.Background())

	if err := scheduler.Stop(context.Background()); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if err := scheduler.Stop(context.Background()); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}
```

- [ ] **Step 5: Make the scheduler own a cancel function and done channel**

Add `sync` to the imports and these fields:

```go
mu     sync.Mutex
cancel context.CancelFunc
done   chan struct{}
```

Replace `Start` and add `Stop`:

```go
func (s *PeriodicScheduler) Start(parent context.Context) {
	if s.cfg.Interval <= 0 || s.coordinator == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	go func() {
		defer close(done)
		s.loop(ctx)
	}()
}

func (s *PeriodicScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.mu.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
```

- [ ] **Step 6: Run race-enabled scrape tests**

```bash
cd go_server
gofmt -w internal/scrape/coordinator.go internal/scrape/coordinator_test.go \
  internal/scrape/periodic.go internal/scrape/periodic_test.go
go test -race ./internal/scrape -count=1
```

Expected: PASS with no race report. Active work sees `context.Canceled`; shutdown returns `context.DeadlineExceeded` when a runner ignores cancellation.

- [ ] **Step 7: Commit**

```bash
git add go_server/internal/scrape/coordinator.go go_server/internal/scrape/coordinator_test.go \
  go_server/internal/scrape/periodic.go go_server/internal/scrape/periodic_test.go
git commit -m "feat: bound scheduler and scrape shutdown"
```

---

### Task 3: Enforce One SQLite Process and Close Resources Reliably

**Files:**

- Create: `go_server/internal/singleinstance/lock.go`
- Create: `go_server/internal/singleinstance/lock_test.go`
- Create: `go_server/bootstrap/app_test.go`
- Modify: `go_server/bootstrap/app.go`
- Modify: `go_server/cmd/scrape/main.go`
- Modify: `go_server/go.mod`
- Modify: `go_server/go.sum`

**Interfaces:**

- Produces: `singleinstance.Acquire(databasePath string) (*singleinstance.Lock, error)`
- Produces: `(*singleinstance.Lock).Close(context.Context) error`
- Changes: `(*bootstrap.Application).Close(context.Context) error`

- [ ] **Step 1: Write the failing exclusive-lock test**

Create `go_server/internal/singleinstance/lock_test.go`:

```go
package singleinstance

import (
	"context"
	"path/filepath"
	"testing"
)

func TestAcquireRejectsSecondSQLiteProcessAndReleases(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "comics.db")
	first, err := Acquire(databasePath)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if _, err := Acquire(databasePath); err == nil {
		t.Fatal("expected second acquire to fail")
	}
	if err := first.Close(context.Background()); err != nil {
		t.Fatalf("close first lock: %v", err)
	}
	second, err := Acquire(databasePath)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	if err := second.Close(context.Background()); err != nil {
		t.Fatalf("close second lock: %v", err)
	}
}
```

Run:

```bash
cd go_server
go test ./internal/singleinstance -count=1
```

Expected: FAIL because the package does not exist.

- [ ] **Step 2: Implement the Unix advisory lock**

Create `go_server/internal/singleinstance/lock.go`:

```go
package singleinstance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

type Lock struct {
	mu     sync.Mutex
	file   *os.File
	closed bool
}

func Acquire(databasePath string) (*Lock, error) {
	lockPath := databasePath + ".instance.lock"
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open SQLite instance lock %q: %w", lockPath, err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf(
			"SQLite catalog %q is already owned by another process; SQLite mode supports exactly one API or CLI process: %w",
			databasePath,
			err,
		)
	}
	if err := file.Truncate(0); err != nil {
		_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
		_ = file.Close()
		return nil, fmt.Errorf("truncate SQLite instance lock: %w", err)
	}
	if _, err := fmt.Fprintf(file, "pid=%d\n", os.Getpid()); err != nil {
		_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
		_ = file.Close()
		return nil, fmt.Errorf("write SQLite instance lock: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
		_ = file.Close()
		return nil, fmt.Errorf("sync SQLite instance lock: %w", err)
	}
	return &Lock{file: file}, nil
}

func (l *Lock) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	unlockErr := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	closeErr := l.file.Close()
	return errors.Join(unlockErr, closeErr)
}
```

This deliberately supports the repository's macOS development and Linux production targets. Do not add a no-op Windows fallback that would weaken the invariant.

- [ ] **Step 3: Write failing application-close tests**

Create `go_server/bootstrap/app_test.go`:

```go
package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestApplicationCloseRunsEveryShutterInOrderAndJoinsErrors(t *testing.T) {
	firstErr := errors.New("first close failed")
	thirdErr := errors.New("third close failed")
	order := []int{}
	app := Application{
		Shutters: []func(context.Context) error{
			func(context.Context) error { order = append(order, 1); return firstErr },
			func(context.Context) error { order = append(order, 2); return nil },
			func(context.Context) error { order = append(order, 3); return thirdErr },
		},
	}

	err := app.Close(context.Background())

	if !errors.Is(err, firstErr) || !errors.Is(err, thirdErr) {
		t.Fatalf("expected joined errors, got %v", err)
	}
	if !reflect.DeepEqual(order, []int{1, 2, 3}) {
		t.Fatalf("unexpected close order %v", order)
	}
}
```

- [ ] **Step 4: Acquire the lock before opening SQLite and release it after the catalog**

In `bootstrap/app.go`, import `errors` and `comics/internal/singleinstance`.

Change `newComicStore` to return a guard:

```go
func newComicStore(
	env *Env,
) (domain.ComicUseCase, *scrape.Coordinator, Closable, error) {
	var (
		repo  domain.ComicRepository
		guard Closable
		err   error
	)
	switch strings.ToLower(strings.TrimSpace(env.ComicsDBDriver)) {
	case "", "sqlite":
		guard, err = singleinstance.Acquire(env.ComicsSQLitePath)
		if err != nil {
			return nil, nil, nil, err
		}
		repo, err = comicrepo.NewSQLiteComicRepository(env.ComicsSQLitePath)
	case "postgres", "postgresql":
		repo, err = comicrepo.NewPostgresComicRepository(env.ComicsPostgresURL)
	default:
		return nil, nil, nil, fmt.Errorf("unknown comics DB driver %q", env.ComicsDBDriver)
	}
	if err != nil {
		if guard != nil {
			_ = guard.Close(context.Background())
		}
		return nil, nil, nil, err
	}
	coord := scrape.NewCoordinator(repo)
	return usecase.NewComicService(repo, coord), coord, guard, nil
}
```

In `MustLoadApp`, receive the guard and build shutters in this order:

```go
comicService, scrapeCoord, instanceGuard, err := newComicStore(env)
if err != nil {
	log.Fatal().Err(err).Msg("Failed to initialize comic store")
}

shutters := []func(context.Context) error{
	comicServiceCloser{comicService}.Close,
}
if instanceGuard != nil {
	shutters = append(shutters, instanceGuard.Close)
}
shutters = append(shutters, userRepo.Close, loggerClose)
```

Set `Shutters: shutters` in the returned `Application`.

Replace `Application.Close` with:

```go
func (app *Application) Close(ctx context.Context) error {
	closeErrors := make([]error, 0, len(app.Shutters))
	for _, shutter := range app.Shutters {
		if err := shutter(ctx); err != nil {
			log.Error().Err(err).Msg("Error during shutdown")
			closeErrors = append(closeErrors, err)
		}
	}
	return errors.Join(closeErrors...)
}
```

- [ ] **Step 5: Close the one-shot scraper cleanly**

Replace `cmd/scrape/main.go` with:

```go
// One-shot publisher scrape (make go-scrape).
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"comics/bootstrap"
	"comics/internal/scrape"

	"github.com/rs/zerolog/log"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	app := bootstrap.MustLoadApp(ctx)
	exitCode := 0
	if err := app.ScrapeCoord.Run(ctx, scrape.SourceCLI); err != nil {
		switch {
		case scrape.IsScrapeInProgress(err):
			log.Warn().Err(err).Msg("scrape skipped")
			exitCode = 2
		case errors.Is(err, scrape.ErrCoordinatorClosed):
			log.Warn().Err(err).Msg("scrape stopped")
			exitCode = 1
		default:
			log.Error().Err(err).Msg("scrape failed")
			exitCode = 1
		}
	} else {
		log.Info().Msg("scrape finished")
	}

	closeCtx, cancelClose := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelClose()
	if err := app.Close(closeCtx); err != nil {
		log.Error().Err(err).Msg("failed to close scrape application")
		if exitCode == 0 {
			exitCode = 1
		}
	}
	return exitCode
}
```

Preserve exit code `2` for an overlapping scrape and exit code `1` for canceled, stopped, failed, or uncleanly closed work.

Workstream 5 Task 7 later replaces the result mapping in this file with its
structured `RunAndWait` exit-code matrix. That replacement must retain this
signal context, bounded `app.Close`, SQLite single-instance behavior, and
`ErrCoordinatorClosed` handling.

- [ ] **Step 6: Tidy and test**

```bash
cd go_server
go mod tidy
gofmt -w internal/singleinstance/lock.go internal/singleinstance/lock_test.go \
  bootstrap/app.go bootstrap/app_test.go cmd/scrape/main.go
go test -race ./internal/singleinstance ./bootstrap ./cmd/scrape -count=1
go build ./cmd/server ./cmd/scrape
```

Expected: PASS. `golang.org/x/sys` moves from indirect to direct in `go.mod`; a second SQLite process gets a clear startup error, while PostgreSQL mode acquires no file lock.

- [ ] **Step 7: Commit**

```bash
git add go_server/internal/singleinstance go_server/bootstrap/app.go \
  go_server/bootstrap/app_test.go go_server/cmd/scrape/main.go \
  go_server/go.mod go_server/go.sum
git commit -m "feat: guard single-process SQLite runtime"
```

---

### Task 4: Own the HTTP Server and Orchestrate Graceful Shutdown

**Files:**

- Create: `go_server/internal/lifecycle/shutdown.go`
- Create: `go_server/internal/lifecycle/shutdown_test.go`
- Create: `go_server/cmd/server/main_test.go`
- Modify: `go_server/bootstrap/env.go`
- Modify: `go_server/cmd/server/main.go`

**Interfaces:**

- Produces: `lifecycle.Manager.Shutdown(context.Context) error`
- Produces: `bootstrap.Env.ShutdownTimeout time.Duration`
- Consumes: readiness `SetDraining`, scheduler `Stop`, coordinator `Shutdown`, application `Close`

- [ ] **Step 1: Write failing lifecycle tests**

Create `go_server/internal/lifecycle/shutdown_test.go` with:

```go
package lifecycle

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

type readinessFake struct{ draining atomic.Bool }

func (f *readinessFake) SetDraining() { f.draining.Store(true) }

type stopperFake struct{ called atomic.Bool }

func (f *stopperFake) Stop(context.Context) error {
	f.called.Store(true)
	return nil
}

type scrapeStopperFake struct{ called atomic.Bool }

func (f *scrapeStopperFake) Shutdown(context.Context) error {
	f.called.Store(true)
	return nil
}

type closerFake struct {
	called *atomic.Bool
	check  func() error
}

func (f closerFake) Close(context.Context) error {
	if f.check != nil {
		if err := f.check(); err != nil {
			return err
		}
	}
	f.called.Store(true)
	return nil
}

func TestManagerDrainsHTTPBeforeClosingRepositories(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = io.WriteString(w, "ok")
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	go func() {
		_ = server.Serve(listener)
	}()
	responseDone := make(chan struct{})
	go func() {
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if requestErr == nil {
			_ = response.Body.Close()
		}
		close(responseDone)
	}()
	<-entered

	ready := &readinessFake{}
	scheduler := &stopperFake{}
	scrapes := &scrapeStopperFake{}
	var closed atomic.Bool
	manager := Manager{
		Readiness:   ready,
		HTTP:        server,
		Scheduler:   scheduler,
		Scrapes:     scrapes,
		Application: closerFake{called: &closed},
		DrainTimeout: time.Second,
		CloseTimeout: time.Second,
	}
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- manager.Shutdown(context.Background())
	}()

	select {
	case err := <-shutdownDone:
		t.Fatalf("shutdown returned before request drained: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	if !ready.draining.Load() {
		t.Fatal("readiness was not marked draining")
	}
	close(release)
	if err := <-shutdownDone; err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	<-responseDone
	if !scheduler.called.Load() || !scrapes.called.Load() || !closed.Load() {
		t.Fatal("expected scheduler, scrape, and repository shutdown")
	}
}

type blockingHTTP struct{ forced atomic.Bool }

func (h *blockingHTTP) Shutdown(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (h *blockingHTTP) Close() error {
	h.forced.Store(true)
	return nil
}

func TestManagerForcesHTTPClosedAfterDrainDeadlineAndStillClosesApp(t *testing.T) {
	server := &blockingHTTP{}
	var closed atomic.Bool
	manager := Manager{
		HTTP:         server,
		Application:  closerFake{called: &closed},
		DrainTimeout: 20 * time.Millisecond,
		CloseTimeout: time.Second,
	}

	err := manager.Shutdown(context.Background())

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline error, got %v", err)
	}
	if !server.forced.Load() {
		t.Fatal("expected forced HTTP close")
	}
	if !closed.Load() {
		t.Fatal("expected application close after forced drain")
	}
}
```

- [ ] **Step 2: Run and confirm the package is absent**

```bash
cd go_server
go test ./internal/lifecycle -count=1
```

Expected: FAIL because `Manager` does not exist.

- [ ] **Step 3: Implement lifecycle ordering and independent close budget**

Create `go_server/internal/lifecycle/shutdown.go`:

```go
package lifecycle

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type ReadinessDrainer interface {
	SetDraining()
}

type HTTPServer interface {
	Shutdown(context.Context) error
	Close() error
}

type Scheduler interface {
	Stop(context.Context) error
}

type ScrapeCoordinator interface {
	Shutdown(context.Context) error
}

type Application interface {
	Close(context.Context) error
}

type Manager struct {
	Readiness    ReadinessDrainer
	HTTP         HTTPServer
	Scheduler    Scheduler
	Scrapes      ScrapeCoordinator
	Application  Application
	DrainTimeout time.Duration
	CloseTimeout time.Duration
}

func (m Manager) Shutdown(parent context.Context) error {
	if m.Readiness != nil {
		m.Readiness.SetDraining()
	}
	base := context.WithoutCancel(parent)
	drainCtx, cancelDrain := context.WithTimeout(base, m.DrainTimeout)
	drainErr := runConcurrent(
		drainCtx,
		func(ctx context.Context) error {
			if m.HTTP == nil {
				return nil
			}
			err := m.HTTP.Shutdown(ctx)
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		},
		func(ctx context.Context) error {
			if m.Scheduler == nil {
				return nil
			}
			return m.Scheduler.Stop(ctx)
		},
		func(ctx context.Context) error {
			if m.Scrapes == nil {
				return nil
			}
			return m.Scrapes.Shutdown(ctx)
		},
	)
	cancelDrain()

	var forceErr error
	if errors.Is(drainErr, context.DeadlineExceeded) && m.HTTP != nil {
		forceErr = m.HTTP.Close()
	}

	closeCtx, cancelClose := context.WithTimeout(base, m.CloseTimeout)
	var closeErr error
	if m.Application != nil {
		closeErr = m.Application.Close(closeCtx)
	}
	cancelClose()
	return errors.Join(drainErr, forceErr, closeErr)
}

func runConcurrent(
	ctx context.Context,
	operations ...func(context.Context) error,
) error {
	results := make(chan error, len(operations))
	for _, operation := range operations {
		go func(run func(context.Context) error) {
			results <- run(ctx)
		}(operation)
	}
	collected := make([]error, 0, len(operations)+1)
	for range operations {
		select {
		case err := <-results:
			if err != nil {
				collected = append(collected, err)
			}
		case <-ctx.Done():
			collected = append(collected, ctx.Err())
			return errors.Join(collected...)
		}
	}
	return errors.Join(collected...)
}
```

- [ ] **Step 4: Add explicit shutdown configuration**

In `Env`, add:

```go
ShutdownTimeout time.Duration `mapstructure:"SHUTDOWN_TIMEOUT"`
```

Add the default and binding:

```go
viper.SetDefault("SHUTDOWN_TIMEOUT", 15*time.Second)
```

```go
"SHUTDOWN_TIMEOUT",
```

Keep `INIT_TIMEOUT=10s` as the repository-close budget. Add direct `Env.Validate` checks that all three lifecycle timeout values are greater than zero, returning:

```go
fmt.Errorf("READINESS_TIMEOUT, SHUTDOWN_TIMEOUT, and INIT_TIMEOUT must be greater than zero")
```

Add `time` and `comics/internal/repo` to the imports in `bootstrap/sanitize_test.go`, then append:

```go
func validLifecycleEnv() *Env {
	return &Env{
		AddressHTTP:       "127.0.0.1",
		PortHTTP:          "8081",
		HostURL:           "localhost:8081",
		ComicsDBDriver:    "sqlite",
		ComicsSQLitePath:  "../db/comics.db",
		InitCtxTimeout:    10 * time.Second,
		ReadinessTimeout: 2 * time.Second,
		ShutdownTimeout:  15 * time.Second,
		DBConfig: &repo.DBConfig{
			Addr: "mongodb://localhost:27017",
			Name: "comics",
		},
		JWTConfig: &JWTConfig{
			AccessTokenSecret:  "0123456789abcdef0123456789abcdef",
			RefreshTokenSecret: "fedcba9876543210fedcba9876543210",
		},
	}
}

func TestValidateRejectsNonPositiveLifecycleTimeouts(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Env)
	}{
		{
			name:   "zero readiness",
			change: func(env *Env) { env.ReadinessTimeout = 0 },
		},
		{
			name:   "negative readiness",
			change: func(env *Env) { env.ReadinessTimeout = -time.Second },
		},
		{
			name:   "zero shutdown",
			change: func(env *Env) { env.ShutdownTimeout = 0 },
		},
		{
			name:   "negative shutdown",
			change: func(env *Env) { env.ShutdownTimeout = -time.Second },
		},
		{
			name:   "zero repository close",
			change: func(env *Env) { env.InitCtxTimeout = 0 },
		},
		{
			name:   "negative repository close",
			change: func(env *Env) { env.InitCtxTimeout = -time.Second },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := validLifecycleEnv()
			test.change(env)
			if err := env.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
```

- [ ] **Step 5: Write the failing HTTP server configuration test**

Create `go_server/cmd/server/main_test.go`:

```go
package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestNewHTTPServerOwnsTimeoutsAndHandler(t *testing.T) {
	handler := gin.New()
	server := newHTTPServer("127.0.0.1:8081", handler)

	if server.Addr != "127.0.0.1:8081" {
		t.Fatalf("unexpected address %q", server.Addr)
	}
	if server.Handler != http.Handler(handler) {
		t.Fatal("unexpected handler")
	}
	if server.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("unexpected read-header timeout %s", server.ReadHeaderTimeout)
	}
	if server.IdleTimeout != 60*time.Second {
		t.Fatalf("unexpected idle timeout %s", server.IdleTimeout)
	}
}
```

- [ ] **Step 6: Replace Gin's convenience runner with `http.Server`**

Replace `cmd/server/main.go` with:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"comics/api/route"
	"comics/bootstrap"
	_ "comics/docs"
	"comics/internal/lifecycle"
	"comics/internal/scrape"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// main serves the HTTP application.
//
//	@title						Comics API
//	@version					1.1
//	@description				Server documentation to query comics from the DB.
//	@termsOfService				http://swagger.io/terms/
//	@contact.name				Estebmaister
//	@contact.url				http://www.github.com/estebmaister
//	@contact.email				estebmaister@gmail.com
//	@license.name				Apache 2.0
//	@license.url				http://www.apache.org/licenses/LICENSE-2.0.html
//	@securityDefinitions.apikey	Bearer JWT
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space paste the JWT.
//	@host						localhost:8081
//	@BasePath					/
func main() {
	if err := run(); err != nil {
		log.Error().Err(err).Msg("server stopped with an error")
		os.Exit(1)
	}
}

func run() error {
	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	app := bootstrap.MustLoadApp(signalCtx)
	if app.Env.AppEnv.IsProduction() || app.Env.AppEnv.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())
	ready, err := route.Setup(app.Env, app.UserRepo, app.ComicService, engine)
	if err != nil {
		return fmt.Errorf("configure routes: %w", err)
	}

	scheduler := scrape.NewPeriodicScheduler(
		scrape.PeriodicConfig{Interval: app.Env.ScrapeInterval},
		app.ScrapeCoord,
	)
	scheduler.Start(signalCtx)
	if app.Env.ScrapeInterval > 0 {
		log.Info().
			Dur("interval", app.Env.ScrapeInterval).
			Msg("Background scrape scheduler enabled")
	}

	address := net.JoinHostPort(app.Env.AddressHTTP, app.Env.PortHTTP)
	server := newHTTPServer(address, engine)
	certFile := ""
	keyFile := ""
	scheme := "http"
	if app.Env.HasHTTPSTLSFiles() {
		certFile = app.Env.HTTPTLSCertFile
		keyFile = app.Env.HTTPTLSKeyFile
		scheme = "https"
	} else if app.Env.HTTPTLSCertFile != "" || app.Env.HTTPTLSKeyFile != "" {
		log.Warn().
			Str("cert", app.Env.HTTPTLSCertFile).
			Str("key", app.Env.HTTPTLSKeyFile).
			Msg("TLS certificate or key unavailable, serving HTTP instead")
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Info().Str("url", scheme+"://"+address).Msg("Serving HTTP")
		serverErrors <- serveHTTP(server, certFile, keyFile)
	}()

	var serveErr error
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr = fmt.Errorf("serve HTTP: %w", err)
		}
	case <-signalCtx.Done():
		stop()
	}

	log.Info().Msg("Shutting down application")
	manager := lifecycle.Manager{
		Readiness:    ready,
		HTTP:         server,
		Scheduler:    scheduler,
		Scrapes:      app.ScrapeCoord,
		Application:  &app,
		DrainTimeout: app.Env.ShutdownTimeout,
		CloseTimeout: app.Env.InitCtxTimeout,
	}
	shutdownErr := manager.Shutdown(signalCtx)
	return errors.Join(serveErr, shutdownErr)
}

func newHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func serveHTTP(server *http.Server, certFile, keyFile string) error {
	if certFile != "" && keyFile != "" {
		return server.ListenAndServeTLS(certFile, keyFile)
	}
	return server.ListenAndServe()
}
```

Do not defer `app.Close`; the manager owns repository closure and must run exactly once.

- [ ] **Step 7: Run lifecycle, server, and race tests**

```bash
cd go_server
gofmt -w internal/lifecycle/shutdown.go internal/lifecycle/shutdown_test.go \
  cmd/server/main.go cmd/server/main_test.go bootstrap/env.go bootstrap/sanitize_test.go
go test -race ./internal/lifecycle ./internal/scrape ./bootstrap ./cmd/server -count=1
go test ./... -count=1
go vet ./...
```

Expected: PASS. A blocked HTTP handler delays repository close until released; an ignored drain is force-closed after 15 seconds in production, then repositories receive their separate 10-second close budget.

- [ ] **Step 8: Commit**

```bash
git add go_server/internal/lifecycle go_server/cmd/server/main.go \
  go_server/cmd/server/main_test.go go_server/bootstrap/env.go \
  go_server/bootstrap/sanitize_test.go
git commit -m "feat: gracefully drain the Go server"
```

---

### Task 5: Add Verified SQLite Backup and Stopped Restore

**Files:**

- Create: `ops/sqlite-backup.sh`
- Create: `ops/sqlite-restore.sh`
- Create: `ops/start-server.sh`
- Create: `ops/tests/operations_test.sh`
- Modify: `Makefile`
- Modify: `.gitignore`

**Interfaces:**

- Produces: `make db-backup BACKUP_LABEL=manual`
- Produces: `make db-restore BACKUP=db/backups/comics_manual_20261026T140000Z.db SERVICE_STOPPED=1`
- Produces: one-shot startup restore via `SQLITE_RESTORE_FROM` and `SQLITE_RESTORE_TOKEN`

- [ ] **Step 1: Write the failing shell test**

Create `ops/tests/operations_test.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fail() {
	echo "FAIL: $*" >&2
	exit 1
}

expect_failure() {
	if "$@"; then
		fail "command unexpectedly succeeded: $*"
	fi
}

source_db="$TMP/comics.db"
backup_dir="$TMP/backups"
sqlite3 "$source_db" "CREATE TABLE comics(id INTEGER PRIMARY KEY, title TEXT); INSERT INTO comics VALUES(1, 'before');"

backup="$(
	DB_FILE="$source_db" \
	BACKUP_DIR="$backup_dir" \
	BACKUP_LABEL=test \
	BACKUP_TIMESTAMP=20261026T140000Z \
	"$ROOT/ops/sqlite-backup.sh"
)"
[[ -f "$backup" ]] || fail "backup file was not created"
[[ "$(sqlite3 "$backup" 'PRAGMA integrity_check;')" == "ok" ]] || fail "backup failed integrity check"

sqlite3 "$source_db" "UPDATE comics SET title='after' WHERE id=1;"
expect_failure env \
	DB_FILE="$source_db" \
	BACKUP="$backup" \
	COMPOSE_FILE="$TMP/no-compose.yaml" \
	"$ROOT/ops/sqlite-restore.sh"

SERVICE_STOPPED=1 \
DB_FILE="$source_db" \
BACKUP="$backup" \
BACKUP_DIR="$backup_dir" \
COMPOSE_FILE="$TMP/no-compose.yaml" \
"$ROOT/ops/sqlite-restore.sh"
[[ "$(sqlite3 "$source_db" 'SELECT title FROM comics WHERE id=1;')" == "before" ]] ||
	fail "restore did not replace catalog"

fake_bin="$TMP/fake-bin"
mkdir -p "$fake_bin"
cat > "$fake_bin/docker" <<'EOF'
#!/usr/bin/env bash
if [[ "$1" == "compose" ]]; then
	echo "running-container"
	exit 0
fi
if [[ "$1" == "inspect" ]]; then
	echo "true"
	exit 0
fi
exit 1
EOF
chmod +x "$fake_bin/docker"
touch "$TMP/compose.yaml"
expect_failure env \
	PATH="$fake_bin:$PATH" \
	SERVICE_STOPPED=1 \
	DB_FILE="$source_db" \
	BACKUP="$backup" \
	COMPOSE_FILE="$TMP/compose.yaml" \
	"$ROOT/ops/sqlite-restore.sh"

restore_source="$TMP/restore-source.db"
sqlite3 "$restore_source" "CREATE TABLE comics(id INTEGER PRIMARY KEY, title TEXT); INSERT INTO comics VALUES(1, 'startup');"
SQLITE_RESTORE_FROM="$restore_source" \
SQLITE_RESTORE_TOKEN=incident-20261026 \
COMICS_SQLITE_PATH="$source_db" \
BACKUP_DIR="$backup_dir" \
SERVER_BIN=/usr/bin/true \
"$ROOT/ops/start-server.sh"
[[ "$(sqlite3 "$source_db" 'SELECT title FROM comics WHERE id=1;')" == "startup" ]] ||
	fail "startup restore did not run"

sqlite3 "$source_db" "UPDATE comics SET title='kept' WHERE id=1;"
SQLITE_RESTORE_FROM="$restore_source" \
SQLITE_RESTORE_TOKEN=incident-20261026 \
COMICS_SQLITE_PATH="$source_db" \
BACKUP_DIR="$backup_dir" \
SERVER_BIN=/usr/bin/true \
"$ROOT/ops/start-server.sh"
[[ "$(sqlite3 "$source_db" 'SELECT title FROM comics WHERE id=1;')" == "kept" ]] ||
	fail "startup restore token ran more than once"

echo "operations tests passed"
```

Run:

```bash
bash ops/tests/operations_test.sh
```

Expected: FAIL because the operational scripts do not exist.

- [ ] **Step 2: Implement verified backup**

Create `ops/sqlite-backup.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
umask 077

die() {
	echo "sqlite-backup: $*" >&2
	exit 1
}

command -v sqlite3 >/dev/null 2>&1 || die "sqlite3 is required"

db_file="${DB_FILE:-db/comics.db}"
backup_dir="${BACKUP_DIR:-db/backups}"
label="${BACKUP_LABEL:-manual}"
timestamp="${BACKUP_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"

[[ "$label" =~ ^[A-Za-z0-9._-]+$ ]] || die "BACKUP_LABEL contains unsupported characters"
[[ "$timestamp" =~ ^[A-Za-z0-9._:-]+$ ]] || die "BACKUP_TIMESTAMP contains unsupported characters"
[[ "$db_file" != *"'"* && "$backup_dir" != *"'"* ]] || die "paths containing a single quote are unsupported"
[[ -f "$db_file" ]] || die "database not found: $db_file"

mkdir -p "$backup_dir"
destination="$backup_dir/comics_${label}_${timestamp}.db"
[[ ! -e "$destination" ]] || die "backup already exists: $destination"

temporary="${destination}.partial.$$"
trap 'rm -f "$temporary"' EXIT
sqlite3 "$db_file" ".backup '$temporary'"
integrity="$(sqlite3 "$temporary" "PRAGMA integrity_check;")"
[[ "$integrity" == "ok" ]] || die "backup integrity check failed: $integrity"
has_comics="$(sqlite3 "$temporary" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='comics';")"
[[ "$has_comics" == "1" ]] || die "backup does not contain the comics table"
mv "$temporary" "$destination"
trap - EXIT

echo "verified SQLite backup: $destination" >&2
printf '%s\n' "$destination"
```

- [ ] **Step 3: Implement guarded, atomic restore**

Create `ops/sqlite-restore.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

die() {
	echo "sqlite-restore: $*" >&2
	exit 1
}

command -v sqlite3 >/dev/null 2>&1 || die "sqlite3 is required"

db_file="${DB_FILE:-db/comics.db}"
backup="${BACKUP:-${1:-}}"
backup_dir="${BACKUP_DIR:-db/backups}"
compose_file="${COMPOSE_FILE:-docker-compose.yaml}"
compose_service="${COMPOSE_SERVICE:-comics_server}"

[[ "${SERVICE_STOPPED:-0}" == "1" ]] ||
	die "refusing live restore; stop the API and set SERVICE_STOPPED=1"
[[ -n "$backup" ]] || die "set BACKUP to a SQLite .db file"
[[ -f "$backup" ]] || die "backup not found: $backup"

if command -v docker >/dev/null 2>&1 && [[ -f "$compose_file" ]]; then
	container_id="$(docker compose -f "$compose_file" ps -q "$compose_service")" ||
		die "cannot determine whether $compose_service is running"
	if [[ -n "$container_id" ]]; then
		running="$(docker inspect --format '{{.State.Running}}' "$container_id")" ||
			die "cannot inspect $compose_service"
		[[ "$running" != "true" ]] ||
			die "refusing restore while $compose_service is running"
	fi
fi

integrity="$(sqlite3 "$backup" "PRAGMA integrity_check;")"
[[ "$integrity" == "ok" ]] || die "backup integrity check failed: $integrity"
has_comics="$(sqlite3 "$backup" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='comics';")"
[[ "$has_comics" == "1" ]] || die "backup does not contain the comics table"

mkdir -p "$(dirname "$db_file")" "$backup_dir"
if [[ -f "$db_file" ]]; then
	if ! BACKUP_LABEL=pre-restore \
		BACKUP_TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)-$$" \
		DB_FILE="$db_file" \
		BACKUP_DIR="$backup_dir" \
		"$ROOT/ops/sqlite-backup.sh" >/dev/null; then
		quarantine="$backup_dir/comics_pre-restore-unverified_$(date -u +%Y%m%dT%H%M%SZ)-$$.db"
		cp "$db_file" "$quarantine"
		[[ ! -f "${db_file}-wal" ]] || cp "${db_file}-wal" "${quarantine}-wal"
		[[ ! -f "${db_file}-shm" ]] || cp "${db_file}-shm" "${quarantine}-shm"
		chmod 0600 "$quarantine" "${quarantine}-wal" "${quarantine}-shm" 2>/dev/null || true
		echo "sqlite-restore: current database was not healthy; quarantined unverified files at $quarantine" >&2
	fi
fi

temporary="${db_file}.restore.$$"
trap 'rm -f "$temporary"' EXIT
cp "$backup" "$temporary"
chmod 0600 "$temporary"
[[ "$(sqlite3 "$temporary" "PRAGMA integrity_check;")" == "ok" ]] ||
	die "temporary restored database failed integrity check"
rm -f "${db_file}-wal" "${db_file}-shm"
mv -f "$temporary" "$db_file"
trap - EXIT
[[ "$(sqlite3 "$db_file" "PRAGMA integrity_check;")" == "ok" ]] ||
	die "restored database failed integrity check"

echo "restored $db_file from $backup" >&2
printf '%s\n' "$db_file"
```

- [ ] **Step 4: Add a one-shot pre-server restore for Render recovery**

Create `ops/start-server.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
db_file="${COMICS_SQLITE_PATH:-/data/comics.db}"
restore_from="${SQLITE_RESTORE_FROM:-}"
restore_token="${SQLITE_RESTORE_TOKEN:-}"
server_bin="${SERVER_BIN:-$ROOT/server}"

if [[ -n "$restore_from" || -n "$restore_token" ]]; then
	[[ -n "$restore_from" && -n "$restore_token" ]] || {
		echo "start-server: SQLITE_RESTORE_FROM and SQLITE_RESTORE_TOKEN must be set together" >&2
		exit 1
	}
	[[ "$restore_token" =~ ^[A-Za-z0-9._-]+$ ]] || {
		echo "start-server: invalid SQLITE_RESTORE_TOKEN" >&2
		exit 1
	}
	marker="${db_file}.restore-${restore_token}.done"
	if [[ ! -f "$marker" ]]; then
		SERVICE_STOPPED=1 \
		DB_FILE="$db_file" \
		BACKUP="$restore_from" \
		BACKUP_DIR="${BACKUP_DIR:-$(dirname "$db_file")/backups}" \
		COMPOSE_FILE="$ROOT/.no-compose-during-startup" \
		"$ROOT/ops/sqlite-restore.sh" >/dev/null
		printf 'restored=%s\n' "$restore_from" > "$marker"
		chmod 0600 "$marker"
	fi
fi

exec "$server_bin" "$@"
```

The token marker makes retries idempotent. A different incident must use a different token.

- [ ] **Step 5: Wire Make targets and ignore generated secrets/data**

Change root `Makefile`:

```make
BACKUP_DIR ?= db/backups

## @database:db-backup         Create and verify a SQLite backup under BACKUP_DIR
db-backup:
	@DB_FILE="$(DB_FILE)" BACKUP_DIR="$(BACKUP_DIR)" BACKUP_LABEL="$(or $(BACKUP_LABEL),manual)" ./ops/sqlite-backup.sh

## @database:db-restore        Restore BACKUP only after API stop and SERVICE_STOPPED=1
db-restore:
	@DB_FILE="$(DB_FILE)" BACKUP_DIR="$(BACKUP_DIR)" BACKUP="$(BACKUP)" \
		SERVICE_STOPPED="$(SERVICE_STOPPED)" COMPOSE_FILE="$(or $(COMPOSE_FILE),docker-compose.yaml)" \
		./ops/sqlite-restore.sh

## @quality:ops-test           Test backup, restore, and startup recovery scripts
ops-test:
	@bash ops/tests/operations_test.sh
```

Add `ops-test` to `.PHONY`.

Add to `.gitignore`:

```gitignore
go_server/.env.production
db/backups/*.db
```

- [ ] **Step 6: Make scripts executable and run tests**

```bash
chmod +x ops/sqlite-backup.sh ops/sqlite-restore.sh ops/start-server.sh \
  ops/tests/operations_test.sh
bash -n ops/sqlite-backup.sh ops/sqlite-restore.sh ops/start-server.sh \
  ops/tests/operations_test.sh
make ops-test
```

Expected: `operations tests passed`. Restore without `SERVICE_STOPPED=1` fails; restore against a running Compose service fails; backup and restored DB both report `ok`.

- [ ] **Step 7: Commit**

```bash
git add ops Makefile .gitignore
git commit -m "feat: verify SQLite backup and restore"
```

---

### Task 6: Align Docker, Render, Mongo, and Dev Readiness

**Files:**

- Create: `ops/tests/runtime_config_test.sh`
- Create: `render.yaml`
- Modify: `Dockerfile`
- Modify: `go_server/Dockerfile`
- Modify: `docker-compose.yaml`
- Modify: `go_server/docker-compose.yaml`
- Modify: `go_server/.air.toml`
- Modify: `go_server/.air.plain.toml`
- Modify: `dev/stack.sh`
- Modify: `dev/README.md`
- Modify: `Makefile`

**Interfaces:**

- Produces: Docker health based on `GET /ready`
- Produces: Render `autoDeployTrigger: checksPass`, `healthCheckPath: /ready`, `numInstances: 1`
- Produces: persistent `mongo_data` for local/self-hosted Mongo

- [ ] **Step 1: Write failing runtime configuration assertions**

Create `ops/tests/runtime_config_test.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

rg -q 'ENV HEALTHCHECK_PATH=/ready' Dockerfile
rg -q 'CMD \["\./ops/start-server\.sh"\]' Dockerfile
rg -q 'stop_grace_period: 30s' docker-compose.yaml
rg -q 'mongo_data:/data/db' docker-compose.yaml
rg -q 'COMICS_DATA_DIR' docker-compose.yaml
rg -q 'healthCheckPath: /ready' render.yaml
rg -q 'autoDeployTrigger: checksPass' render.yaml
rg -q 'numInstances: 1' render.yaml
rg -q 'curl.+/ready' dev/stack.sh
rg -q 'i -lt 150' dev/stack.sh
rg -q 'kill_delay = "30s"' go_server/.air.toml go_server/.air.plain.toml

COMICS_ENV_FILE=go_server/local.env \
COMICS_DATA_DIR="$ROOT/db" \
docker compose --profile local-mongo config --quiet

bash -n dev/stack.sh
echo "runtime config tests passed"
```

Run:

```bash
bash ops/tests/runtime_config_test.sh
```

Expected: FAIL because Docker still uses `/health`, Mongo has no volume, and `render.yaml` is absent.

- [ ] **Step 2: Make the root image readiness-aware and recovery-capable**

In root `Dockerfile`, replace the runtime package/copy/ownership section with:

```dockerfile
RUN apk add --no-cache bash curl sqlite ca-certificates tzdata \
	&& addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /server .
COPY go_server/templates ./templates
COPY go_server/static ./static
COPY go_server/docs ./docs
COPY config/url_switch.json ./config/url_switch.json
COPY ops ./ops

RUN chmod 0555 /app/ops/*.sh \
	&& chown -R appuser:appgroup /app

USER appuser

ENV HTTP_ADDRESS=0.0.0.0
ENV COMICS_DB_DRIVER=sqlite
ENV COMICS_SQLITE_PATH=/data/comics.db
ENV COMICS_URL_SWITCH=/app/config/url_switch.json
ENV HEALTHCHECK_PATH=/ready
```

Then use:

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
	CMD sh -c 'curl --fail --silent --show-error "http://127.0.0.1:$${HTTP_PORT:-$${PORT:-8081}}$${HEALTHCHECK_PATH}"' || exit 1

CMD ["./ops/start-server.sh"]
```

In `go_server/Dockerfile`, keep `CMD ["./server"]` but align the `HEALTHCHECK` timeout, start period, and `/ready` path.

- [ ] **Step 3: Replace root Compose with one-instance production defaults**

Use this service contract in `docker-compose.yaml`:

```yaml
name: comics

services:
  comics_server:
    container_name: comics_server
    image: ${COMICS_IMAGE:-comics-go-api:local}
    build:
      context: .
      dockerfile: Dockerfile
    env_file:
      - ${COMICS_ENV_FILE:-go_server/.env.production}
    environment:
      HTTP_ADDRESS: 0.0.0.0
      HTTP_PORT: 8081
      COMICS_DB_DRIVER: sqlite
      COMICS_SQLITE_PATH: /data/comics.db
    ports:
      - "8081:8081"
    volumes:
      - ${COMICS_DATA_DIR:-./db}:/data
    depends_on:
      mongodb:
        condition: service_healthy
        required: false
    healthcheck:
      test: ["CMD", "curl", "--fail", "--silent", "http://127.0.0.1:8081/ready"]
      interval: 10s
      timeout: 5s
      start_period: 20s
      retries: 6
    stop_grace_period: 30s
    restart: unless-stopped
    networks:
      - comics_net

  mongodb:
    profiles: ["local-mongo"]
    image: mongo:7
    container_name: comics_mongodb
    ports:
      - "127.0.0.1:27017:27017"
    volumes:
      - mongo_data:/data/db
    healthcheck:
      test: ["CMD", "mongosh", "--quiet", "--eval", "db.adminCommand('ping')"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped
    networks:
      - comics_net

  postgres:
    profiles: ["postgres"]
    container_name: comics_postgres
    image: postgres:14
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${PGUSER:-comics}
      POSTGRES_PASSWORD: ${PGPASSWORD:-comics}
      POSTGRES_DB: ${PGDATABASE:-comics}
      PGDATA: /var/lib/postgresql/data/pgdata
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "127.0.0.1:5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - comics_net

networks:
  comics_net:
    driver: bridge

volumes:
  mongo_data:
  pgdata:
```

Update `go_server/docker-compose.yaml` to mount `mongo_data:/data/db` and declare `mongo_data:` at the bottom. Its purpose remains local development.

- [ ] **Step 4: Add the exact Render Blueprint**

Create `render.yaml`:

```yaml
services:
  - type: web
    name: comic-tracker-server
    runtime: docker
    dockerfilePath: ./Dockerfile
    dockerContext: .
    autoDeployTrigger: checksPass
    healthCheckPath: /ready
    numInstances: 1
    disk:
      name: comics-data
      mountPath: /data
      sizeGB: 1
    envVars:
      - key: ENVIRONMENT
        value: production
      - key: HOST_URL
        value: comic-tracker-server.onrender.com
      - key: COMICS_DB_DRIVER
        value: sqlite
      - key: COMICS_SQLITE_PATH
        value: /data/comics.db
      - key: SCRAPE_INTERVAL
        value: 10m
      - key: READINESS_TIMEOUT
        value: 2s
      - key: SHUTDOWN_TIMEOUT
        value: 15s
      - key: INIT_TIMEOUT
        value: 10s
      - key: DB_NAME
        value: comics
      - key: CORS_ALLOWED_ORIGINS
        value: https://estebmaister.github.io
      - key: FRONTEND_URL
        value: https://estebmaister.github.io/comics
      - key: DB_ADDR
        sync: false
      - key: DB_USER
        sync: false
      - key: DB_PASS
        sync: false
      - key: ACCESS_TOKEN_SECRET
        sync: false
      - key: ACCESS_TOKEN_KEY_ID
        value: access-2026-10
      - key: REFRESH_TOKEN_SECRET
        sync: false
      - key: REFRESH_TOKEN_KEY_ID
        value: refresh-2026-10
      - key: GOOGLE_CLIENT_ID
        sync: false
      - key: GOOGLE_CLIENT_SECRET
        sync: false
```

Do not add `maxShutdownDelaySeconds`: Render does not permit changing it for a service with a persistent disk.

- [ ] **Step 5: Make the dev stack wait for readiness**

In `stop_pidfile`, change the bounded TERM wait from 30 iterations to 150:

```bash
		local i=0
		while kill -0 "$pid" 2>/dev/null && [[ $i -lt 150 ]]; do
			sleep 0.2
			i=$((i + 1))
		done
```

This gives the process the same 30-second ceiling as Docker/Render before the
existing `kill -9` fallback.

Set this value in both `go_server/.air.toml` and
`go_server/.air.plain.toml`:

```toml
kill_delay = "30s"
```

Keep `send_interrupt = true` so Air sends the signal handled by
`signal.NotifyContext`.

In `dev/stack.sh`, replace the fixed four-second sleep with:

```bash
wait_for_ready() {
	local attempt
	for attempt in $(seq 1 30); do
		if curl -skf https://localhost:8081/ready >/dev/null 2>&1; then
			echo "Go API ready at https://localhost:8081/ready"
			return 0
		fi
		if curl -sf http://localhost:8081/ready >/dev/null 2>&1; then
			echo "Go API ready at http://localhost:8081/ready"
			return 0
		fi
		sleep 1
	done
	echo "Go API did not become ready within 30 seconds." >&2
	tail -n 80 "$LOG_DIR/go.log" >&2
	return 1
}
```

Call `wait_for_ready` before `cmd_status`. Replace the current health block in `cmd_status` with:

```bash
	if curl -skf https://localhost:8081/health >/dev/null 2>&1; then
		echo "  Liveness   https://localhost:8081/health OK"
	elif curl -sf http://localhost:8081/health >/dev/null 2>&1; then
		echo "  Liveness   http://localhost:8081/health OK"
	else
		echo "  Liveness   DOWN"
	fi
	if curl -skf https://localhost:8081/ready >/dev/null 2>&1; then
		echo "  Readiness  https://localhost:8081/ready OK"
	elif curl -sf http://localhost:8081/ready >/dev/null 2>&1; then
		echo "  Readiness  http://localhost:8081/ready OK"
	else
		echo "  Readiness  DOWN"
	fi
```

Add this exact paragraph to `dev/README.md` under **Full stack**:

```markdown
`make dev-up` returns success only after `GET /ready` confirms both MongoDB and
the configured catalog store. `GET /health` only proves that the Go process can
serve HTTP; use `make dev-status` to see both signals separately.
```

- [ ] **Step 6: Update Docker Make targets and operational tests**

Set local root-Compose targets explicitly:

```make
docker-run:
	COMICS_ENV_FILE=go_server/.env COMPOSE_PROFILES=local-mongo docker compose up --build comics_server

docker-dev:
	COMICS_ENV_FILE=go_server/.env COMPOSE_PROFILES=local-mongo docker compose up --build
```

Extend `ops-test`:

```make
ops-test:
	@bash ops/tests/operations_test.sh
	@bash ops/tests/runtime_config_test.sh
```

- [ ] **Step 7: Validate configurations and image**

```bash
chmod +x ops/tests/runtime_config_test.sh
bash ops/tests/runtime_config_test.sh
docker build -t comics-go-api:lifecycle-plan -f Dockerfile .
docker inspect comics-go-api:lifecycle-plan \
  --format '{{json .Config.Healthcheck.Test}} {{json .Config.Cmd}}'
```

Expected:

```text
runtime config tests passed
["CMD-SHELL","sh -c 'curl --fail --silent --show-error \"http://127.0.0.1:${HTTP_PORT:-${PORT:-8081}}${HEALTHCHECK_PATH}\"' || exit 1"] ["./ops/start-server.sh"]
```

- [ ] **Step 8: Commit**

```bash
git add Dockerfile go_server/Dockerfile docker-compose.yaml \
  go_server/docker-compose.yaml go_server/.air.toml go_server/.air.plain.toml \
  render.yaml dev/stack.sh dev/README.md \
  Makefile ops/tests/runtime_config_test.sh
git commit -m "ops: gate runtimes on datastore readiness"
```

---

### Task 7: Gate Production Deploys and Roll Back Failed Releases

**Files:**

- Create: `ops/postdeploy-smoke.sh`
- Create: `ops/deploy-production.sh`
- Create: `ops/tests/deploy_config_test.sh`
- Modify: `ops/tests/operations_test.sh`
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/ci-cd.yml`
- Modify: `Makefile`

**Interfaces:**

- Produces: `SERVICE_URL=http://127.0.0.1:8081 ./ops/postdeploy-smoke.sh`
- Consumes: `DEPLOY_SHA`, `PREVIOUS_SHA`, `COMICS_ENV_FILE`, `SERVICE_URL`
- Produces: deployment only from a successful `CI` `push` run on `main`

- [ ] **Step 1: Write failing deploy policy tests**

Create `ops/tests/deploy_config_test.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

rg -q '^  workflow_run:' .github/workflows/ci-cd.yml
rg -q 'workflows: \["CI"\]' .github/workflows/ci-cd.yml
rg -q "github\.event\.workflow_run\.conclusion == 'success'" .github/workflows/ci-cd.yml
rg -q "github\.event\.workflow_run\.event == 'push'" .github/workflows/ci-cd.yml
rg -q 'github\.event\.workflow_run\.head_sha' .github/workflows/ci-cd.yml
rg -q 'make ops-test' .github/workflows/ci.yml
rg -q 'contract-test' .github/workflows/ci.yml
rg -q 'sqlite-backup\.sh' ops/deploy-production.sh
rg -q 'sqlite-restore\.sh' ops/deploy-production.sh
rg -q 'postdeploy-smoke\.sh' ops/deploy-production.sh

bash -n ops/deploy-production.sh ops/postdeploy-smoke.sh
echo "deploy config tests passed"
```

Run:

```bash
bash ops/tests/deploy_config_test.sh
```

Expected: FAIL because deployment still triggers directly from a merged PR and has no backup, readiness, smoke, or rollback stages.

- [ ] **Step 2: Implement retrying post-deploy smoke**

Create `ops/postdeploy-smoke.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

service_url="${SERVICE_URL:-http://127.0.0.1:8081}"
attempts="${SMOKE_ATTEMPTS:-30}"
sleep_seconds="${SMOKE_SLEEP_SECONDS:-2}"
curl_bin="${CURL_BIN:-curl}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

request() {
	local path="$1"
	local body="$2"
	"$curl_bin" --silent --show-error --max-time 5 \
		--output "$body" --write-out '%{http_code}' \
		"${service_url}${path}"
}

ready=0
for ((attempt = 1; attempt <= attempts; attempt++)); do
	code="$(request /ready "$tmp/ready.json" || true)"
	if [[ "$code" == "200" ]] && [[ "$(cat "$tmp/ready.json")" == '{"status":"UP"}' ]]; then
		ready=1
		break
	fi
	sleep "$sleep_seconds"
done
[[ "$ready" == "1" ]] || {
	echo "postdeploy-smoke: /ready did not pass after $attempts attempts" >&2
	exit 1
}

code="$(request /health "$tmp/health.json")"
[[ "$code" == "200" && "$(cat "$tmp/health.json")" == '{"status":"UP"}' ]] || {
	echo "postdeploy-smoke: /health failed" >&2
	exit 1
}

code="$(request '/comics?from=0&limit=1' "$tmp/comics.json")"
[[ "$code" == "200" ]] || {
	echo "postdeploy-smoke: public catalog read returned HTTP $code" >&2
	exit 1
}

echo "postdeploy smoke passed: health, readiness, catalog"
```

Extend `operations_test.sh` with:

```bash
cat > "$fake_bin/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
output=""
url=""
while [[ $# -gt 0 ]]; do
	case "$1" in
		--output)
			output="$2"
			shift 2
			;;
		http://*|https://*)
			url="$1"
			shift
			;;
		*)
			shift
			;;
	esac
done
case "$url" in
	*/health|*/ready)
		printf '%s' '{"status":"UP"}' > "$output"
		;;
	*/comics\?*)
		printf '%s' '[]' > "$output"
		;;
	*)
		exit 1
		;;
esac
printf '200'
EOF
chmod +x "$fake_bin/curl"

CURL_BIN="$fake_bin/curl" \
SERVICE_URL=http://127.0.0.1:8081 \
SMOKE_ATTEMPTS=1 \
SMOKE_SLEEP_SECONDS=0 \
"$ROOT/ops/postdeploy-smoke.sh"
```

- [ ] **Step 3: Implement stopped backup, candidate rollout, and automatic rollback**

Create `ops/deploy-production.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

die() {
	echo "deploy-production: $*" >&2
	exit 1
}

env_value() {
	local key="$1"
	awk -F= -v key="$key" '
		$1 == key {
			sub(/^[^=]*=/, "")
			gsub(/^"|"$/, "")
			print
			exit
		}
	' "$COMICS_ENV_FILE"
}

deploy_sha="${DEPLOY_SHA:-}"
previous_sha="${PREVIOUS_SHA:-}"
service_url="${SERVICE_URL:-http://127.0.0.1:8081}"
COMICS_ENV_FILE="${COMICS_ENV_FILE:-$ROOT/go_server/.env.production}"
image_repo="${COMICS_IMAGE_REPOSITORY:-comics-go-api}"
data_dir="${COMICS_DATA_DIR:-$ROOT/db}"
export COMICS_ENV_FILE COMICS_DATA_DIR

[[ "$deploy_sha" =~ ^[0-9a-f]{40}$ ]] || die "DEPLOY_SHA must be a full commit SHA"
[[ "$previous_sha" =~ ^[0-9a-f]{40}$ ]] || die "PREVIOUS_SHA must be a full commit SHA"
[[ -f "$COMICS_ENV_FILE" ]] || die "production env file not found: $COMICS_ENV_FILE"
mkdir -p "$data_dir/backups"
chmod 0700 "$data_dir" "$data_dir/backups"
mode="$(stat -c '%a' "$COMICS_ENV_FILE")"
[[ "$mode" == "600" || "$mode" == "400" ]] ||
	die "production env file must have mode 0600 or 0400, got $mode"

access_secret="$(env_value ACCESS_TOKEN_SECRET)"
refresh_secret="$(env_value REFRESH_TOKEN_SECRET)"
db_addr="$(env_value DB_ADDR)"
[[ ${#access_secret} -ge 32 ]] || die "ACCESS_TOKEN_SECRET must contain at least 32 characters"
[[ ${#refresh_secret} -ge 32 ]] || die "REFRESH_TOKEN_SECRET must contain at least 32 characters"
[[ "$access_secret" != "$refresh_secret" ]] || die "JWT secrets must be different"
[[ -n "$db_addr" ]] || die "DB_ADDR is required"

case "$db_addr" in
	mongodb://mongodb:*)
		export COMPOSE_PROFILES=local-mongo
		;;
	mongodb+srv://*)
		[[ -n "$(env_value DB_USER)" ]] || die "DB_USER is required for Atlas"
		[[ -n "$(env_value DB_PASS)" ]] || die "DB_PASS is required for Atlas"
		;;
esac

docker compose config --quiet

old_container="$(docker compose ps -q comics_server 2>/dev/null || true)"
old_image_available=0
if [[ -n "$old_container" ]]; then
	old_image_id="$(docker inspect --format '{{.Image}}' "$old_container")"
	docker image tag "$old_image_id" "$image_repo:$previous_sha"
	old_image_available=1
fi

docker build --pull --tag "$image_repo:$deploy_sha" -f Dockerfile .

restart_previous() {
	if [[ "$old_image_available" == "1" ]]; then
		recovery_data="$data_dir"
		if [[ ! -f "$recovery_data/comics.db" && -f "$ROOT/db/comics.db" ]]; then
			recovery_data="$ROOT/db"
		fi
		COMICS_IMAGE="$image_repo:$previous_sha" \
		COMICS_DATA_DIR="$recovery_data" \
		docker compose up -d --no-build --wait --wait-timeout 90 comics_server
	fi
}

docker compose stop -t 30 comics_server
if [[ ! -f "$data_dir/comics.db" ]]; then
	legacy_db="$ROOT/db/comics.db"
	[[ -f "$legacy_db" ]] || {
		restart_previous
		die "neither $data_dir/comics.db nor seed $legacy_db exists"
	}
	initial_backup="$(
		DB_FILE="$legacy_db" \
		BACKUP_DIR="$data_dir/backups" \
		BACKUP_LABEL="initial-${deploy_sha:0:12}" \
		"$ROOT/ops/sqlite-backup.sh"
	)" || {
		restart_previous
		die "initial catalog seed failed; previous image restarted"
	}
	install -m 0600 "$initial_backup" "$data_dir/comics.db"
fi
backup="$(
	DB_FILE="$data_dir/comics.db" \
	BACKUP_DIR="$data_dir/backups" \
	BACKUP_LABEL="predeploy-${deploy_sha:0:12}" \
	"$ROOT/ops/sqlite-backup.sh"
)" || {
	restart_previous
	die "predeploy backup failed; previous image restarted"
}

rollback() {
	echo "candidate failed; restoring $backup and $previous_sha" >&2
	set +e
	docker compose stop -t 30 comics_server
	SERVICE_STOPPED=1 \
	DB_FILE="$data_dir/comics.db" \
	BACKUP_DIR="$data_dir/backups" \
	BACKUP="$backup" \
	COMPOSE_FILE="$ROOT/docker-compose.yaml" \
	"$ROOT/ops/sqlite-restore.sh"
	restore_status=$?
	if [[ "$old_image_available" == "1" ]]; then
		export COMICS_IMAGE="$image_repo:$previous_sha"
		docker compose up -d --no-build --wait --wait-timeout 90 comics_server
		SERVICE_URL="$service_url" "$ROOT/ops/postdeploy-smoke.sh"
		smoke_status=$?
	else
		smoke_status=1
	fi
	git switch --detach "$previous_sha"
	set -e
	if [[ "$restore_status" != "0" || "$smoke_status" != "0" ]]; then
		echo "automatic rollback requires operator intervention" >&2
	fi
	return 1
}

export COMICS_IMAGE="$image_repo:$deploy_sha"
if ! docker compose up -d --no-build --wait --wait-timeout 90 comics_server; then
	rollback || true
	exit 1
fi
if ! SERVICE_URL="$service_url" "$ROOT/ops/postdeploy-smoke.sh"; then
	rollback || true
	exit 1
fi

mkdir -p "$ROOT/.deploy"
cat > "$ROOT/.deploy/current.env" <<EOF
DEPLOY_SHA=$deploy_sha
PREVIOUS_SHA=$previous_sha
PREDEPLOY_BACKUP=$backup
EOF
chmod 0600 "$ROOT/.deploy/current.env"
echo "deployment passed: $deploy_sha"
```

Also ignore `.deploy/` in `.gitignore`.

- [ ] **Step 4: Add operational and contract checks to CI**

In `.github/workflows/ci.yml`:

- Add `make contract-test` to the Go job after `go test ./...`.
- Add an `operations` job that checks out the repository, installs `sqlite3`, and runs `make ops-test`.
- Keep the existing Go, frontend, vet, build, and lint steps.

The operations job command must be:

```yaml
- name: Test operational contracts
  run: |
    sudo apt-get update
    sudo apt-get install -y ripgrep sqlite3
    make ops-test
```

- [ ] **Step 5: Gate SSH deployment on successful `main` CI**

Replace `.github/workflows/ci-cd.yml` with:

```yaml
name: Deploy after CI

on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches: [main]

concurrency:
  group: production-deploy
  cancel-in-progress: false

permissions:
  contents: read

jobs:
  deploy_production:
    if: >-
      github.event.workflow_run.conclusion == 'success' &&
      github.event.workflow_run.event == 'push'
    runs-on: ubuntu-latest
    environment: production
    steps:
      - name: Deploy verified commit
        uses: appleboy/ssh-action@v1.2.0
        env:
          DEPLOY_SHA: ${{ github.event.workflow_run.head_sha }}
        with:
          host: ${{ secrets.HOST }}
          username: ${{ secrets.USERNAME }}
          key: ${{ secrets.PRIVATE_KEY }}
          port: ${{ secrets.PORT }}
          envs: DEPLOY_SHA
          script_stop: true
          script: |
            set -euo pipefail
            if [ ! -d /home/dev/comics/.git ]; then
              git clone git@github.com:estebmaister/comics.git /home/dev/comics
            fi
            cd /home/dev/comics
            previous_sha="$(git rev-parse HEAD)"
            git fetch --prune origin main
            test "$(git rev-parse "${DEPLOY_SHA}^{commit}")" = "$DEPLOY_SHA"
            git switch main
            git merge --ff-only "$DEPLOY_SHA"
            PREVIOUS_SHA="$previous_sha" \
            DEPLOY_SHA="$DEPLOY_SHA" \
            COMICS_ENV_FILE=/home/dev/comics/go_server/.env.production \
            COMICS_DATA_DIR=/home/dev/comics-data \
            SERVICE_URL=http://127.0.0.1:8081 \
            ./ops/deploy-production.sh
```

The workflow does not carry application secrets; only SSH connection secrets are stored in GitHub. Runtime secrets remain on the host.

- [ ] **Step 6: Add Make smoke and complete ops-test**

Add:

```make
## @deploy:postdeploy-smoke     Check liveness, readiness, and one public catalog read
postdeploy-smoke:
	@SERVICE_URL="$(or $(SERVICE_URL),http://127.0.0.1:8081)" ./ops/postdeploy-smoke.sh

ops-test:
	@bash ops/tests/operations_test.sh
	@bash ops/tests/runtime_config_test.sh
	@bash ops/tests/deploy_config_test.sh
```

Add `postdeploy-smoke` to `.PHONY`.

- [ ] **Step 7: Run policy, script, and workflow checks**

```bash
chmod +x ops/postdeploy-smoke.sh ops/deploy-production.sh \
  ops/tests/deploy_config_test.sh
bash ops/tests/operations_test.sh
bash ops/tests/runtime_config_test.sh
bash ops/tests/deploy_config_test.sh
make ops-test
```

Expected:

```text
operations tests passed
runtime config tests passed
deploy config tests passed
```

- [ ] **Step 8: Commit**

```bash
git add ops .github/workflows/ci.yml .github/workflows/ci-cd.yml \
  Makefile .gitignore
git commit -m "ops: gate deploys with backup and rollback"
```

---

### Task 8: Publish the Operations, Secrets, and Recovery Runbook

**Files:**

- Create: `docs/operations.md`
- Modify: `docs/deploy-render.md`
- Modify: `README.md`
- Modify: `go_server/README.md`
- Modify: `dev/README.md`

**Interfaces:**

- Documents: liveness/readiness, shutdown bounds, SQLite one-replica rule, Mongo choices, secret ownership, backup/restore, SSH rollback, Render recovery, and smoke.

- [ ] **Step 1: Add documentation assertions before writing prose**

Extend `ops/tests/runtime_config_test.sh`:

```bash
rg -q 'SQLite mode runs exactly one API replica' docs/operations.md
rg -q 'SERVICE_STOPPED=1' docs/operations.md
rg -q 'MongoDB Atlas' docs/operations.md
rg -q 'SQLITE_RESTORE_TOKEN' docs/operations.md
rg -q '/ready' docs/deploy-render.md
rg -q 'autoDeployTrigger: checksPass' docs/deploy-render.md
rg -q 'docs/operations.md' README.md
```

Run `make ops-test`.

Expected: FAIL because `docs/operations.md` does not exist and Render still tells operators to probe `/health`.

- [ ] **Step 2: Create the canonical operations runbook**

Create `docs/operations.md` with these exact operational contracts and commands:

````markdown
# Production Operations

## Runtime contract

SQLite mode runs exactly one API replica and one periodic scheduler. The process
holds `comics.db.instance.lock`; a second API or `make go-scrape` process exits
before opening the catalog. PostgreSQL does not use the file lock, but the API
must still remain at one replica until scrape coordination has a database-backed
lease.

The SSH deployment stores mutable catalog data in `/home/dev/comics-data`, not
in the Git checkout. The tracked `db/comics.db` is only the first-deploy seed and
local development database.

`GET /health` is process liveness and returns `200 {"status":"UP"}` without
touching a datastore. `GET /ready` synchronously pings MongoDB and the catalog,
returns 503 while either is unavailable, and returns 503 as soon as shutdown
starts.

SIGTERM starts a 15-second concurrent drain of HTTP, the periodic scheduler, and
active scrape work. Repository closure then has 10 seconds. Docker and Render
allow 30 seconds before forced termination.

## Secrets

Render secrets are `sync: false` values in `render.yaml`. The SSH host stores
runtime values only in `/home/dev/comics/go_server/.env.production`:

```bash
cd /home/dev/comics
umask 077
install -m 0600 /dev/null go_server/.env.production
access_secret="$(openssl rand -hex 32)"
refresh_secret="$(openssl rand -hex 32)"
cat > go_server/.env.production <<EOF
ENVIRONMENT=production
HOST_URL=comic-tracker-server.onrender.com
DB_ADDR=mongodb+srv://sandbox.ux3yw.mongodb.net/
DB_USER=esteb
DB_PASS=$DB_PASS
DB_NAME=comics
ACCESS_TOKEN_SECRET=$access_secret
REFRESH_TOKEN_SECRET=$refresh_secret
CORS_ALLOWED_ORIGINS=https://estebmaister.github.io
SCRAPE_INTERVAL=10m
READINESS_TIMEOUT=2s
SHUTDOWN_TIMEOUT=15s
INIT_TIMEOUT=10s
EOF
chmod 0600 go_server/.env.production
```

Set `DB_PASS` in the interactive shell before running that block. Never print,
commit, or place the production env file in a GitHub Actions input. Rotate access
and refresh secrets independently; changing either invalidates its existing
tokens.

## MongoDB

Render uses MongoDB Atlas through `DB_ADDR`, `DB_USER`, and `DB_PASS`. The SSH
stack may also use Atlas. A self-hosted Compose MongoDB is enabled only with the
`local-mongo` profile, binds its host port to `127.0.0.1`, and persists
`/data/db` in the `mongo_data` volume:

```bash
COMICS_ENV_FILE=go_server/.env.production \
COMPOSE_PROFILES=local-mongo \
docker compose up -d mongodb
docker volume inspect comics_mongo_data
```

Do not run an unpersisted Mongo container in production.

## Verified backup

The deployment script stops `comics_server`, creates a transactionally
consistent SQLite backup, runs `PRAGMA integrity_check`, verifies the `comics`
table, and only then starts the candidate image.

Manual backup can run online:

```bash
cd /home/dev/comics
backup="$(DB_FILE=/home/dev/comics-data/comics.db \
  BACKUP_DIR=/home/dev/comics-data/backups \
  BACKUP_LABEL=manual ./ops/sqlite-backup.sh)"
test "$(sqlite3 "$backup" 'PRAGMA integrity_check;')" = "ok"
printf 'verified backup: %s\n' "$backup"
```

## Safe restore on the SSH host

Restore never runs against a live API:

```bash
cd /home/dev/comics
backup="$(ls -1t /home/dev/comics-data/backups/comics_predeploy_*.db | awk 'NR == 1 { print; exit }')"
docker compose stop -t 30 comics_server
SERVICE_STOPPED=1 \
DB_FILE=/home/dev/comics-data/comics.db \
BACKUP_DIR=/home/dev/comics-data/backups \
BACKUP="$backup" \
make db-restore
COMICS_ENV_FILE=go_server/.env.production \
COMICS_DATA_DIR=/home/dev/comics-data \
docker compose up -d --wait comics_server
SERVICE_URL=http://127.0.0.1:8081 make postdeploy-smoke
```

The restore script first creates a verified `pre-restore` backup, copies the
selected database to a temporary file on the same filesystem, verifies it, and
removes stopped-process `-wal`/`-shm` sidecars before atomically renaming it over
`comics.db`.

## SSH deployment rollback

`.github/workflows/ci-cd.yml` runs only after the `CI` workflow succeeds for the
exact `main` push SHA. The host builds that SHA, stops the old API, backs up
SQLite, starts the candidate, and checks `/ready`, `/health`, and a public
catalog read. A failure stops the candidate, restores the predeploy backup,
switches to the prior commit, starts the prior image, and repeats smoke.

Inspect the last successful release:

```bash
cd /home/dev/comics
cat .deploy/current.env
docker compose ps
SERVICE_URL=http://127.0.0.1:8081 ./ops/postdeploy-smoke.sh
```

## Render deployment and recovery

Render uses `autoDeployTrigger: checksPass`, `/ready`, one instance, and the
`/data` disk. The disk prevents zero-downtime deploys and is inaccessible from
build, pre-deploy, and one-off job instances.

Before merging a destructive data/schema release, open the running Render Shell
and create a backup on the disk:

```bash
backup="$(DB_FILE=/data/comics.db BACKUP_DIR=/data/backups \
  BACKUP_LABEL=predeploy ./ops/sqlite-backup.sh)"
test "$(sqlite3 "$backup" 'PRAGMA integrity_check;')" = "ok"
printf 'verified backup: %s\n' "$backup"
```

Code rollback from Render's Deploys page does not roll back `/data`. For a data
restore, set `SQLITE_RESTORE_FROM` to the printed `/data/backups` path and set
`SQLITE_RESTORE_TOKEN` to a new incident identifier such as
`incident-20261026-01`, then deploy the last known-good image. Render stops the
old instance first; `ops/start-server.sh` verifies and atomically restores the
database before the Go process opens it. After `/ready` passes, remove both
restore variables and redeploy. The token marker prevents a restart from
reapplying the same restore.

Do not use Render's full-disk snapshot restore for SQLite recovery; Render
documents that custom databases should use database-native backup and restore.

## Postdeploy smoke

```bash
SERVICE_URL=https://comic-tracker-server.onrender.com \
  ./ops/postdeploy-smoke.sh
```

Success prints:

```text
postdeploy smoke passed: health, readiness, catalog
```
````

- [ ] **Step 3: Correct Render and project documentation**

Replace the health/deploy verification guidance in `docs/deploy-render.md` with:

````markdown
### Readiness and deploy gate

The repository Blueprint sets `autoDeployTrigger: checksPass`,
`healthCheckPath: /ready`, and `numInstances: 1`. Do not enable autoscaling:
SQLite and the process-local scrape coordinator support one API instance.
`/health` is liveness only and must not gate a deployment.

Set these lifecycle values:

```env
READINESS_TIMEOUT=2s
SHUTDOWN_TIMEOUT=15s
INIT_TIMEOUT=10s
```

Render cannot access `/data` from a build, pre-deploy command, or one-off job.
Follow [Production Operations](operations.md) to create a database-native backup
from the running instance and to perform a one-shot restore before the server
opens SQLite.

Verify the live service:

```bash
SERVICE_URL=https://comic-tracker-server.onrender.com \
  ./ops/postdeploy-smoke.sh
```
````

Add the following section to root `README.md`:

```markdown
### Production lifecycle and recovery

SQLite mode supports exactly one API process and one scheduler; a process-held
lock rejects a second API or CLI writer. `/health` reports process liveness,
while `/ready` checks MongoDB plus the catalog and is the Docker/Render deploy
gate. The SSH deployment mounts `/home/dev/comics-data`; the tracked
`db/comics.db` is not the mutable production file. Production backup, stopped
restore, secrets, MongoDB persistence, smoke, and rollback procedures are documented in
[Production Operations](docs/operations.md).
```

Replace the production Docker setup paragraph in `go_server/README.md` with:

```markdown
**Production:** create the ignored `go_server/.env.production` on the host with
mode `0600`; never copy the tracked local template into production. MongoDB
Atlas is the recommended auth store. Self-hosted Compose MongoDB is enabled with
the `local-mongo` profile and persists in `mongo_data`.

SQLite mode holds `${COMICS_SQLITE_PATH}.instance.lock` and permits exactly one
API or CLI process. `/health` is liveness; `/ready` checks MongoDB and the
catalog. See [`docs/operations.md`](../docs/operations.md) for verified `.db`
backup, stopped restore, deploy smoke, and rollback. `db/comics.json` is not
part of the Go runtime backup or restore path.
```

- [ ] **Step 4: Run the complete release-path verification**

```bash
make ops-test
make check-go
make check-front
make check
docker build -t comics-go-api:lifecycle-final -f Dockerfile .
```

Expected: all commands exit `0`; CI contract tests are part of `check-go`; runtime and deploy configuration tests print their three success lines.

- [ ] **Step 5: Run an isolated Compose smoke and graceful-stop rehearsal**

```bash
test_root="$(mktemp -d)"
test_env="$(mktemp)"
cp db/comics.db "$test_root/comics.db"
cat > "$test_env" <<'EOF'
ENVIRONMENT=production
HOST_URL=localhost:8081
DB_ADDR=mongodb://mongodb:27017
DB_NAME=comics
DB_USER=
DB_PASS=
ACCESS_TOKEN_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
REFRESH_TOKEN_SECRET=fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210
CORS_ALLOWED_ORIGINS=http://localhost:3000
SCRAPE_INTERVAL=0
READINESS_TIMEOUT=2s
SHUTDOWN_TIMEOUT=15s
INIT_TIMEOUT=10s
EOF
chmod 0600 "$test_env"
COMPOSE_PROJECT_NAME=comics-lifecycle-test \
COMICS_ENV_FILE="$test_env" \
COMICS_DATA_DIR="$test_root" \
COMPOSE_PROFILES=local-mongo \
docker compose up -d --build --wait --wait-timeout 90
SERVICE_URL=http://127.0.0.1:8081 ./ops/postdeploy-smoke.sh
COMPOSE_PROJECT_NAME=comics-lifecycle-test \
COMICS_ENV_FILE="$test_env" \
COMICS_DATA_DIR="$test_root" \
COMPOSE_PROFILES=local-mongo \
docker compose stop -t 30 comics_server
rollback_backup="$(
  DB_FILE="$test_root/comics.db" \
  BACKUP_DIR="$test_root/backups" \
  BACKUP_LABEL=rollback-rehearsal \
  ./ops/sqlite-backup.sh
)"
sqlite3 "$test_root/comics.db" \
  "CREATE TABLE lifecycle_rollback_probe(value TEXT); INSERT INTO lifecycle_rollback_probe VALUES('candidate');"
COMPOSE_PROJECT_NAME=comics-lifecycle-test \
COMICS_ENV_FILE="$test_env" \
COMICS_DATA_DIR="$test_root" \
COMPOSE_PROFILES=local-mongo \
SERVICE_STOPPED=1 \
DB_FILE="$test_root/comics.db" \
BACKUP_DIR="$test_root/backups" \
BACKUP="$rollback_backup" \
COMPOSE_FILE="$PWD/docker-compose.yaml" \
./ops/sqlite-restore.sh
test "$(sqlite3 "$test_root/comics.db" \
  "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='lifecycle_rollback_probe';")" = "0"
COMPOSE_PROJECT_NAME=comics-lifecycle-test \
COMICS_ENV_FILE="$test_env" \
COMICS_DATA_DIR="$test_root" \
COMPOSE_PROFILES=local-mongo \
docker compose up -d --no-build --wait --wait-timeout 90
SERVICE_URL=http://127.0.0.1:8081 ./ops/postdeploy-smoke.sh
COMPOSE_PROJECT_NAME=comics-lifecycle-test \
COMICS_ENV_FILE="$test_env" \
COMICS_DATA_DIR="$test_root" \
COMPOSE_PROFILES=local-mongo \
docker compose stop -t 30 comics_server
COMPOSE_PROJECT_NAME=comics-lifecycle-test \
COMICS_ENV_FILE="$test_env" \
COMICS_DATA_DIR="$test_root" \
COMPOSE_PROFILES=local-mongo \
docker compose down -v
rm -rf "$test_root" "$test_env"
```

Expected: Compose reports `comics_server` healthy before both smoke runs; stop exits within 30 seconds without `SIGKILL`; restore removes `lifecycle_rollback_probe`; only the test project's Mongo volume is removed.

- [ ] **Step 6: Commit**

```bash
git add docs/operations.md docs/deploy-render.md README.md \
  go_server/README.md dev/README.md ops/tests/runtime_config_test.sh
git commit -m "docs: publish production recovery runbook"
```

---

## Rollback Procedure for This Workstream

If a release containing this workstream fails before the candidate passes smoke:

```bash
cd /home/dev/comics
cat .deploy/current.env
docker compose stop -t 30 comics_server
set -a
. .deploy/current.env
set +a
SERVICE_STOPPED=1 \
DB_FILE=/home/dev/comics-data/comics.db \
BACKUP_DIR=/home/dev/comics-data/backups \
BACKUP="$PREDEPLOY_BACKUP" \
make db-restore
COMICS_IMAGE="comics-go-api:$PREVIOUS_SHA" \
COMICS_ENV_FILE=/home/dev/comics/go_server/.env.production \
COMICS_DATA_DIR=/home/dev/comics-data \
docker compose up -d --no-build --wait --wait-timeout 90 comics_server
SERVICE_URL=http://127.0.0.1:8081 ./ops/postdeploy-smoke.sh
git switch --detach "$PREVIOUS_SHA"
```

Expected: restore reports `PRAGMA integrity_check: ok` through the script's success path, the prior image becomes healthy, and smoke prints `postdeploy smoke passed: health, readiness, catalog`.

If the new SQLite process lock itself prevents startup, verify no old API or CLI process still owns the database before rollback:

```bash
docker compose ps
lsof db/comics.db.instance.lock
```

Do not delete the lock file to bypass an active owner. The file may remain after a crash; the kernel lock, not file existence, determines ownership.

For Render, use a prior successful deploy for code and the `SQLITE_RESTORE_FROM` plus a new `SQLITE_RESTORE_TOKEN` flow for data. A Render code rollback alone deliberately leaves the disk unchanged.

## Acceptance Evidence

- Go tests prove liveness never pings stores and readiness checks both stores.
- Go race tests prove coordinator stop cancels active work, rejects new work, and scheduler stop is waitable.
- A real `http.Server` test proves in-flight HTTP completes before repository close.
- A timeout test proves forced drain remains bounded and repository cleanup still runs.
- The SQLite lock test proves a second process fails and a later process succeeds after release.
- Shell tests prove backup integrity, stopped-service restore enforcement, running-container refusal, atomic restore, and one-shot Render recovery.
- Runtime config tests prove Docker and Render use `/ready`, Render and SQLite use one instance, Mongo has durable storage, and the dev stack waits for readiness.
- Deploy config tests prove deployment is downstream of successful `main` CI and includes backup, restore, and smoke.
- The isolated Compose rehearsal proves Mongo/catalog readiness, public catalog smoke, and termination within the platform window.
- No horizontal-scaling claim is made for SQLite or the process-local scrape coordinator.
