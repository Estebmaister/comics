# Scrape Reliability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every scrape run observable and truthful: report structured publisher outcomes, retry only transient fetch failures, expose bounded run history, cancel and drain safely, and prevent overlap locally for SQLite or across replicas with PostgreSQL advisory locks.

**Architecture:** The coordinator owns run IDs, state transitions, single-flight admission, history, cancellation, and shutdown draining. The runner owns publisher execution and returns a structured report; a retrying `PageFetcher` decorates only network fetches, so parser, configuration, and registration failures are never retried. SQLite uses one in-process locker plus an in-memory bounded store under the existing one-replica deployment invariant; PostgreSQL uses one dedicated advisory-lock connection and a `scrape_runs` table so all replicas see the same active run and history without Redis.

**Tech Stack:** Go 1.27.3, Gin, `database/sql`, pgx, modernc SQLite, Prometheus client, React 19, TypeScript 6, Vitest, Testing Library.

## Global Constraints

- Preserve the Go modular monolith, React REST client, SQLite default catalog, MongoDB user store, and optional PostgreSQL catalog.
- Do not introduce Redis, a queue, a worker service, or another runtime dependency.
- SQLite coordination is process-local only. SQLite deployments must run exactly one API replica and one periodic scheduler; do not claim that `cmd/scrape` is coordinated with a separately running API process.
- SQLite run status/history is bounded process memory and resets on restart; document that behavior rather than implying durable history.
- PostgreSQL horizontal scaling is allowed only when the PostgreSQL advisory-lock/run-store implementation in Task 5 is active.
- Keep publisher fetches concurrent, keep `Manganato` URLs sequential for the shared cookie jar, and keep catalog registration serialized within one run.
- Retry only typed transient fetch failures. Never retry parser, publisher configuration, normalization, or registration/database failures.
- Use four total fetch attempts with exponential delays of `500ms`, `1s`, and `2s`, capped at `2s`, with symmetric `±20%` jitter. Cancellation must interrupt backoff immediately.
- Treat HTTP `408`, `425`, `429`, `500`, `502`, `503`, and `504`, transport timeouts/resets, truncated response bodies, and the Cloudflare challenge page as transient. Treat malformed URLs, TLS certificate validation failures, and all other non-`200` responses as permanent.
- Use a `30m` default run timeout from `SCRAPE_TIMEOUT`; preserve the existing `10m` default `SCRAPE_INTERVAL`.
- Retain at most 50 terminal runs. `GET /scrape/history` defaults to 10 and clamps `limit` to `1..50`.
- Public responses contain stable failure codes and safe messages. Log wrapped transport/database errors with `run_id`, `publisher`, and `url`; never serialize raw internal errors.
- Initialize run publisher/failure slices and publisher failure slices as empty non-nil slices so API JSON uses `[]`, never `null`.
- A canceled run remains `canceled` even when some publisher writes committed before cancellation. Totals and publisher outcomes must still show that partial work.
- A run is `complete` only when every configured publisher completed without a failed URL or registration. It is `partial_failure` when any publisher work succeeded and any publisher/configuration work failed. It is `total_failure` when no publisher work succeeded. Cancellation takes precedence over both failure states.
- Manual scrape trigger, status, and history routes remain admin-only after workstream 1. Public catalog reads remain public.
- Preserve Workstream 4's
  `(*PeriodicScheduler).Stop(context.Context) error` and its waitable shutdown
  tests when rewriting periodic scheduling.
- Preserve the dark-first frontend, existing floating action rail, and toast provider. Do not add a modal for this workstream.
- Follow TDD: add the named failing test, observe the expected failure, implement the minimum contract, rerun the focused test, then commit.

## Dependencies and Sequencing

1. Apply workstream 1 first so scrape routes use JWT plus admin-role middleware and frontend requests can send the stored access token.
2. Apply workstream 3 first so registration writes use the final transactional repository behavior.
3. Apply workstream 4 first so the lifecycle manager owns the one idempotent `Coordinator.Shutdown(context.Context)` call before catalog/user repositories close. Do not also register coordinator shutdown as an application shutter.
4. Workstream 7 may later move the frontend helpers into the shared API client; this plan keeps scrape-specific request and response types in `ScrapeButton.tsx` so it does not depend on that later refactor.
5. Workstream 9 may aggregate dashboards and alerts, but the low-cardinality scrape metrics are created here because the coordinator and runner are their source of truth.

## Final HTTP and State Contract

- `POST /scrape`
  - `202 Accepted`: an object with one `run` field containing the accepted `domain.ScrapeRun`.
  - `409 Conflict`: `{"code":"scrape_in_progress","message":"scrape already in progress","run":null}` when the active row is not yet visible; otherwise `run` contains the active `domain.ScrapeRun`.
  - `503 Service Unavailable`: `{"code":"scrape_coordination_unavailable","message":"scrape coordination is unavailable"}`.
- `GET /scrape`
  - Keep as an authenticated synchronous compatibility endpoint during this workstream.
  - Start through the same coordinator, wait without making request cancellation cancel the accepted run, and return `200`; its body has `message: "success"` and a `run` field containing the terminal `domain.ScrapeRun` for `complete` or `partial_failure`.
  - Return `502` with code `scrape_total_failure` for `total_failure`, `503` with code `scrape_canceled` for `canceled`, and the same `409` busy response as `POST`.
  - Document it as deprecated; do not remove it in this workstream because the frontend and backend deploy independently.
- `GET /scrape/status`
  - `200 OK`: `{"running":false,"active":null,"latest":null}` before the first run; populated fields use `domain.ScrapeRun`.
- `GET /scrape/history?limit=10`
  - `200 OK`: `{"runs":[]}` before the first run; populated arrays are newest-terminal-run first.
- Run states are exactly `accepted`, `running`, `complete`, `partial_failure`, `total_failure`, and `canceled`.
- Publisher states are exactly `complete`, `partial_failure`, `failed`, and `canceled`.
- The frontend refreshes the catalog after `complete` and `partial_failure`, but not after `total_failure` or `canceled`.

## File Structure

### Create

- `go_server/domain/scrape.go` — API-visible run, publisher outcome, totals, failure, source, and state types.
- `go_server/domain/scrape_test.go` — state terminality and JSON contract tests.
- `go_server/internal/scrape/errors.go` — typed fetch/task errors and safe failure conversion.
- `go_server/internal/scrape/retry.go` — capped exponential retrying fetcher.
- `go_server/internal/scrape/retry_test.go` — transient/permanent/cancellation retry tests.
- `go_server/internal/scrape/outcome.go` — report aggregation and run-state classification.
- `go_server/internal/scrape/runner_test.go` — complete/partial/total/canceled runner tests.
- `go_server/internal/scrape/run_store.go` — `RunStore`, process-local locker, leases, and bounded memory store.
- `go_server/internal/scrape/run_store_test.go` — history ordering, retention, cloning, and stale-run recovery tests.
- `go_server/internal/scrape/postgres_coordination.go` — shared run store and advisory lock for PostgreSQL.
- `go_server/internal/scrape/postgres_coordination_test.go` — opt-in PostgreSQL lock/history integration tests.
- `go_server/internal/scrape/metrics.go` — scrape Prometheus observer.
- `go_server/internal/scrape/metrics_test.go` — metric names, labels, and values.
- `go_server/cmd/scrape/main_test.go` — state-to-process-exit-code tests.

### Modify

- `go_server/domain/comic.go` — replace the legacy scrape methods/status shape in `ComicUseCase`.
- `go_server/internal/scrape/types.go` — registration disposition/result, runner interfaces, publisher job, retry event, and observer interfaces.
- `go_server/internal/scrape/fetch.go` — return typed fetch failures instead of empty HTML.
- `go_server/internal/scrape/chrome_fetch.go` — return typed transport/status/challenge/read failures.
- `go_server/internal/scrape/fetch_test.go` — replace soft-failure assertions with typed error assertions.
- `go_server/internal/scrape/chrome_fetch_test.go` — add challenge and permanent TLS/URL classification coverage.
- `go_server/internal/scrape/publisher.go` — keep IDs and type/status constants; expose deterministic active publisher definitions.
- `go_server/internal/scrape/publishers.go` — classify an empty `200` document as a permanent parse failure; keep existing selectors.
- `go_server/internal/scrape/urls.go` — validate active publisher names, URL presence, URL schemes, and scraper registration without `sync.Once` hiding corrected configuration in tests.
- `go_server/internal/scrape/register.go` — return created/updated/unchanged/skipped dispositions.
- `go_server/internal/scrape/register_log.go` — log registration failures and return detailed registration results.
- `go_server/internal/scrape/register_lookup.go` — preserve title-merge behavior used by detailed registration.
- `go_server/internal/scrape/runner.go` — aggregate URL work into ordered per-publisher reports.
- `go_server/internal/scrape/scrape_test.go` — registration dispositions, empty-page parse failure, and fixture regressions.
- `go_server/internal/scrape/coordinator.go` — state machine, start/wait, status/history, cancellation, and drain.
- `go_server/internal/scrape/coordinator_test.go` — single-flight, transitions, timeout, cancellation, and drain.
- `go_server/internal/scrape/periodic.go` — consume terminal run states and shared cooldown timestamps.
- `go_server/internal/scrape/periodic_test.go` — cooldown after every terminal state and remote-busy behavior.
- `go_server/internal/usecase/comics.go` — expose start/status/history without flattening state.
- `go_server/internal/usecase/comics_test.go` — use-case forwarding tests.
- `go_server/api/route/comics.go` — `202/409/503`, status, history, and compatibility alias.
- `go_server/api/route/comics_test.go` — exact response/status tests.
- `go_server/api/route/auth_route_test.go` — unauthenticated, user-role, and admin-role policy coverage for every scrape route.
- `go_server/api/route/route.go` — mount all scrape routes on Workstream 1's existing root-path `catalogAdmin` group.
- `go_server/bootstrap/env.go` — bind and default `SCRAPE_TIMEOUT=30m`.
- `go_server/bootstrap/app.go` — select process-local SQLite or PostgreSQL coordination; lifecycle/CLI callers, not application shutters, own coordinator drain.
- `go_server/cmd/server/main.go` — pass lifecycle context and scrape timeout into coordinator/scheduler wiring supplied by bootstrap.
- `go_server/cmd/scrape/main.go` — signal-aware synchronous run and state-based exit codes.
- `src/frontend/components/Comics/Edition/ScrapeButton.tsx` — authenticated async trigger, polling, outcome-specific copy, and safe poll failure behavior.
- Rename `src/frontend/components/Comics/Edition/ScrapeButton.test.ts` to `src/frontend/components/Comics/Edition/ScrapeButton.test.tsx` — helper and rendered component coverage.
- `README.md` — final scrape states, endpoints, metrics, and SQLite/PostgreSQL deployment boundary.
- `docs/deploy-render.md` — one-replica SQLite constraint and PostgreSQL scaling path.
- `docs/superpowers/specs/2026-10-08-scrape-coordination.md` — supersede the old synchronous success-only contract with the implemented contract.

---

### Task 1: Define the Stable Scrape Domain Contract

**Files:**
- Create: `go_server/domain/scrape.go`
- Create: `go_server/domain/scrape_test.go`

**Interfaces:**
- Produces: `domain.ScrapeRun`, `domain.ScrapeRunStatus`, `domain.ScrapePublisherOutcome`, `domain.ScrapeTotals`, `domain.ScrapeFailure`.
- Produces: exact source/state/failure constants consumed by every later task.
- Consumes: only Go `time`.

- [ ] **Step 1: Write the failing domain contract tests**

```go
package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestScrapeRunStateTerminal(t *testing.T) {
	for _, state := range []ScrapeRunState{
		ScrapeRunComplete,
		ScrapeRunPartialFailure,
		ScrapeRunTotalFailure,
		ScrapeRunCanceled,
	} {
		if !state.Terminal() {
			t.Fatalf("%q should be terminal", state)
		}
	}
	for _, state := range []ScrapeRunState{ScrapeRunAccepted, ScrapeRunRunning} {
		if state.Terminal() {
			t.Fatalf("%q should not be terminal", state)
		}
	}
}

func TestScrapeRunJSONUsesStableSnakeCase(t *testing.T) {
	now := time.Date(2026, time.October, 26, 14, 0, 0, 0, time.UTC)
	run := ScrapeRun{
		ID:         "run-1",
		Source:     ScrapeSourceManual,
		State:      ScrapeRunPartialFailure,
		AcceptedAt: now,
		Totals: ScrapeTotals{
			PublishersTotal:    2,
			PublishersComplete: 1,
			PublishersFailed:   1,
		},
	}
	raw, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, field := range []string{
		`"id":"run-1"`,
		`"source":"manual"`,
		`"state":"partial_failure"`,
		`"accepted_at":"2026-10-26T14:00:00Z"`,
		`"publishers_total":2`,
	} {
		if !strings.Contains(body, field) {
			t.Fatalf("missing %s in %s", field, body)
		}
	}
}
```

- [ ] **Step 2: Run the tests and verify the contract is absent**

Run:

```bash
cd go_server
go test ./domain -run 'TestScrapeRun' -count=1
```

Expected: build failure containing `undefined: ScrapeRunState`.

- [ ] **Step 3: Add the exact domain types**

Create `go_server/domain/scrape.go` with this public contract:

```go
package domain

import "time"

type ScrapeSource string

const (
	ScrapeSourceManual    ScrapeSource = "manual"
	ScrapeSourceAutomatic ScrapeSource = "automatic"
	ScrapeSourceCLI       ScrapeSource = "cli"
)

type ScrapeRunState string

const (
	ScrapeRunAccepted       ScrapeRunState = "accepted"
	ScrapeRunRunning        ScrapeRunState = "running"
	ScrapeRunComplete       ScrapeRunState = "complete"
	ScrapeRunPartialFailure ScrapeRunState = "partial_failure"
	ScrapeRunTotalFailure   ScrapeRunState = "total_failure"
	ScrapeRunCanceled       ScrapeRunState = "canceled"
)

func (s ScrapeRunState) Terminal() bool {
	switch s {
	case ScrapeRunComplete, ScrapeRunPartialFailure, ScrapeRunTotalFailure, ScrapeRunCanceled:
		return true
	default:
		return false
	}
}

type ScrapePublisherState string

const (
	ScrapePublisherComplete       ScrapePublisherState = "complete"
	ScrapePublisherPartialFailure ScrapePublisherState = "partial_failure"
	ScrapePublisherFailed         ScrapePublisherState = "failed"
	ScrapePublisherCanceled       ScrapePublisherState = "canceled"
)

type ScrapeFailureKind string

const (
	ScrapeFailureFetchTransient ScrapeFailureKind = "fetch_transient"
	ScrapeFailureFetchPermanent ScrapeFailureKind = "fetch_permanent"
	ScrapeFailureParse          ScrapeFailureKind = "parse"
	ScrapeFailureConfig         ScrapeFailureKind = "config"
	ScrapeFailureRegistration   ScrapeFailureKind = "registration"
	ScrapeFailureCanceled       ScrapeFailureKind = "canceled"
)

type ScrapeFailure struct {
	Kind       ScrapeFailureKind `json:"kind"`
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	Publisher  string            `json:"publisher,omitempty"`
	URL        string            `json:"url,omitempty"`
	HTTPStatus int               `json:"http_status,omitempty"`
	Attempts   int               `json:"attempts,omitempty"`
	Retryable  bool              `json:"retryable"`
}

type ScrapePublisherOutcome struct {
	Publisher           string               `json:"publisher"`
	State               ScrapePublisherState `json:"state"`
	URLsTotal           int                  `json:"urls_total"`
	URLsSucceeded       int                  `json:"urls_succeeded"`
	URLsFailed          int                  `json:"urls_failed"`
	ItemsSeen           int                  `json:"items_seen"`
	Created             int                  `json:"created"`
	Updated             int                  `json:"updated"`
	Unchanged           int                  `json:"unchanged"`
	Skipped             int                  `json:"skipped"`
	Failed              int                  `json:"failed"`
	FetchRetries        int                  `json:"fetch_retries"`
	DurationMilliseconds int64               `json:"duration_ms"`
	Failures            []ScrapeFailure      `json:"failures"`
}

type ScrapeTotals struct {
	PublishersTotal    int `json:"publishers_total"`
	PublishersComplete int `json:"publishers_complete"`
	PublishersPartial  int `json:"publishers_partial"`
	PublishersFailed   int `json:"publishers_failed"`
	PublishersCanceled int `json:"publishers_canceled"`
	URLsTotal          int `json:"urls_total"`
	URLsSucceeded      int `json:"urls_succeeded"`
	URLsFailed         int `json:"urls_failed"`
	ItemsSeen          int `json:"items_seen"`
	Created            int `json:"created"`
	Updated            int `json:"updated"`
	Unchanged          int `json:"unchanged"`
	Skipped            int `json:"skipped"`
	Failed             int `json:"failed"`
	FetchRetries       int `json:"fetch_retries"`
}

type ScrapeRun struct {
	ID          string                   `json:"id"`
	Source      ScrapeSource             `json:"source"`
	State       ScrapeRunState           `json:"state"`
	AcceptedAt  time.Time                `json:"accepted_at"`
	StartedAt   *time.Time               `json:"started_at,omitempty"`
	FinishedAt  *time.Time               `json:"finished_at,omitempty"`
	Publishers  []ScrapePublisherOutcome `json:"publishers"`
	Totals      ScrapeTotals             `json:"totals"`
	Failures    []ScrapeFailure          `json:"failures"`
}

type ScrapeRunStatus struct {
	Running bool       `json:"running"`
	Active  *ScrapeRun `json:"active"`
	Latest  *ScrapeRun `json:"latest"`
}
```

Use `DurationMilliseconds` as the Go field name and `duration_ms` as JSON. Do not expose `error` strings or `error`-typed fields.

- [ ] **Step 4: Run the domain tests**

Run:

```bash
cd go_server
go test ./domain -run 'TestScrapeRun' -count=1
```

Expected:

```text
ok  	comics/domain
```

- [ ] **Step 5: Commit**

```bash
git add go_server/domain/scrape.go go_server/domain/scrape_test.go
git commit -m "feat(scrape): define reliable run outcomes"
```

---

### Task 2: Classify Fetch Failures and Retry Only Transient Fetches

**Files:**
- Create: `go_server/internal/scrape/errors.go`
- Create: `go_server/internal/scrape/retry.go`
- Create: `go_server/internal/scrape/retry_test.go`
- Modify: `go_server/internal/scrape/fetch.go`
- Modify: `go_server/internal/scrape/chrome_fetch.go`
- Modify: `go_server/internal/scrape/fetch_test.go`
- Modify: `go_server/internal/scrape/chrome_fetch_test.go`
- Modify: `go_server/internal/scrape/publishers.go`

**Interfaces:**
- Produces: `RetryPolicy`, `DefaultRetryPolicy()`, `RetryEvent`, `RetryOptions`, `NewRetryingFetcher(PageFetcher, RetryPolicy, RetryOptions) PageFetcher`.
- Produces: `FetchError` and `TaskError` with stable kinds/codes.
- Preserves: `PageFetcher.FetchHTML(context.Context, string) (string, error)`.
- Consumes: `domain.ScrapeFailureKind` from Task 1.

- [ ] **Step 1: Write the failing retry and HTTP classification tests**

```go
package scrape

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"comics/domain"
)

type sequenceFetcher struct {
	errors []error
	calls  int
}

func (f *sequenceFetcher) FetchHTML(_ context.Context, _ string) (string, error) {
	f.calls++
	if f.calls <= len(f.errors) {
		return "", f.errors[f.calls-1]
	}
	return "<html>ok</html>", nil
}

func TestRetryingFetcherRetriesTransientWithCappedBackoff(t *testing.T) {
	next := &sequenceFetcher{errors: []error{
		transientFetchError("https://example.test", "transport", errors.New("reset")),
		transientFetchError("https://example.test", "transport", errors.New("reset")),
		transientFetchError("https://example.test", "transport", errors.New("reset")),
	}}
	var delays []time.Duration
	fetcher := NewRetryingFetcher(next, DefaultRetryPolicy(), RetryOptions{
		Sleep: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
		Jitter: func(delay time.Duration, _ float64) time.Duration { return delay },
	})

	html, err := fetcher.FetchHTML(context.Background(), "https://example.test")
	if err != nil || html != "<html>ok</html>" {
		t.Fatalf("html=%q err=%v", html, err)
	}
	if next.calls != 4 {
		t.Fatalf("expected 4 attempts, got %d", next.calls)
	}
	want := []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second}
	if len(delays) != len(want) ||
		delays[0] != want[0] ||
		delays[1] != want[1] ||
		delays[2] != want[2] {
		t.Fatalf("delays=%v want=%v", delays, want)
	}
}

func TestJitterDelayUsesSymmetricTwentyPercentBounds(t *testing.T) {
	delay := 500 * time.Millisecond
	if got := jitterDelay(delay, 0.20, 0); got != 400*time.Millisecond {
		t.Fatalf("lower bound=%s", got)
	}
	if got := jitterDelay(delay, 0.20, 0.5); got != 500*time.Millisecond {
		t.Fatalf("midpoint=%s", got)
	}
	if got := jitterDelay(delay, 0.20, 1); got != 600*time.Millisecond {
		t.Fatalf("upper bound=%s", got)
	}
}

func TestRetryingFetcherDoesNotRetryPermanentFailure(t *testing.T) {
	next := &sequenceFetcher{errors: []error{
		permanentFetchError("https://example.test", "http_status", http.StatusNotFound, errors.New("404")),
	}}
	fetcher := NewRetryingFetcher(next, DefaultRetryPolicy(), RetryOptions{})
	_, err := fetcher.FetchHTML(context.Background(), "https://example.test")
	if err == nil || next.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, next.calls)
	}
}

func TestRetryingFetcherNeverRetriesParseOrConfigErrors(t *testing.T) {
	for _, kind := range []domain.ScrapeFailureKind{
		domain.ScrapeFailureParse,
		domain.ScrapeFailureConfig,
	} {
		next := &sequenceFetcher{errors: []error{&TaskError{
			Kind: kind,
			Code: "permanent_test",
			Err:  errors.New("permanent"),
		}}}
		fetcher := NewRetryingFetcher(next, DefaultRetryPolicy(), RetryOptions{})
		_, err := fetcher.FetchHTML(context.Background(), "https://example.test")
		if err == nil || next.calls != 1 {
			t.Fatalf("kind=%s err=%v calls=%d", kind, err, next.calls)
		}
	}
}

func TestRetryingFetcherCancellationInterruptsBackoff(t *testing.T) {
	next := &sequenceFetcher{errors: []error{
		transientFetchError("https://example.test", "transport", errors.New("reset")),
	}}
	ctx, cancel := context.WithCancel(context.Background())
	fetcher := NewRetryingFetcher(next, DefaultRetryPolicy(), RetryOptions{
		Sleep: func(ctx context.Context, _ time.Duration) error {
			cancel()
			<-ctx.Done()
			return ctx.Err()
		},
		Jitter: func(delay time.Duration, _ float64) time.Duration { return delay },
	})
	_, err := fetcher.FetchHTML(ctx, "https://example.test")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestHTTPFetcherReturnsTypedStatusErrors(t *testing.T) {
	for _, tc := range []struct {
		status    int
		transient bool
	}{
		{status: http.StatusServiceUnavailable, transient: true},
		{status: http.StatusTooManyRequests, transient: true},
		{status: http.StatusNotFound, transient: false},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
		}))
		fetcher := NewHTTPFetcher()
		_, err := fetcher.FetchHTML(context.Background(), server.URL)
		server.Close()
		if err == nil {
			t.Fatalf("status %d should fail", tc.status)
		}
		if IsTransientFetchError(err) != tc.transient {
			t.Fatalf("status %d transient=%v err=%v", tc.status, tc.transient, err)
		}
	}
}
```

- [ ] **Step 2: Run the tests and verify they fail before implementation**

Run:

```bash
cd go_server
go test ./internal/scrape -run 'TestRetryingFetcher|TestHTTPFetcherReturnsTypedStatusErrors' -count=1
```

Expected: build failure containing `undefined: NewRetryingFetcher`.

- [ ] **Step 3: Add typed errors with safe public conversion**

Implement these exact types in `errors.go`:

```go
package scrape

import (
	"context"
	"errors"
	"fmt"

	"comics/domain"
)

type FetchError struct {
	URL        string
	Code       string
	StatusCode int
	Attempts   int
	Transient  bool
	Err        error
}

func (e *FetchError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("fetch %s returned HTTP %d", e.URL, e.StatusCode)
	}
	return fmt.Sprintf("fetch %s failed: %v", e.URL, e.Err)
}

func (e *FetchError) Unwrap() error { return e.Err }

func transientFetchError(url, code string, err error) *FetchError {
	return &FetchError{URL: url, Code: code, Attempts: 1, Transient: true, Err: err}
}

func permanentFetchError(url, code string, status int, err error) *FetchError {
	return &FetchError{
		URL: url, Code: code, StatusCode: status, Attempts: 1, Transient: false, Err: err,
	}
}

func IsTransientFetchError(err error) bool {
	var fetchErr *FetchError
	return errors.As(err, &fetchErr) && fetchErr.Transient
}

type TaskError struct {
	Kind domain.ScrapeFailureKind
	Code string
	Err  error
}

func (e *TaskError) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *TaskError) Unwrap() error { return e.Err }

func safeFailure(publisher, pageURL string, err error) domain.ScrapeFailure {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return domain.ScrapeFailure{
			Kind: domain.ScrapeFailureCanceled, Code: "canceled",
			Message: "scrape work was canceled", Publisher: publisher, URL: pageURL,
		}
	}
	var fetchErr *FetchError
	if errors.As(err, &fetchErr) {
		kind := domain.ScrapeFailureFetchPermanent
		message := "publisher request failed"
		if fetchErr.Transient {
			kind = domain.ScrapeFailureFetchTransient
			message = "publisher request failed after transient retries"
		}
		return domain.ScrapeFailure{
			Kind: kind, Code: fetchErr.Code, Message: message,
			Publisher: publisher, URL: pageURL, HTTPStatus: fetchErr.StatusCode,
			Attempts: fetchErr.Attempts, Retryable: fetchErr.Transient,
		}
	}
	var taskErr *TaskError
	if errors.As(err, &taskErr) {
		message := map[domain.ScrapeFailureKind]string{
			domain.ScrapeFailureParse:        "publisher page could not be parsed",
			domain.ScrapeFailureConfig:       "publisher configuration is invalid",
			domain.ScrapeFailureRegistration: "catalog registration failed",
		}[taskErr.Kind]
		return domain.ScrapeFailure{
			Kind: taskErr.Kind, Code: taskErr.Code, Message: message,
			Publisher: publisher, URL: pageURL,
		}
	}
	return domain.ScrapeFailure{
		Kind: domain.ScrapeFailureRegistration, Code: "unexpected",
		Message: "scrape task failed", Publisher: publisher, URL: pageURL,
	}
}
```

Only `safeFailure` output may enter `domain.ScrapeRun`. Log the original `err` separately.

- [ ] **Step 4: Implement the retrying fetch decorator**

Use these exact defaults and interfaces in `retry.go`:

```go
package scrape

import (
	"context"
	"errors"
	"math"
	rand "math/rand/v2"
	"time"
)

type RetryPolicy struct {
	MaxAttempts   int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	JitterFraction float64
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 4, BaseDelay: 500 * time.Millisecond,
		MaxDelay: 2 * time.Second, JitterFraction: 0.20,
	}
}

type RetryEvent struct {
	URL     string
	Attempt int
	Delay   time.Duration
	Code    string
}

type RetryOptions struct {
	Sleep   func(context.Context, time.Duration) error
	Jitter  func(time.Duration, float64) time.Duration
	Observe func(RetryEvent)
}

type retryingFetcher struct {
	next   PageFetcher
	policy RetryPolicy
	opts   RetryOptions
}

func NewRetryingFetcher(next PageFetcher, policy RetryPolicy, opts RetryOptions) PageFetcher {
	if opts.Sleep == nil {
		opts.Sleep = func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		}
	}
	if opts.Jitter == nil {
		opts.Jitter = func(delay time.Duration, fraction float64) time.Duration {
			return jitterDelay(delay, fraction, rand.Float64())
		}
	}
	return &retryingFetcher{next: next, policy: policy, opts: opts}
}

func (f *retryingFetcher) FetchHTML(ctx context.Context, pageURL string) (string, error) {
	for attempt := 1; attempt <= f.policy.MaxAttempts; attempt++ {
		html, err := f.next.FetchHTML(ctx, pageURL)
		if err == nil {
			return html, nil
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		var fetchErr *FetchError
		if !errors.As(err, &fetchErr) || !fetchErr.Transient || attempt == f.policy.MaxAttempts {
			if errors.As(err, &fetchErr) {
				fetchErr.Attempts = attempt
			}
			return "", err
		}
		exponent := math.Pow(2, float64(attempt-1))
		delay := time.Duration(float64(f.policy.BaseDelay) * exponent)
		if delay > f.policy.MaxDelay {
			delay = f.policy.MaxDelay
		}
		delay = f.opts.Jitter(delay, f.policy.JitterFraction)
		if f.opts.Observe != nil {
			f.opts.Observe(RetryEvent{
				URL: pageURL, Attempt: attempt + 1, Delay: delay, Code: fetchErr.Code,
			})
		}
		if err := f.opts.Sleep(ctx, delay); err != nil {
			return "", err
		}
	}
	return "", context.Canceled
}

func jitterDelay(
	delay time.Duration,
	fraction float64,
	sample float64,
) time.Duration {
	factor := 1 - fraction + sample*(2*fraction)
	return time.Duration(float64(delay) * factor)
}
```

The final return is unreachable with a valid policy. Validate `MaxAttempts >= 1`, `BaseDelay > 0`, `MaxDelay >= BaseDelay`, and `JitterFraction` in `0..1` in `NewRetryingFetcher`; panic with a precise configuration message because these are programmer constants, not user input.

- [ ] **Step 5: Make both concrete fetchers return typed failures**

Change `HTTPFetcher.FetchHTML` and `ChromeTLSFetcher.fetchURL` so they obey this matrix:

- `http.NewRequestWithContext` error: permanent, code `invalid_request`.
- TLS client construction error: permanent, code `tls_client_init`.
- `client.Do` with canceled context: return `ctx.Err()`.
- TLS certificate validation error: permanent, code `tls_certificate`.
- Other `client.Do` errors: transient, code `transport`.
- HTTP `408`, `425`, `429`, `500`, `502`, `503`, `504`: transient, code `http_status`, with `StatusCode`.
- Any other non-`200`: permanent, code `http_status`, with `StatusCode`.
- `io.ErrUnexpectedEOF`: transient, code `truncated_body`.
- Other body read errors: transient, code `read_body`.
- Body containing `Just a moment`: transient, code `cloudflare_challenge`.
- Successful `200`: return the HTML without logging.

Delete the old warning-and-empty-string branches. Add a shared `httpStatusFetchError(pageURL string, status int) error` helper so the standard and TLS fetchers cannot drift.
Read the limited response body before final status classification so a
Cloudflare challenge body is transient even when delivered with `403`; a
generic `403` without that marker remains permanent.

In `loadDocument`, reject an empty/whitespace `200` body without retry:

```go
html, err := fetcher.FetchHTML(ctx, url)
if err != nil {
	return nil, err
}
if strings.TrimSpace(html) == "" {
	return nil, &TaskError{
		Kind: domain.ScrapeFailureParse,
		Code: "empty_document",
		Err:  errors.New("publisher returned an empty document"),
	}
}
return goquery.NewDocumentFromReader(strings.NewReader(html))
```

Add `errors` and `comics/domain` imports to `publishers.go`.

- [ ] **Step 6: Replace the old soft-failure assertions**

Delete `TestFetchHTMLSoftFailsOnNon200`. Keep the `200` body/header tests and add:

- `404` is a permanent `FetchError` with `StatusCode == 404` and `Attempts == 1`.
- `503` is transient.
- canceled request returns `context.Canceled`.
- Cloudflare challenge is transient in `chrome_fetch_test.go`.
- empty `200` is a permanent `TaskError` with kind `parse` when passed through `loadDocument`.

- [ ] **Step 7: Run focused tests**

Run:

```bash
cd go_server
go test ./internal/scrape -run 'TestRetryingFetcher|TestHTTPFetcher|TestFetchHTML|TestNewPageFetcher|TestNelomanga|TestLoadDocument' -count=1
```

Expected:

```text
ok  	comics/internal/scrape
```

- [ ] **Step 8: Commit**

```bash
git add \
  go_server/internal/scrape/errors.go \
  go_server/internal/scrape/retry.go \
  go_server/internal/scrape/retry_test.go \
  go_server/internal/scrape/fetch.go \
  go_server/internal/scrape/chrome_fetch.go \
  go_server/internal/scrape/fetch_test.go \
  go_server/internal/scrape/chrome_fetch_test.go \
  go_server/internal/scrape/publishers.go
git commit -m "feat(scrape): retry transient fetch failures"
```

---

### Task 3: Return Detailed Registration and Publisher Outcomes

**Files:**
- Create: `go_server/internal/scrape/outcome.go`
- Create: `go_server/internal/scrape/runner_test.go`
- Modify: `go_server/internal/scrape/types.go`
- Modify: `go_server/internal/scrape/publisher.go`
- Modify: `go_server/internal/scrape/urls.go`
- Modify: `go_server/internal/scrape/register.go`
- Modify: `go_server/internal/scrape/register_log.go`
- Modify: `go_server/internal/scrape/register_lookup.go`
- Modify: `go_server/internal/scrape/runner.go`
- Modify: `go_server/internal/scrape/scrape_test.go`

**Interfaces:**
- Produces: `RegistrationDisposition`, `RegistrationResult`, `Registrar.RegisterWithResult`.
- Produces: `Runner`, `RunnerOptions`, `RunExecutor`, `RunExecutorFunc`, `RunReport`.
- Produces: deterministic `PublisherJob` loading in active order.
- Consumes: Task 1 domain types and Task 2 retrying fetcher.

- [ ] **Step 1: Write failing registration and run-classification tests**

Add these assertions to `scrape_test.go`:

```go
func TestRegistrarReportsCreatedUnchangedAndDuplicate(t *testing.T) {
	ctx := context.Background()
	repo := newTestScrapeRepo(t)
	registrar := NewRegistrar(repo, NewRunState())
	raw := ScrapedComic{
		Chapter: "Chapter 7", Title: "Outcome hero",
		ComType: "manhwa", Status: "ongoing",
	}

	created, err := registrar.RegisterWithResult(ctx, raw, PublisherAsura)
	if err != nil || created.Disposition != RegistrationCreated || created.ComicID == 0 {
		t.Fatalf("created=%#v err=%v", created, err)
	}
	duplicate, err := registrar.RegisterWithResult(ctx, raw, PublisherAsura)
	if err != nil || duplicate.Disposition != RegistrationSkippedDuplicate {
		t.Fatalf("duplicate=%#v err=%v", duplicate, err)
	}

	secondRun := NewRegistrar(repo, NewRunState())
	unchanged, err := secondRun.RegisterWithResult(ctx, raw, PublisherAsura)
	if err != nil || unchanged.Disposition != RegistrationUnchanged {
		t.Fatalf("unchanged=%#v err=%v", unchanged, err)
	}
}
```

Create `runner_test.go` with deterministic jobs:

```go
package scrape

import (
	"context"
	"errors"
	"testing"

	"comics/domain"
)

func registeringScraper(title string) PublisherScraper {
	return func(
		ctx context.Context,
		pageURL string,
		publisherID int,
		fetcher PageFetcher,
		register func(ScrapedComic, int) error,
	) error {
		if _, err := fetcher.FetchHTML(ctx, pageURL); err != nil {
			return err
		}
		return register(ScrapedComic{
			Chapter: "1", Title: title, ComType: "manhwa", Status: "ongoing",
		}, publisherID)
	}
}

func TestRunnerClassifiesPartialFailure(t *testing.T) {
	repo := newTestScrapeRepo(t)
	runner := NewRunner(repo, staticFetcher{
		"fixture://ok": "<html>ok</html>",
	}, RunnerOptions{
		LoadJobs: func() ([]PublisherJob, error) {
			return []PublisherJob{
				{Name: "Good", ID: 1, URLs: []string{"fixture://ok"}, Scraper: registeringScraper("Good hero")},
				{
					Name: "Broken", ID: 2, URLs: []string{"fixture://broken"},
					Scraper: registeringScraper("Broken hero"),
				},
			}, nil
		},
		Retry: DefaultRetryPolicy(),
	})

	report, err := runner.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state := ClassifyRun(report, nil); state != domain.ScrapeRunPartialFailure {
		t.Fatalf("state=%s report=%#v", state, report)
	}
	if len(report.Publishers) != 2 || report.Publishers[0].Publisher != "Good" {
		t.Fatalf("publishers=%#v", report.Publishers)
	}
}

func TestClassifyRunTotalFailureAndCancellation(t *testing.T) {
	failed := RunReport{
		Publishers: []domain.ScrapePublisherOutcome{{
			Publisher: "Broken", State: domain.ScrapePublisherFailed,
			URLsTotal: 1, URLsFailed: 1,
		}},
	}
	if got := ClassifyRun(failed, nil); got != domain.ScrapeRunTotalFailure {
		t.Fatalf("got %s", got)
	}
	if got := ClassifyRun(failed, context.Canceled); got != domain.ScrapeRunCanceled {
		t.Fatalf("got %s", got)
	}
}

func TestClassifyRunFatalConfigurationFailure(t *testing.T) {
	report := RunReport{Failures: []domain.ScrapeFailure{{
		Kind: domain.ScrapeFailureConfig, Code: "config_json",
		Message: "publisher configuration is invalid",
	}}}
	if got := ClassifyRun(report, errors.New("config")); got != domain.ScrapeRunTotalFailure {
		t.Fatalf("got %s", got)
	}
}
```

Make `staticFetcher` return a permanent `FetchError` for absent keys instead of `("", nil)` so `fixture://broken` is an explicit failed URL.

- [ ] **Step 2: Run the tests and verify missing detailed outcomes**

Run:

```bash
cd go_server
go test ./internal/scrape -run 'TestRegistrarReports|TestRunnerClassifies|TestClassifyRun' -count=1
```

Expected: build failure containing `RegisterWithResult undefined` and `undefined: RunReport`.

- [ ] **Step 3: Add detailed registration results without changing publisher selectors**

Add to `types.go`:

```go
type RegistrationDisposition string

const (
	RegistrationCreated          RegistrationDisposition = "created"
	RegistrationUpdated          RegistrationDisposition = "updated"
	RegistrationUnchanged        RegistrationDisposition = "unchanged"
	RegistrationSkippedInvalid   RegistrationDisposition = "skipped_invalid"
	RegistrationSkippedDuplicate RegistrationDisposition = "skipped_duplicate"
)

type RegistrationResult struct {
	Disposition RegistrationDisposition
	ComicID     int
}
```

Implement:

```go
func (r *Registrar) Register(ctx context.Context, raw ScrapedComic, publisherID int) error {
	_, err := r.RegisterWithResult(ctx, raw, publisherID)
	return err
}

func (r *Registrar) RegisterWithResult(
	ctx context.Context,
	raw ScrapedComic,
	publisherID int,
) (RegistrationResult, error)
```

Apply these exact disposition rules:

- Normalization returns `ok == false`: `skipped_invalid`, no error.
- The same publisher/discovery key already exists in `RunState`: `skipped_duplicate`, no error.
- Repository create succeeds: `created` with the new comic ID.
- Existing comic changes titles, type, publisher IDs, chapter, author, status, cover, or visibility: update once and return `updated`.
- Existing comic has no changed registration-owned fields: do not call `repo.Update`; return `unchanged`.
- Repository/transaction failure: zero `RegistrationResult` plus the wrapped error.

Use `slices.Equal` for title/publisher comparisons. Continue to update `LastUpdate` only when `CurrentChap` increases.

Change `registerWithLogging` to:

```go
func registerWithLogging(
	ctx context.Context,
	publisherName string,
	registrar *Registrar,
) func(ScrapedComic, int) (RegistrationResult, error)
```

It logs the original error with `publisher` and `title` and returns the result/error unchanged.

- [ ] **Step 4: Replace map iteration with deterministic publisher jobs**

Add to `types.go`:

```go
type PublisherJob struct {
	Name          string
	ID            int
	URLs          []string
	SequentialURL bool
	Scraper       PublisherScraper
	ConfigFailure *domain.ScrapeFailure
}

type RunReport struct {
	Publishers []domain.ScrapePublisherOutcome
	Totals     domain.ScrapeTotals
	Failures   []domain.ScrapeFailure
}

type RunExecutor interface {
	Run(context.Context) (RunReport, error)
}

type RunExecutorFunc func(context.Context) (RunReport, error)

func (f RunExecutorFunc) Run(ctx context.Context) (RunReport, error) {
	return f(ctx)
}
```

Replace `publisherScrapers`, `publisherIDs`, `sequentialURLPublishers`, and the separate active-name slice with one ordered definition slice in `publisher.go`:

```go
type publisherDefinition struct {
	Name          string
	ID            int
	SequentialURL bool
	Scraper       PublisherScraper
}

var activePublisherDefinitions = []publisherDefinition{
	{Name: "ManhuaPlus", ID: PublisherManhuaPlus, Scraper: scrapeManhuaPlusPage},
	{Name: "Asura", ID: PublisherAsura, Scraper: scrapeAsuraPage},
	{Name: "RealmScans", ID: PublisherRealmScans, Scraper: scrapeRealmPage},
	{Name: "DemonicScans", ID: PublisherDemonicScans, Scraper: scrapeDemonicPage},
	{
		Name: "Manganato", ID: PublisherManganato,
		SequentialURL: true, Scraper: scrapeManganatoPage,
	},
}
```

Keep the inactive Flame parser and `PublisherFlameScans` constant; it has an empty URL list and is not an active definition.

Change `publisherURLPairs` into:

```go
func publisherJobs() ([]PublisherJob, error)
```

Rules:

- Read and unmarshal `config/url_switch.json` on every call. Remove package-level `sync.Once`; a corrected mounted file must be visible without a process restart.
- A missing/unreadable/malformed file returns one fatal `TaskError` with kind `config` and code `config_read` or `config_json`.
- For each active definition, preserve definition order and URL order.
- Missing key, empty list, blank URL, unsupported scheme, missing scraper, or ID zero creates that publisher's `ConfigFailure` with code `missing_urls`, `invalid_url`, or `missing_scraper`; it does not suppress healthy publishers.
- Accept only `http` and `https` schemes from the file. Tests inject `fixture` jobs through `RunnerOptions.LoadJobs` and do not pass through file validation.

- [ ] **Step 5: Implement publisher/page aggregation**

Add to `types.go`:

```go
type Observer interface {
	RunStarted(domain.ScrapeRun)
	RunFinished(domain.ScrapeRun)
	PublisherFinished(domain.ScrapePublisherOutcome)
	FetchRetry(string, RetryEvent)
}

type NopObserver struct{}

func (NopObserver) RunStarted(domain.ScrapeRun) {}
func (NopObserver) RunFinished(domain.ScrapeRun) {}
func (NopObserver) PublisherFinished(domain.ScrapePublisherOutcome) {}
func (NopObserver) FetchRetry(string, RetryEvent) {}

type RunnerOptions struct {
	LoadJobs func() ([]PublisherJob, error)
	Retry    RetryPolicy
	Observer Observer
}
```

`NewRunner` and `Run` signatures are:

```go
func NewRunner(
	repo domain.ComicRepository,
	fetcher PageFetcher,
	opts RunnerOptions,
) *Runner

func (r *Runner) Run(ctx context.Context) (RunReport, error)
```

Implementation requirements:

1. Default `fetcher` to `NewPageFetcher`, `LoadJobs` to `publisherJobs`, retry policy to `DefaultRetryPolicy`, and observer to `NopObserver`.
2. Create one `Registrar`/`RunState` per run and one `registerMu`; all registration remains serialized.
3. Launch one goroutine per `PublisherJob`, store its result by original index, and return outcomes in active definition order.
4. Within a publisher, run URLs concurrently unless `SequentialURL` is true.
5. Wrap the concrete fetcher in `NewRetryingFetcher` per page. Increment `FetchRetries` and call `Observer.FetchRetry(publisher, event)` from `RetryOptions.Observe`.
6. Wrap registration so every invocation increments `ItemsSeen` and exactly one of `Created`, `Updated`, `Unchanged`, `Skipped`, or `Failed`.
7. If a scraper returns nil but called registration zero times, emit a permanent parse failure with code `no_items`.
8. If registration failed for some items, mark that page failed with kind `registration`, code `register_failed`, while preserving successful counts.
9. Publisher state:
   - all URLs succeeded: `complete`;
   - at least one URL succeeded and at least one failed: `partial_failure`;
   - no URL succeeded: `failed`;
   - context ended: `canceled`.
10. Call `Observer.PublisherFinished` once after aggregation.
11. Return `ctx.Err()` only for cancellation/deadline. Fatal configuration read/JSON errors go in `RunReport.Failures` and are also returned for internal logging.

The coordinator adds the run ID to the runner context before execution:

```go
type runIDContextKey struct{}

func withRunID(ctx context.Context, runID string) context.Context {
	return context.WithValue(ctx, runIDContextKey{}, runID)
}

func runIDFromContext(ctx context.Context) string {
	runID, _ := ctx.Value(runIDContextKey{}).(string)
	return runID
}
```

Use `runIDFromContext(ctx)` in `registerWithLogging` and the runner's
page-failure log. Those logs include the original wrapped error plus `run_id`,
`publisher`, and `url`. The observer does not need a run-ID argument because
run IDs are deliberately excluded from metric labels.

Keep the legacy signature only so the Task 3 commit compiles before Task 4
replaces the coordinator:

```go
func RunAll(
	ctx context.Context,
	repo domain.ComicRepository,
	fetcher PageFetcher,
) error {
	_, err := NewRunner(repo, fetcher, RunnerOptions{}).Run(ctx)
	return err
}
```

Replace `TestRunAllContinuesAfterFetchFailure` with
`TestRunnerReportsTotalFailureWhenEveryConfiguredPageFails`. Call
`NewRunner(...).Run`, assert that the report classifies as `total_failure`, and
assert every failed URL appears in its publisher outcome. Task 4 deletes
`RunAll` after all production callers use `RunExecutor`, so the final code has
no error-only API that can suppress publisher outcomes.

- [ ] **Step 6: Add pure aggregation and classification**

In `outcome.go`, implement:

```go
func SummarizePublishers(items []domain.ScrapePublisherOutcome) domain.ScrapeTotals

func ClassifyRun(report RunReport, runErr error) domain.ScrapeRunState {
	if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
		return domain.ScrapeRunCanceled
	}
	failed := len(report.Failures) > 0
	succeeded := false
	for _, item := range report.Publishers {
		if item.URLsSucceeded > 0 {
			succeeded = true
		}
		if item.State != domain.ScrapePublisherComplete {
			failed = true
		}
	}
	if !failed && len(report.Publishers) > 0 {
		return domain.ScrapeRunComplete
	}
	if succeeded {
		return domain.ScrapeRunPartialFailure
	}
	return domain.ScrapeRunTotalFailure
}
```

`SummarizePublishers` must sum every numeric publisher field and count publisher states. The runner must assign `report.Totals = SummarizePublishers(report.Publishers)` before returning.

- [ ] **Step 7: Run scrape package tests**

Run:

```bash
cd go_server
go test ./internal/scrape -run 'TestRegistrarReports|TestRunner|TestClassifyRun|TestScrape|TestNormalize|TestAsura|TestFlame' -count=1
```

Expected:

```text
ok  	comics/internal/scrape
```

- [ ] **Step 8: Commit**

```bash
git add \
  go_server/internal/scrape/outcome.go \
  go_server/internal/scrape/runner_test.go \
  go_server/internal/scrape/types.go \
  go_server/internal/scrape/publisher.go \
  go_server/internal/scrape/urls.go \
  go_server/internal/scrape/register.go \
  go_server/internal/scrape/register_log.go \
  go_server/internal/scrape/register_lookup.go \
  go_server/internal/scrape/runner.go \
  go_server/internal/scrape/scrape_test.go
git commit -m "feat(scrape): report publisher outcomes"
```

---

### Task 4: Add Process-Local Single-Flight, Accurate History, Cancellation, and Drain

**Files:**
- Create: `go_server/internal/scrape/run_store.go`
- Create: `go_server/internal/scrape/run_store_test.go`
- Modify: `go_server/internal/scrape/coordinator.go`
- Modify: `go_server/internal/scrape/coordinator_test.go`
- Modify: `go_server/internal/scrape/periodic.go`
- Modify: `go_server/internal/scrape/periodic_test.go`

**Interfaces:**
- Produces: `RunStore`, `RunLocker`, `RunLease`, `MemoryRunStore`, `LocalRunLocker`.
- Produces: `Coordinator.Start`, `Coordinator.Run`, `Coordinator.Wait`, `Coordinator.Status`, `Coordinator.History`, `Coordinator.Shutdown`.
- Produces: `Coordinator.DelayUntilNextAutomatic(context.Context, time.Duration)`.
- Consumes: `RunExecutor`, `RunReport`, `ClassifyRun`, and `Observer`.

- [ ] **Step 1: Write failing store and coordinator lifecycle tests**

Create `run_store_test.go`:

```go
package scrape

import (
	"context"
	"fmt"
	"testing"
	"time"

	"comics/domain"
)

func TestMemoryRunStoreRetainsNewestTerminalRuns(t *testing.T) {
	store := NewMemoryRunStore(2)
	ctx := context.Background()
	for i := 1; i <= 3; i++ {
		finished := time.Unix(int64(i), 0).UTC()
		run := domain.ScrapeRun{
			ID: fmt.Sprintf("run-%d", i), Source: domain.ScrapeSourceManual,
			State: domain.ScrapeRunComplete, AcceptedAt: finished, FinishedAt: &finished,
		}
		if err := store.Save(ctx, run); err != nil {
			t.Fatal(err)
		}
	}
	history, err := store.History(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].ID != "run-3" || history[1].ID != "run-2" {
		t.Fatalf("history=%#v", history)
	}
}
```

Add to `coordinator_test.go`:

```go
func TestCoordinatorStartTransitionsAndRejectsOverlap(t *testing.T) {
	release := make(chan struct{})
	runner := RunExecutorFunc(func(ctx context.Context) (RunReport, error) {
		select {
		case <-release:
			return RunReport{Publishers: []domain.ScrapePublisherOutcome{{
				Publisher: "Fixture", State: domain.ScrapePublisherComplete,
				URLsTotal: 1, URLsSucceeded: 1,
			}}}, nil
		case <-ctx.Done():
			return RunReport{}, ctx.Err()
		}
	})
	coord := NewCoordinatorWithOptions(context.Background(), runner, CoordinatorOptions{
		Store: NewMemoryRunStore(50), Locker: NewLocalRunLocker(),
		RunTimeout: time.Minute, Observer: NopObserver{},
		NewRunID: func() string { return "run-1" },
	})

	accepted, err := coord.Start(context.Background(), domain.ScrapeSourceManual)
	if err != nil || accepted.State != domain.ScrapeRunAccepted {
		t.Fatalf("accepted=%#v err=%v", accepted, err)
	}
	_, err = coord.Start(context.Background(), domain.ScrapeSourceAutomatic)
	if !IsScrapeInProgress(err) {
		t.Fatalf("expected in-progress error, got %v", err)
	}
	close(release)
	finished, err := coord.Wait(context.Background(), accepted.ID)
	if err != nil || finished.State != domain.ScrapeRunComplete {
		t.Fatalf("finished=%#v err=%v", finished, err)
	}
}

func TestCoordinatorShutdownCancelsAndDrains(t *testing.T) {
	started := make(chan struct{})
	runner := RunExecutorFunc(func(ctx context.Context) (RunReport, error) {
		close(started)
		<-ctx.Done()
		return RunReport{}, ctx.Err()
	})
	coord := NewCoordinatorWithOptions(context.Background(), runner, CoordinatorOptions{
		Store: NewMemoryRunStore(50), Locker: NewLocalRunLocker(),
		RunTimeout: time.Minute, Observer: NopObserver{},
		NewRunID: func() string { return "run-cancel" },
	})
	if _, err := coord.Start(context.Background(), domain.ScrapeSourceManual); err != nil {
		t.Fatal(err)
	}
	<-started
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := coord.Shutdown(shutdownCtx); err != nil {
		t.Fatal(err)
	}
	status, err := coord.Status(context.Background())
	if err != nil || status.Latest == nil || status.Latest.State != domain.ScrapeRunCanceled {
		t.Fatalf("status=%#v err=%v", status, err)
	}
}
```

- [ ] **Step 2: Run tests and verify missing store/coordinator APIs**

Run:

```bash
cd go_server
go test ./internal/scrape -run 'TestMemoryRunStore|TestCoordinatorStartTransitions|TestCoordinatorShutdown' -count=1
```

Expected: build failure containing `undefined: NewMemoryRunStore`.

- [ ] **Step 3: Add the store and locker contracts**

Use these signatures in `run_store.go`:

```go
type RunStore interface {
	Save(context.Context, domain.ScrapeRun) error
	Active(context.Context) (*domain.ScrapeRun, error)
	Latest(context.Context) (*domain.ScrapeRun, error)
	History(context.Context, int) ([]domain.ScrapeRun, error)
	RecoverActive(context.Context, time.Time, domain.ScrapeFailure) error
	Close() error
}

type RunLease interface {
	Release(context.Context) error
}

type RunLocker interface {
	TryAcquire(context.Context) (RunLease, bool, error)
}

func NewMemoryRunStore(historyLimit int) *MemoryRunStore
func NewLocalRunLocker() *LocalRunLocker
```

Implementation rules:

- `MemoryRunStore` stores deep copies under a mutex; copy publisher and failure slices on save/read so route serialization cannot race with mutation.
- `Active` returns the newest `accepted` or `running` run.
- `Latest` returns the newest terminal run.
- `History` returns newest terminal first, clamps to the caller's positive limit, and never returns active runs.
- Saving a terminal run prunes older terminal entries beyond `historyLimit`, preserving active entries.
- `RecoverActive` changes every accepted/running entry to `canceled`, assigns the supplied finish time, appends the supplied safe failure, and retains totals/outcomes.
- `LocalRunLocker` is a buffered channel of size one. `TryAcquire` is non-blocking. Lease release is idempotent through `sync.Once`.

- [ ] **Step 4: Replace coordinator booleans with the run state machine**

Use these exact public types/signatures:

```go
const (
	DefaultScrapeTimeout = 30 * time.Minute
	DefaultHistoryLimit  = 50
)

type CoordinatorOptions struct {
	Store      RunStore
	Locker     RunLocker
	Observer   Observer
	RunTimeout time.Duration
	Clock      func() time.Time
	NewRunID   func() string
}

type ErrScrapeInProgress struct {
	Active *domain.ScrapeRun
}

var ErrCoordinatorClosed = errors.New("scrape coordinator is shutting down")

func (e *ErrScrapeInProgress) Error() string {
	if e.Active == nil {
		return "scrape already in progress"
	}
	return "scrape already in progress (source=" + string(e.Active.Source) + ")"
}

func IsScrapeInProgress(err error) bool {
	var inProgress *ErrScrapeInProgress
	return errors.As(err, &inProgress)
}

func NewCoordinatorWithOptions(
	lifecycle context.Context,
	runner RunExecutor,
	opts CoordinatorOptions,
) *Coordinator

func NewProcessLocalCoordinator(
	lifecycle context.Context,
	repo domain.ComicRepository,
	observer Observer,
	runTimeout time.Duration,
) *Coordinator

func (c *Coordinator) Start(
	triggerCtx context.Context,
	source domain.ScrapeSource,
) (domain.ScrapeRun, error)

func (c *Coordinator) RunAndWait(
	ctx context.Context,
	source domain.ScrapeSource,
) (domain.ScrapeRun, error)

func (c *Coordinator) Wait(ctx context.Context, runID string) (domain.ScrapeRun, error)
func (c *Coordinator) Status(ctx context.Context) (domain.ScrapeRunStatus, error)
func (c *Coordinator) History(ctx context.Context, limit int) ([]domain.ScrapeRun, error)
func (c *Coordinator) DelayUntilNextAutomatic(ctx context.Context, interval time.Duration) (time.Duration, error)
func (c *Coordinator) WaitUntilIdle(ctx context.Context) error
func (c *Coordinator) Shutdown(ctx context.Context) error
```

Keep these compatibility adapters through the Task 4 commit so the unchanged
use case, bootstrap, and CLI continue to compile:

```go
type Source = domain.ScrapeSource

const (
	SourceManual    = domain.ScrapeSourceManual
	SourceAutomatic = domain.ScrapeSourceAutomatic
	SourceCLI       = domain.ScrapeSourceCLI
)

type RunStatus struct {
	Running             bool
	ActiveSource        Source
	LastCompletedAt     time.Time
	LastCompletedSource Source
}

func NewCoordinator(repo domain.ComicRepository) *Coordinator {
	return NewProcessLocalCoordinator(
		context.Background(),
		repo,
		NopObserver{},
		DefaultScrapeTimeout,
	)
}

func (c *Coordinator) Run(ctx context.Context, source Source) error {
	run, err := c.RunAndWait(ctx, source)
	if err != nil {
		return err
	}
	if run.State != domain.ScrapeRunComplete {
		return &ErrScrapeRunFailed{State: run.State}
	}
	return nil
}

func (c *Coordinator) Snapshot() RunStatus
```

`Snapshot` maps the new store-backed active/latest records into the old fields;
it never owns a second state copy. `ErrScrapeRunFailed` includes the terminal
state in `Error()`:

```go
type ErrScrapeRunFailed struct {
	State domain.ScrapeRunState
}

func (e *ErrScrapeRunFailed) Error() string {
	return "scrape finished with state=" + string(e.State)
}
```

Task 7 removes every adapter after all callers move to the structured methods.

State transition requirements:

1. `Start` rejects after shutdown begins.
2. Acquire the `RunLease` before creating a run. If acquisition returns `false`, query `RunStore.Active` and return `ErrScrapeInProgress{Active: active}`.
3. After acquiring the lease, call `RecoverActive` with code `interrupted_owner` before saving the new run. This is a no-op in healthy process-local operation and repairs stale PostgreSQL rows after a crashed owner.
4. Create an `accepted` run with `xid.New().String()` by default and a UTC `AcceptedAt`.
5. Save `accepted`, create a run context rooted in the coordinator lifecycle with `RunTimeout`, retain its cancel function/done channel, and launch execution.
6. Execution saves `running` with `StartedAt`, calls `Observer.RunStarted`, invokes the runner with `withRunID(runCtx, run.ID)`, assigns report publishers/totals/failures, classifies the terminal state, and saves `FinishedAt`.
7. If the run context ended, append one safe failure with kind `canceled` and code `deadline_exceeded` for a timeout or `canceled` for explicit shutdown.
8. Save the terminal run before releasing the lease. Then call `Observer.RunFinished`, release the lease, publish the result to waiters, and close the done channel.
9. `Run` starts with the caller context as the run context and waits for terminal persistence even when that context is canceled; it returns the recorded `canceled` run. `Start` uses the coordinator lifecycle so HTTP disconnection cannot cancel an accepted run.
10. `Wait` returns the terminal run for a locally started ID. Return `context.Canceled` only when the waiting caller stops; the scrape continues.
11. `Status` reads `Active` and `Latest` from the store, then probes the non-blocking locker. If an active row exists but the lock is acquirable, recover that stale row as `canceled`, release the probe lease, and reread status. If no active row exists and the lock is not acquirable, return `Running: true` with `Active: nil` for the brief interval before the remote owner persists `accepted`. It does not infer success from an idle flag.
12. `DelayUntilNextAutomatic` uses `Latest.FinishedAt` for every terminal state, including failures and cancellation.
13. `WaitUntilIdle` polls the shared store every `250ms`; this also works when PostgreSQL reports a run owned by another replica.
14. `Shutdown` atomically rejects new starts, cancels the active local run, waits for its done channel within the supplied context, then closes the store. Return the context error if drain exceeds its bound.

Terminal persistence and advisory-lock release must not use an already canceled
run context. Create a bounded cleanup context with:

```go
finishCtx, finishCancel := context.WithTimeout(
	context.WithoutCancel(c.lifecycle),
	5*time.Second,
)
defer finishCancel()
```

Use `finishCtx` for the terminal `Save` and lease `Release`, including canceled
and timed-out runs.

Delete legacy `Running` and `LastCompleted`; the compatibility `Snapshot`
method above is store-backed rather than boolean-backed. Delete the Task 3
`RunAll` compatibility wrapper in this task; the coordinator
now receives a `RunExecutor` and no production path may discard `RunReport`.

- [ ] **Step 5: Update the scheduler**

`runAutomatic` must inspect terminal state instead of treating a nil error as success:

```go
run, err := s.coordinator.RunAndWait(ctx, domain.ScrapeSourceAutomatic)
if IsScrapeInProgress(err) {
	slog.Info("scrape periodic tick skipped", "run", runIndex)
	_ = s.coordinator.WaitUntilIdle(ctx)
	return
}
if err != nil {
	slog.Warn("scrape periodic coordination failed", "run", runIndex, "error", err)
	return
}
slog.Info(
	"scrape periodic run finished",
	"run", runIndex,
	"run_id", run.ID,
	"state", run.State,
	"publishers_complete", run.Totals.PublishersComplete,
	"publishers_failed", run.Totals.PublishersFailed,
)
```

The loop calls `DelayUntilNextAutomatic(ctx, interval)` and exits on a returned context error. Add tests proving a partial failure and a total failure both start a full cooldown and do not hot-loop.

- [ ] **Step 6: Run coordinator, store, and scheduler tests with the race detector**

Run:

```bash
cd go_server
go test -race ./internal/scrape -run 'TestMemoryRunStore|TestCoordinator|TestPeriodicScheduler' -count=1
```

Expected:

```text
ok  	comics/internal/scrape
```

No `DATA RACE` output is permitted.

- [ ] **Step 7: Commit**

```bash
git add \
  go_server/internal/scrape/run_store.go \
  go_server/internal/scrape/run_store_test.go \
  go_server/internal/scrape/coordinator.go \
  go_server/internal/scrape/coordinator_test.go \
  go_server/internal/scrape/periodic.go \
  go_server/internal/scrape/periodic_test.go
git commit -m "feat(scrape): track and drain scrape runs"
```

---

### Task 5: Add PostgreSQL Advisory-Lock Coordination and Shared History

**Files:**
- Create: `go_server/internal/scrape/postgres_coordination.go`
- Create: `go_server/internal/scrape/postgres_coordination_test.go`

**Interfaces:**
- Produces: `PostgresCoordinationStore`, implementing both `RunStore` and `RunLocker`.
- Produces: `NewPostgresCoordinationStore(context.Context, string, int)`.
- Consumes: Task 4 store/locker contracts.

- [ ] **Step 1: Write the opt-in PostgreSQL integration tests**

```go
package scrape

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"comics/domain"
)

func testPostgresCoordinationStore(t *testing.T) *PostgresCoordinationStore {
	t.Helper()
	dsn := os.Getenv("COMICS_POSTGRES_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("COMICS_POSTGRES_URL not set")
	}
	store, err := NewPostgresCoordinationStore(context.Background(), dsn, 50)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestPostgresAdvisoryLockIsSingleFlightAcrossPools(t *testing.T) {
	first := testPostgresCoordinationStore(t)
	second := testPostgresCoordinationStore(t)
	ctx := context.Background()

	lease, acquired, err := first.TryAcquire(ctx)
	if err != nil || !acquired {
		t.Fatalf("first acquired=%v err=%v", acquired, err)
	}
	defer lease.Release(ctx)
	other, acquired, err := second.TryAcquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if acquired {
		_ = other.Release(ctx)
		t.Fatal("second pool acquired the same advisory lock")
	}
}

func TestPostgresRunHistoryVisibleAcrossPools(t *testing.T) {
	first := testPostgresCoordinationStore(t)
	second := testPostgresCoordinationStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	run := domain.ScrapeRun{
		ID: "postgres-shared-history", Source: domain.ScrapeSourceManual,
		State: domain.ScrapeRunComplete, AcceptedAt: now, FinishedAt: &now,
		Publishers: []domain.ScrapePublisherOutcome{},
		Failures: []domain.ScrapeFailure{},
	}
	if err := first.Save(ctx, run); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = first.db.ExecContext(ctx, `DELETE FROM scrape_runs WHERE id = $1`, run.ID)
	})
	latest, err := second.Latest(ctx)
	if err != nil || latest == nil || latest.ID != run.ID {
		t.Fatalf("latest=%#v err=%v", latest, err)
	}
}

func TestPostgresCoordinatorsRejectCrossPoolOverlap(t *testing.T) {
	first := testPostgresCoordinationStore(t)
	second := testPostgresCoordinationStore(t)
	ctx := context.Background()
	release := make(chan struct{})
	firstRunner := RunExecutorFunc(func(context.Context) (RunReport, error) {
		<-release
		return RunReport{Publishers: []domain.ScrapePublisherOutcome{{
			Publisher: "Fixture", State: domain.ScrapePublisherComplete,
			URLsTotal: 1, URLsSucceeded: 1,
		}}}, nil
	})
	var secondRan atomic.Bool
	secondRunner := RunExecutorFunc(func(context.Context) (RunReport, error) {
		secondRan.Store(true)
		return RunReport{}, nil
	})
	firstCoord := NewCoordinatorWithOptions(ctx, firstRunner, CoordinatorOptions{
		Store: first, Locker: first, Observer: NopObserver{},
		RunTimeout: time.Minute, NewRunID: func() string { return "pg-first-run" },
	})
	secondCoord := NewCoordinatorWithOptions(ctx, secondRunner, CoordinatorOptions{
		Store: second, Locker: second, Observer: NopObserver{},
		RunTimeout: time.Minute, NewRunID: func() string { return "pg-second-run" },
	})
	accepted, err := firstCoord.Start(ctx, domain.ScrapeSourceManual)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := secondCoord.Start(ctx, domain.ScrapeSourceAutomatic); !IsScrapeInProgress(err) {
		t.Fatalf("expected cross-pool in-progress error, got %v", err)
	}
	if secondRan.Load() {
		t.Fatal("second coordinator executed despite the advisory lock")
	}
	status, err := secondCoord.Status(ctx)
	if err != nil || !status.Running || status.Active == nil || status.Active.ID != accepted.ID {
		t.Fatalf("status=%#v err=%v", status, err)
	}
	close(release)
	if _, err := firstCoord.Wait(ctx, accepted.ID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = first.db.ExecContext(
			ctx,
			`DELETE FROM scrape_runs WHERE id IN ($1, $2)`,
			"pg-first-run",
			"pg-second-run",
		)
	})
}
```

- [ ] **Step 2: Run without PostgreSQL and verify the tests skip cleanly**

Run:

```bash
cd go_server
env -u COMICS_POSTGRES_URL -u DATABASE_URL \
  go test ./internal/scrape -run 'TestPostgres' -count=1 -v
```

Expected output includes:

```text
COMICS_POSTGRES_URL not set
--- SKIP: TestPostgresAdvisoryLockIsSingleFlightAcrossPools
--- SKIP: TestPostgresRunHistoryVisibleAcrossPools
--- SKIP: TestPostgresCoordinatorsRejectCrossPoolOverlap
PASS
```

- [ ] **Step 3: Implement the additive PostgreSQL schema**

`NewPostgresCoordinationStore` opens its own `pgx` pool, pings it, and executes:

```sql
CREATE TABLE IF NOT EXISTS scrape_runs (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    state TEXT NOT NULL CHECK (
        state IN (
            'accepted',
            'running',
            'complete',
            'partial_failure',
            'total_failure',
            'canceled'
        )
    ),
    accepted_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    publishers JSONB NOT NULL DEFAULT '[]'::jsonb,
    totals JSONB NOT NULL DEFAULT '{}'::jsonb,
    failures JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_scrape_runs_terminal_finished
    ON scrape_runs (finished_at DESC)
    WHERE state IN ('complete', 'partial_failure', 'total_failure', 'canceled');

CREATE UNIQUE INDEX IF NOT EXISTS idx_scrape_runs_one_active
    ON scrape_runs ((state IN ('accepted', 'running')))
    WHERE state IN ('accepted', 'running');
```

The partial unique index is a data-integrity guard for persisted status. The advisory lock remains the admission mechanism.

- [ ] **Step 4: Implement the session advisory lock**

Use one stable lock ID:

```go
const scrapeAdvisoryLockID int64 = 0x636f6d6963730005
```

`TryAcquire` must reserve a dedicated `*sql.Conn`:

```go
func (s *PostgresCoordinationStore) TryAcquire(
	ctx context.Context,
) (RunLease, bool, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	var acquired bool
	if err := conn.QueryRowContext(
		ctx,
		`SELECT pg_try_advisory_lock($1)`,
		scrapeAdvisoryLockID,
	).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	if !acquired {
		_ = conn.Close()
		return nil, false, nil
	}
	return &postgresRunLease{conn: conn}, true, nil
}
```

The lease executes `SELECT pg_advisory_unlock($1)` on the same connection and then closes it. Protect release with `sync.Once`. Never use a transaction-level lock; the scrape spans many independent catalog transactions.

- [ ] **Step 5: Implement the shared run store**

Use JSON marshal/unmarshal for publishers, totals, and failures. Required SQL behavior:

- `Save`: execute this parameterized upsert:

```sql
INSERT INTO scrape_runs (
    id,
    source,
    state,
    accepted_at,
    started_at,
    finished_at,
    publishers,
    totals,
    failures
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
    source = EXCLUDED.source,
    state = EXCLUDED.state,
    accepted_at = EXCLUDED.accepted_at,
    started_at = EXCLUDED.started_at,
    finished_at = EXCLUDED.finished_at,
    publishers = EXCLUDED.publishers,
    totals = EXCLUDED.totals,
    failures = EXCLUDED.failures;
```
- `Active`: newest row in `accepted`/`running`.
- `Latest`: newest terminal row by `finished_at DESC, accepted_at DESC`.
- `History`: terminal rows in the same order with a parameterized limit.
- `RecoverActive`: update active rows to `canceled`, set `finished_at`, and append the supplied safe failure to the decoded failure list. Perform read/update in one SQL transaction while the advisory lease is held.
- Terminal `Save`: prune terminal rows beyond `historyLimit` with a subquery using `OFFSET $1`; never delete active rows.
- `Close`: close the dedicated pool after coordinator drain.

When scanning, validate source/state strings against the Task 1 constants. Return a wrapped store error for unknown database values rather than serving invalid JSON.

- [ ] **Step 6: Run PostgreSQL tests when a test database is configured**

Run:

```bash
cd go_server
go test ./internal/scrape -run 'TestPostgres' -count=1 -v
```

Expected with `COMICS_POSTGRES_URL` configured:

```text
--- PASS: TestPostgresAdvisoryLockIsSingleFlightAcrossPools
--- PASS: TestPostgresRunHistoryVisibleAcrossPools
--- PASS: TestPostgresCoordinatorsRejectCrossPoolOverlap
PASS
ok  	comics/internal/scrape
```

Expected without it: the clean skips from Step 2.

- [ ] **Step 7: Commit**

```bash
git add \
  go_server/internal/scrape/postgres_coordination.go \
  go_server/internal/scrape/postgres_coordination_test.go
git commit -m "feat(scrape): coordinate postgres replicas"
```

---

### Task 6: Instrument Runs, Publishers, Retries, and Item Outcomes

**Files:**
- Create: `go_server/internal/scrape/metrics.go`
- Create: `go_server/internal/scrape/metrics_test.go`
- Modify: `go_server/internal/scrape/runner.go`
- Modify: `go_server/internal/scrape/coordinator.go`

**Interfaces:**
- Produces: `NewMetricsObserver(prometheus.Registerer) *MetricsObserver`.
- Produces: `DefaultMetricsObserver() *MetricsObserver`, registered once on `prometheus.DefaultRegisterer`.
- Implements: Task 3 `Observer`.

- [ ] **Step 1: Write failing metric tests**

```go
package scrape

import (
	"testing"
	"time"

	"comics/domain"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsObserverRecordsTerminalRunAndPublisher(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetricsObserver(registry)
	started := time.Now().Add(-time.Second)
	finished := time.Now()
	run := domain.ScrapeRun{
		ID: "metrics-run", Source: domain.ScrapeSourceManual,
		State: domain.ScrapeRunComplete, StartedAt: &started, FinishedAt: &finished,
	}
	outcome := domain.ScrapePublisherOutcome{
		Publisher: "Asura", State: domain.ScrapePublisherComplete,
		Created: 2, Updated: 3,
	}

	metrics.RunStarted(run)
	metrics.PublisherFinished(outcome)
	metrics.FetchRetry("Asura", RetryEvent{Code: "http_status"})
	metrics.RunFinished(run)

	if got := testutil.ToFloat64(metrics.active); got != 0 {
		t.Fatalf("active=%v", got)
	}
	if got := testutil.ToFloat64(
		metrics.runs.WithLabelValues("manual", "complete"),
	); got != 1 {
		t.Fatalf("runs=%v", got)
	}
	if got := testutil.ToFloat64(
		metrics.publisherRuns.WithLabelValues("Asura", "complete"),
	); got != 1 {
		t.Fatalf("publisher runs=%v", got)
	}
	if got := testutil.ToFloat64(
		metrics.fetchRetries.WithLabelValues("Asura", "http_status"),
	); got != 1 {
		t.Fatalf("retries=%v", got)
	}
}
```

- [ ] **Step 2: Run and verify missing observer**

Run:

```bash
cd go_server
go test ./internal/scrape -run TestMetricsObserver -count=1
```

Expected: build failure containing `undefined: NewMetricsObserver`.

- [ ] **Step 3: Register low-cardinality metrics**

Create these collectors:

```text
comics_scrape_active
comics_scrape_runs_total{source,state}
comics_scrape_run_duration_seconds{source,state}
comics_scrape_publisher_runs_total{publisher,state}
comics_scrape_fetch_retries_total{publisher,kind}
comics_scrape_items_total{publisher,disposition}
```

Use fixed publisher/source/state/kind/disposition labels only. Never label by run ID, title, URL, error message, or HTTP path.

Observer behavior:

- `RunStarted`: set active gauge to `1`.
- `RunFinished`: set active to `0`, increment run counter, observe duration when both timestamps exist.
- `PublisherFinished`: increment publisher counter and add nonzero created/updated/unchanged/skipped/failed counts to `items_total`.
- `FetchRetry`: increment retry counter using the stable fetch code.
- `DefaultMetricsObserver`: use `sync.OnceValue` so repeated callers do not double-register collectors.

- [ ] **Step 4: Wire observer callbacks once**

- Coordinator owns `RunStarted` and `RunFinished`.
- Runner owns `PublisherFinished` and `FetchRetry`.
- Do not increment terminal metrics in routes, use cases, or scheduler; retries and alternate triggers would double-count.

- [ ] **Step 5: Run metrics and scrape tests**

Run:

```bash
cd go_server
go test ./internal/scrape -run 'TestMetricsObserver|TestRunner|TestCoordinator' -count=1
```

Expected:

```text
ok  	comics/internal/scrape
```

- [ ] **Step 6: Commit**

```bash
git add \
  go_server/internal/scrape/metrics.go \
  go_server/internal/scrape/metrics_test.go \
  go_server/internal/scrape/runner.go \
  go_server/internal/scrape/coordinator.go
git commit -m "feat(scrape): publish outcome metrics"
```

---

### Task 7: Wire Use Cases, Admin Routes, Scheduler, CLI, and Shutdown

**Files:**
- Modify: `go_server/domain/comic.go`
- Modify: `go_server/internal/usecase/comics.go`
- Modify: `go_server/internal/usecase/comics_test.go`
- Modify: `go_server/api/route/comics.go`
- Modify: `go_server/api/route/comics_test.go`
- Modify: `go_server/api/route/auth_route_test.go`
- Modify: `go_server/api/route/route.go`
- Modify: `go_server/bootstrap/env.go`
- Modify: `go_server/bootstrap/app.go`
- Modify: `go_server/cmd/server/main.go`
- Modify: `go_server/cmd/scrape/main.go`

**Interfaces:**
- Replaces: `ComicUseCase.Scrape(context.Context) error`.
- Produces: `StartScrape`, `ScrapeStatus`, and `ScrapeHistory`.
- Consumes: process-local or PostgreSQL coordinator selected by `COMICS_DB_DRIVER`.

- [ ] **Step 1: Write failing route contract tests**

Replace the scrape stub with:

```go
type scrapeStubComicService struct {
	domain.ComicUseCase
	start            domain.ScrapeRun
	status           domain.ScrapeRunStatus
	runs             []domain.ScrapeRun
	lastHistoryLimit int
	err              error
}

func (s *scrapeStubComicService) StartScrape(context.Context) (domain.ScrapeRun, error) {
	return s.start, s.err
}

func (s *scrapeStubComicService) RunScrape(context.Context) (domain.ScrapeRun, error) {
	return s.start, s.err
}

func (s *scrapeStubComicService) ScrapeStatus(context.Context) (domain.ScrapeRunStatus, error) {
	return s.status, s.err
}

func (s *scrapeStubComicService) ScrapeHistory(
	_ context.Context,
	limit int,
) ([]domain.ScrapeRun, error) {
	s.lastHistoryLimit = limit
	return s.runs, s.err
}
```

Add:

```go
func TestStartScrapeReturnsAcceptedRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	now := time.Date(2026, time.October, 26, 14, 0, 0, 0, time.UTC)
	svc := &scrapeStubComicService{
		ComicUseCase: newTestComicService(t),
		start: domain.ScrapeRun{
			ID: "run-accepted", Source: domain.ScrapeSourceManual,
			State: domain.ScrapeRunAccepted, AcceptedAt: now,
			Publishers: []domain.ScrapePublisherOutcome{},
			Failures: []domain.ScrapeFailure{},
		},
	}
	scrapeRouter(svc, router.Group("/"))

	req := httptest.NewRequest(http.MethodPost, "/scrape", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"state":"accepted"`) {
		t.Fatalf("unexpected body %s", res.Body.String())
	}
}

func TestScrapeHistoryClampsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc := &scrapeStubComicService{ComicUseCase: newTestComicService(t)}
	scrapeRouter(svc, router.Group("/"))
	req := httptest.NewRequest(http.MethodGet, "/scrape/history?limit=500", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK || res.Body.String() != `{"runs":[]}` {
		t.Fatalf("code=%d body=%s", res.Code, res.Body.String())
	}
	if svc.lastHistoryLimit != 50 {
		t.Fatalf("limit=%d want=50", svc.lastHistoryLimit)
	}
}
```

Add a busy stub that returns `&scrape.ErrScrapeInProgress{Active: &run}` and assert `409` with code `scrape_in_progress`. Add a generic coordination error assertion for `503`.
Add `TestLegacyGetScrapeMapsTerminalStates`: table-test `complete` and
`partial_failure` as `200`, `total_failure` as `502`, and `canceled` as `503`;
assert every response includes the terminal `run.state`.

- [ ] **Step 2: Run route tests and verify the old contract fails**

Run:

```bash
cd go_server
go test ./api/route -run 'TestStartScrape|TestScrapeStatus|TestScrapeHistory|TestScrapeRoutes' -count=1
```

Expected: build failure containing `StartScrape undefined` or `undefined: scrapeRouter`.

- [ ] **Step 3: Change the use-case interface and implementation**

In `domain.ComicUseCase`, use:

```go
StartScrape(ctx context.Context) (ScrapeRun, error)
RunScrape(ctx context.Context) (ScrapeRun, error)
ScrapeStatus(ctx context.Context) (ScrapeRunStatus, error)
ScrapeHistory(ctx context.Context, limit int) ([]ScrapeRun, error)
```

Delete the old `Scrape(ctx) error` and the legacy `ScrapeStatus` declaration from `comic.go`.
In `coordinator.go`, delete the temporary `Source` aliases, `RunStatus`,
`NewCoordinator(domain.ComicRepository)`, error-only `Run`, `Snapshot`, and
`ErrScrapeRunFailed`. After this task, production and tests use
`NewCoordinatorWithOptions`, `NewProcessLocalCoordinator`, `Start`,
`RunAndWait`, `Status`, and `History` only.

Implement forwarding in `ComicService`:

```go
func (s *ComicService) StartScrape(ctx context.Context) (domain.ScrapeRun, error) {
	return s.scrapeCoord.Start(ctx, domain.ScrapeSourceManual)
}

func (s *ComicService) RunScrape(ctx context.Context) (domain.ScrapeRun, error) {
	run, err := s.scrapeCoord.Start(ctx, domain.ScrapeSourceManual)
	if err != nil {
		return domain.ScrapeRun{}, err
	}
	return s.scrapeCoord.Wait(ctx, run.ID)
}

func (s *ComicService) ScrapeStatus(ctx context.Context) (domain.ScrapeRunStatus, error) {
	return s.scrapeCoord.Status(ctx)
}

func (s *ComicService) ScrapeHistory(
	ctx context.Context,
	limit int,
) ([]domain.ScrapeRun, error) {
	return s.scrapeCoord.History(ctx, limit)
}
```

Keep test convenience by changing the nil-coordinator branch in `NewComicService` to `NewProcessLocalCoordinator(context.Background(), repo, NopObserver{}, DefaultScrapeTimeout)`. Production always supplies its lifecycle-bound coordinator.

- [ ] **Step 4: Add the scrape-only admin router**

Remove scrape registration from `comicsRouter`. Add:

```go
func scrapeRouter(comics domain.ComicUseCase, group *gin.RouterGroup) {
	group.POST("/scrape", startScrape(comics))
	group.GET("/scrape", runScrapeLegacy(comics))
	group.GET("/scrape/status", scrapeStatus(comics))
	group.GET("/scrape/history", scrapeHistory(comics))
}
```

`startScrape` maps:

- accepted run to `202` with `gin.H{"run": run}`;
- `ErrScrapeInProgress` to `409` with code/message/active run;
- every coordination/store error to `503` with stable code/message and logs the original error.

`runScrapeLegacy` calls `RunScrape`, returns `200` plus
`{"message":"success","run":run}` for complete/partial, `502` plus
`{"code":"scrape_total_failure","message":"scrape failed","run":run}` for total
failure, and `503` plus
`{"code":"scrape_canceled","message":"scrape canceled","run":run}` for canceled.
It uses the same `409` and coordination-unavailable mappings as `startScrape`.
If its waiting request disconnects, `Coordinator.Wait` returns but the
lifecycle-rooted run continues and remains visible through status.

`scrapeStatus` writes `domain.ScrapeRunStatus` directly. `scrapeHistory` parses
`limit`, defaults to 10, rejects a non-integer with `400`, clamps below 1 to 1
and above 50 to 50, and returns `gin.H{"runs": runs}`. Status/history store
errors return `503` with code `scrape_coordination_unavailable`; they never
return a fabricated idle/empty response.

In `route.Setup`, mount the routes on Workstream 1's existing root-path
`catalogAdmin` group:

```go
scrapeRouter(comics, catalogAdmin)
```

Do not mount scrape handlers on `publicRouter` and do not create a second admin
group. At this point `catalogAdmin` already uses Workstream 1's bearer-only
authentication plus `RoleMiddleware(tokenutil.RoleAdmin)`. Workstream 6 later
replaces that group's authenticator with the Mongo-backed
`domain.AccessAuthenticator` without changing its route policy.

Add `TestScrapeRoutesRequireAdmin`, one table-driven route-policy test over
`POST /scrape`, `GET /scrape`, `GET /scrape/status`, and
`GET /scrape/history`:

```go
for _, path := range []struct {
	method string
	path   string
}{
	{method: http.MethodPost, path: "/scrape"},
	{method: http.MethodGet, path: "/scrape"},
	{method: http.MethodGet, path: "/scrape/status"},
	{method: http.MethodGet, path: "/scrape/history"},
} {
	t.Run(path.method+" "+path.path, func(t *testing.T) {
		assertStatus := func(token string, want int) {
			req := httptest.NewRequest(path.method, path.path, nil)
			req.Header.Set(keyAccept, contentTypeJSON)
			if token != "" {
				req.Header.Set(keyAuthorization, "Bearer "+token)
			}
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != want {
				t.Fatalf("status=%d body=%s want=%d", res.Code, res.Body.String(), want)
			}
		}
		userToken, _ := tokenutil.GenerateTokenWithRole(
			uuid.New(), []byte("access-secret"), time.Hour, tokenutil.RoleUser,
		)
		adminToken, _ := tokenutil.GenerateTokenWithRole(
			uuid.New(), []byte("access-secret"), time.Hour, tokenutil.RoleAdmin,
		)
		assertStatus("", http.StatusUnauthorized)
		assertStatus(userToken, http.StatusForbidden)
		assertStatus(adminToken, expectedAdminStatus(path.method, path.path))
	})
}
```

Define `expectedAdminStatus` in the test as `202` for `POST /scrape`, `200` for
the synchronous compatibility `GET /scrape` and both read routes. Give the stub
a terminal `complete` run so the compatibility request cannot block.

- [ ] **Step 5: Select coordination by catalog driver**

Add to `Env`:

```go
ScrapeTimeout time.Duration `mapstructure:"SCRAPE_TIMEOUT"`
```

Set default `30*time.Minute` and bind `SCRAPE_TIMEOUT`. Reject a value `<= 0` in `Validate` with `SCRAPE_TIMEOUT should be greater than zero`.

Change `newComicStore` to accept the lifecycle context and observer, and return the same use case/coordinator pair. Exact selection:

- SQLite/empty driver: `MemoryRunStore(50)` plus `LocalRunLocker`.
- PostgreSQL driver: `NewPostgresCoordinationStore(ctx, env.ComicsPostgresURL, 50)` passed as both store and locker.
- Both modes use one `Runner` with `DefaultMetricsObserver`.

Construct:

```go
runner := scrape.NewRunner(repo, nil, scrape.RunnerOptions{
	Retry: scrape.DefaultRetryPolicy(), Observer: observer,
})
coord := scrape.NewCoordinatorWithOptions(ctx, runner, scrape.CoordinatorOptions{
	Store: store, Locker: locker, Observer: observer,
	RunTimeout: env.ScrapeTimeout,
})
```

Do not place `coord.Shutdown` in `Application.Shutters`: Workstream 4's
lifecycle manager owns server drain, and the one-shot CLI calls it explicitly.
Keep repository/store close functions in `Application.Shutters`; the
coordinator owns and closes its run store during its idempotent `Shutdown`, so
do not add a second PostgreSQL coordination-store close.

`MustLoadApp` currently shadows its incoming context with the initialization
timeout and cancels that context before returning. Preserve the process context:

```go
func MustLoadApp(lifecycleCtx context.Context) Application {
	env := MustLoadEnv(lifecycleCtx)
	initCtx, cancel := context.WithTimeout(lifecycleCtx, env.InitCtxTimeout)
	defer cancel()

	// Use initCtx for repository construction and startup pings.
	// Pass lifecycleCtx, not initCtx, to newComicStore/coordinator construction.
}
```

Do not root the coordinator in `initCtx`; doing so would cancel every scrape as
soon as bootstrap returns.

- [ ] **Step 6: Update periodic and CLI entry points**

The server scheduler continues to use the application lifecycle context and shared coordinator.

Refactor `cmd/scrape/main.go` into `os.Exit(run())`; `run` preserves Workstream
4's `signal.NotifyContext` and SQLite single-instance bootstrap, calls
`ScrapeCoord.RunAndWait(ctx, domain.ScrapeSourceCLI)`, logs run
ID/state/totals, then calls `ScrapeCoord.Shutdown` before bounded `app.Close`.
It returns:

- `0` for `complete`;
- `3` for `partial_failure`;
- `1` for `total_failure` or coordination/store error;
- `130` for `canceled`;
- `2` for `ErrScrapeInProgress`.

Use `errors.Is(err, scrape.ErrCoordinatorClosed)` for the closed coordinator
case. Workstream 6 later rewrites admin-route tests to seed a Mongo-backed
admin and use its authenticator fake; do not retain
`GenerateTokenWithRole` after role claims are removed.

This command has its own process-local locker in SQLite mode. Add an explicit log warning in SQLite mode: `SQLite scrape coordination does not span processes; stop the API before using go-scrape`.

- [ ] **Step 7: Run use-case, route, bootstrap, and command tests**

Run:

```bash
cd go_server
go test ./internal/usecase ./api/route ./bootstrap ./cmd/scrape -count=1
```

Expected:

```text
ok  	comics/internal/usecase
ok  	comics/api/route
ok  	comics/bootstrap
ok  	comics/cmd/scrape
```

Create `go_server/cmd/scrape/main_test.go` around a pure
`exitCode(domain.ScrapeRunState, error) int` helper and assert all five values.

- [ ] **Step 8: Commit**

```bash
git add \
  go_server/domain/comic.go \
  go_server/internal/usecase/comics.go \
  go_server/internal/usecase/comics_test.go \
  go_server/api/route/comics.go \
  go_server/api/route/comics_test.go \
  go_server/api/route/auth_route_test.go \
  go_server/api/route/route.go \
  go_server/bootstrap/env.go \
  go_server/bootstrap/app.go \
  go_server/cmd/server/main.go \
  go_server/cmd/scrape/main.go \
  go_server/cmd/scrape/main_test.go
git commit -m "feat(scrape): expose run status and history"
```

---

### Task 8: Make the Scrape Button Follow the Real Run Outcome

**Files:**
- Modify: `src/frontend/components/Comics/Edition/ScrapeButton.tsx`
- Rename: `src/frontend/components/Comics/Edition/ScrapeButton.test.ts` to `src/frontend/components/Comics/Edition/ScrapeButton.test.tsx`

**Interfaces:**
- Consumes: Task 7 HTTP contract.
- Produces: typed `startScrape` and `fetchScrapeStatus` helpers.
- Preserves: `<ScrapeButton onSuccess?: () => void>`.

- [ ] **Step 1: Rename the helper test and add typed API/component behavior tests**

Rename the test to `ScrapeButton.test.tsx`, then add these cases:

```tsx
import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import ScrapeButton, { startScrape } from './ScrapeButton';
import { ToastProvider } from '../../Toast/ToastProvider';

const acceptedRun = {
  id: 'run-1',
  source: 'manual' as const,
  state: 'accepted' as const,
  accepted_at: '2026-10-26T14:00:00Z',
  publishers: [],
  totals: {
    publishers_total: 0,
    publishers_complete: 0,
    publishers_partial: 0,
    publishers_failed: 0,
    publishers_canceled: 0,
    urls_total: 0,
    urls_succeeded: 0,
    urls_failed: 0,
    items_seen: 0,
    created: 0,
    updated: 0,
    unchanged: 0,
    skipped: 0,
    failed: 0,
    fetch_retries: 0,
  },
  failures: [],
};

beforeEach(() => {
  vi.useFakeTimers();
  sessionStorage.setItem('comics_access_token', 'admin-token');
});

afterEach(() => {
  vi.clearAllTimers();
  vi.useRealTimers();
  vi.unstubAllGlobals();
  sessionStorage.clear();
});

test('starts an authenticated scrape and returns the accepted run', async () => {
  const fetchMock = vi.fn().mockResolvedValue({
    ok: true,
    status: 202,
    json: () => Promise.resolve({ run: acceptedRun }),
  });
  vi.stubGlobal('fetch', fetchMock);

  await expect(startScrape('http://localhost:8081')).resolves.toEqual({
    kind: 'accepted',
    run: acceptedRun,
  });
  expect(fetchMock).toHaveBeenCalledWith(
    'http://localhost:8081/scrape',
    expect.objectContaining({
      method: 'POST',
      headers: expect.any(Headers),
    }),
  );
  const headers = fetchMock.mock.calls[0][1].headers as Headers;
  expect(headers.get('Authorization')).toBe('Bearer admin-token');
  expect(headers.get('X-CSRF-Protection')).toBe('1');
});

test('refreshes and warns when a run finishes partially', async () => {
  const onSuccess = vi.fn();
  const partial = {
    ...acceptedRun,
    state: 'partial_failure' as const,
    finished_at: '2026-10-26T14:01:00Z',
    totals: {
      ...acceptedRun.totals,
      publishers_total: 5,
      publishers_complete: 4,
      publishers_failed: 1,
    },
  };
  vi.stubGlobal('fetch', vi.fn()
    .mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ running: false, active: null, latest: null }),
    })
    .mockResolvedValueOnce({
      ok: true,
      status: 202,
      json: () => Promise.resolve({ run: acceptedRun }),
    })
    .mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ running: false, active: null, latest: partial }),
    }));

  render(
    <ToastProvider>
      <ScrapeButton onSuccess={onSuccess} />
    </ToastProvider>,
  );
  await act(async () => Promise.resolve());
  fireEvent.click(screen.getByRole('button', { name: 'Scrape sources' }));
  await act(async () => Promise.resolve());
  await act(async () => {
    vi.advanceTimersByTime(2500);
    await Promise.resolve();
  });

  expect(screen.getByText('Catalog partially refreshed')).toBeTruthy();
  expect(screen.getByText('4 of 5 publishers completed; 1 failed.')).toBeTruthy();
  expect(onSuccess).toHaveBeenCalledTimes(1);
});
```

Also add:

- `409` returns `{kind:"busy", run}` and enters polling without an error toast.
- `401`/`403` returns `{kind:"unauthorized"}` and displays `Admin access required`.
- `total_failure` displays `Scrape failed`, includes publisher counts, and does not call `onSuccess`.
- `canceled` displays `Scrape canceled` and does not call `onSuccess`.
- Three consecutive status request failures stop the local busy indicator and show `Scrape status unavailable`; one or two failures keep polling and never claim success.
- Unmount clears the interval.

- [ ] **Step 2: Run the focused frontend test and verify the old helper contract fails**

Run:

```bash
npm test -- src/frontend/components/Comics/Edition/ScrapeButton.test.tsx
```

Expected: failure containing `startScrape is not a function`.

- [ ] **Step 3: Add exact TypeScript response types and helpers**

Define:

```ts
export type ScrapeRunState =
  | 'accepted'
  | 'running'
  | 'complete'
  | 'partial_failure'
  | 'total_failure'
  | 'canceled';

export type ScrapePublisherState =
  | 'complete'
  | 'partial_failure'
  | 'failed'
  | 'canceled';

export type ScrapeTotals = {
  publishers_total: number;
  publishers_complete: number;
  publishers_partial: number;
  publishers_failed: number;
  publishers_canceled: number;
  urls_total: number;
  urls_succeeded: number;
  urls_failed: number;
  items_seen: number;
  created: number;
  updated: number;
  unchanged: number;
  skipped: number;
  failed: number;
  fetch_retries: number;
};

export type ScrapeRun = {
  id: string;
  source: 'manual' | 'automatic' | 'cli';
  state: ScrapeRunState;
  accepted_at: string;
  started_at?: string;
  finished_at?: string;
  publishers: Array<{
    publisher: string;
    state: ScrapePublisherState;
    urls_total: number;
    urls_succeeded: number;
    urls_failed: number;
    items_seen: number;
    created: number;
    updated: number;
    unchanged: number;
    skipped: number;
    failed: number;
    fetch_retries: number;
    duration_ms: number;
    failures: Array<{
      kind: string;
      code: string;
      message: string;
      retryable: boolean;
    }>;
  }>;
  totals: ScrapeTotals;
  failures: Array<{
    kind: string;
    code: string;
    message: string;
    retryable: boolean;
  }>;
};

export type ScrapeStatusResponse = {
  running: boolean;
  active: ScrapeRun | null;
  latest: ScrapeRun | null;
};

export type StartScrapeResult =
  | { kind: 'accepted'; run: ScrapeRun }
  | { kind: 'busy'; run: ScrapeRun | null }
  | { kind: 'unauthorized' }
  | { kind: 'error'; message: string };
```

Use `readAccessToken` and one `scrapeHeaders(method)` helper returning a
`Headers` object with `Accept: application/json`. When a token exists, set
`Authorization` to the literal prefix `Bearer ` followed by the token returned
by login. For every non-GET method, also set
`X-CSRF-Protection: 1`; this is harmless before Workstream 6 and required by
its browser mutation guard after it lands.

`startScrape(server = SERVER)` sends `POST`; map `202`, `409`, `401/403`, malformed payload, network error, and other status to the discriminated union. `fetchScrapeStatus(server = SERVER)` sends authenticated `GET` and throws on non-`200` or malformed `{running,active,latest}` instead of converting failures to idle.

- [ ] **Step 4: Implement polling and outcome-specific UI**

Required behavior:

1. Initial status load adopts an active remote run but does not toast for an old terminal run.
2. Click sends `POST` directly; remove the preflight status request because it is racy and the coordinator already returns `409`.
3. Record the accepted/busy run ID in a ref and poll every `2500ms`. When a
   `409` body has `run: null`, adopt `status.active.id` on the first poll that
   exposes the remote run.
4. While running, title is `Scraping`, description is `Refreshing catalog data...`, and the button is disabled.
5. When polling reports no active run, require `latest.id` to match the tracked run ID before announcing a result.
6. `complete`: success toast `Catalog refreshed`, description `All N publishers completed.`, call `onSuccess`.
7. `partial_failure`: info toast `Catalog partially refreshed`, description `X of N publishers completed; Y failed.`, call `onSuccess`.
8. `total_failure`: error toast `Scrape failed`, description `No publisher completed successfully.`, do not refresh.
9. `canceled`: info toast `Scrape canceled`, description `The scrape stopped before all publishers finished.`, do not refresh.
10. One or two polling failures keep the button busy. The third clears local busy state and shows the status-unavailable error; a later click can safely receive `409` if the backend is still running.
11. `401/403`: error toast `Admin access required`, description `Sign in with an administrator account to run a scrape.`
12. Clear interval on terminal state and unmount.

- [ ] **Step 5: Run focused frontend tests**

Run:

```bash
npm test -- src/frontend/components/Comics/Edition/ScrapeButton.test.tsx
```

Expected:

```text
Test Files  1 passed (1)
Tests       8 passed (8)
```

The final count must equal the eight named cases; do not weaken assertions to
match a different count.

- [ ] **Step 6: Build the frontend**

Run:

```bash
npm run build
```

Expected output includes:

```text
✓ built
```

TypeScript must report no scrape response narrowing errors.

- [ ] **Step 7: Commit**

```bash
git add \
  src/frontend/components/Comics/Edition/ScrapeButton.tsx \
  src/frontend/components/Comics/Edition/ScrapeButton.test.ts \
  src/frontend/components/Comics/Edition/ScrapeButton.test.tsx
git commit -m "feat(frontend): show truthful scrape outcomes"
```

---

### Task 9: Document Operations, Verify the Release Path, and Rehearse Rollback

**Files:**
- Modify: `README.md`
- Modify: `docs/deploy-render.md`
- Modify: `docs/superpowers/specs/2026-10-08-scrape-coordination.md`

**Interfaces:**
- Documents: exact runtime states, routes, metrics, retry policy, deployment constraints, and rollback.
- Verifies: Go, frontend, race, contract, build, and optional PostgreSQL paths.

- [ ] **Step 1: Update operational documentation with exact statements**

Add these facts, without retaining contradictory success-only language:

```text
SQLite scrape coordination is process-local. Run one API replica and one
periodic scheduler. Stop the API before `make go-scrape`; a separate CLI
process does not share the API coordinator.

PostgreSQL mode stores scrape status/history in `scrape_runs` and holds the
session advisory lock `0x636f6d6963730005` for the full run. This permits
multiple API replicas without Redis, while still allowing only one scrape.

SQLite status/history retains the newest 50 terminal runs in process memory and
resets when the single API process restarts.

Fetch retries use at most four attempts with 500ms/1s/2s exponential delays,
a 2s cap, and ±20% jitter. Only transient fetch failures are retried.

`POST /scrape` accepts a run, `GET /scrape/status` returns active/latest state,
and `GET /scrape/history?limit=10` returns terminal history. All are admin-only.
The authenticated `GET /scrape` trigger remains a deprecated synchronous
compatibility endpoint.
```

List all six run states and the six Prometheus metric names from Task 6. Document `SCRAPE_TIMEOUT=30m` and `SCRAPE_INTERVAL=10m`.

- [ ] **Step 2: Run formatting**

Run:

```bash
cd go_server
gofmt -w \
  domain/scrape.go \
  domain/scrape_test.go \
  domain/comic.go \
  internal/scrape/*.go \
  internal/usecase/comics.go \
  internal/usecase/comics_test.go \
  api/route/comics.go \
  api/route/comics_test.go \
  api/route/route.go \
  bootstrap/env.go \
  bootstrap/app.go \
  cmd/server/main.go \
  cmd/scrape/*.go
```

Expected: command exits `0` with no output.

- [ ] **Step 3: Run the complete Go suite and race-sensitive scrape suite**

Run:

```bash
make go-test
cd go_server
go test -race ./internal/scrape ./api/route -count=1
```

Expected:

```text
ok  	comics/internal/scrape
ok  	comics/api/route
```

All remaining Go packages must also be `ok` or explicitly show `[no test files]`. No race report is permitted.

- [ ] **Step 4: Run frontend tests/build**

Run:

```bash
make check-front
```

Expected:

```text
All frontend test files passed.
All frontend tests passed.
✓ built
```

No React `act(...)` warning, unhandled promise rejection, or TypeScript error is permitted.

- [ ] **Step 5: Run Go release checks**

Run:

```bash
make check-go
```

Expected: Go tests, REST contract tests, vet, builds, and lint all exit `0`.

- [ ] **Step 6: Exercise the optional PostgreSQL path**

With a disposable PostgreSQL URL:

```bash
cd go_server
COMICS_POSTGRES_URL="$COMICS_POSTGRES_URL" \
  go test ./internal/scrape -run 'TestPostgres' -count=1 -v
```

Expected: both PostgreSQL tests pass, not skip. Run two test coordinators and verify the second receives `ErrScrapeInProgress` while the first holds its lease; this assertion belongs in the integration test, not a manual production scrape.

- [ ] **Step 7: Verify metrics names**

Extend `metrics_test.go` to gather its private registry and compare the sorted
metric-family names to this exact list:

```text
comics_scrape_active
comics_scrape_fetch_retries_total
comics_scrape_items_total
comics_scrape_publisher_runs_total
comics_scrape_run_duration_seconds
comics_scrape_runs_total
```

Run:

```bash
cd go_server
go test ./internal/scrape -run TestMetricsObserver -count=1 -v
```

Expected:

```text
--- PASS: TestMetricsObserverRecordsTerminalRunAndPublisher
--- PASS: TestMetricsObserverRegistersExpectedFamilies
PASS
ok  	comics/internal/scrape
```

Do not use a live publisher scrape as release verification.

- [ ] **Step 8: Commit documentation**

```bash
git add \
  README.md \
  docs/deploy-render.md \
  docs/superpowers/specs/2026-10-08-scrape-coordination.md
git commit -m "docs(scrape): document reliability operations"
```

- [ ] **Step 9: Rehearse rollback in a detached disposable worktree**

The implementation commit subjects are fixed by this plan. From the repository
containing all workstream commits, run:

```bash
set -euo pipefail
repo_root="$(git rev-parse --show-toplevel)"
rollback_parent="$(mktemp -d)"
rollback_worktree="$rollback_parent/worktree"
git worktree add --detach "$rollback_worktree" HEAD
cleanup_rollback_worktree() {
  cd "$repo_root"
  git worktree remove --force "$rollback_worktree" 2>/dev/null || true
  rmdir "$rollback_parent" 2>/dev/null || true
}
trap cleanup_rollback_worktree EXIT
cd "$rollback_worktree"
subjects=(
  "docs(scrape): document reliability operations"
  "feat(frontend): show truthful scrape outcomes"
  "feat(scrape): expose run status and history"
  "feat(scrape): publish outcome metrics"
  "feat(scrape): coordinate postgres replicas"
  "feat(scrape): track and drain scrape runs"
  "feat(scrape): report publisher outcomes"
  "feat(scrape): retry transient fetch failures"
  "feat(scrape): define reliable run outcomes"
)
for subject in "${subjects[@]}"; do
  sha="$(git log --format='%H' --grep="^${subject}$" -n 1)"
  test -n "$sha" || { echo "missing commit: $subject"; exit 1; }
  git revert --no-edit "$sha" || exit 1
done
make check
```

Expected: all reverts apply and `make check` returns the pre-workstream baseline. The additive PostgreSQL `scrape_runs` table may remain after code rollback; it is inert and preserves history. If policy requires removal after rollback, run only against the scrape coordination database:

```sql
DROP TABLE IF EXISTS scrape_runs;
```

No SQLite schema rollback is required. Session advisory locks release
automatically when their dedicated connections close. For immediate mitigation
without a code rollback, set `SCRAPE_INTERVAL=0`, restart the API, and leave
manual scraping disabled at the authorization layer.

## Acceptance Checklist

- [ ] Every configured publisher has one ordered outcome containing URL, item, mutation, retry, duration, and safe failure counts.
- [ ] A successful publisher plus any failed publisher/configuration produces `partial_failure`.
- [ ] No successful publisher work produces `total_failure`.
- [ ] Context cancellation or deadline produces `canceled` even after partial commits.
- [ ] `complete` is impossible when any URL, registration, or publisher configuration failed.
- [ ] Only transient fetch errors retry; parse/config/registration failures execute once.
- [ ] Retry attempts, backoff cap, jitter, and cancellation are deterministic under test hooks.
- [ ] Status is sourced from run records, not a stale boolean.
- [ ] History is newest-first, terminal-only, and bounded to 50.
- [ ] Shutdown rejects new runs, cancels the active local run, records `canceled`, and drains before the catalog repository closes.
- [ ] SQLite remains process-local and explicitly single-replica.
- [ ] PostgreSQL holds one session advisory lock across the full run and shares status/history across pools.
- [ ] No Redis, queue, or worker service was added.
- [ ] Manual trigger/status/history are admin-only and send bearer auth from the frontend.
- [ ] Frontend toasts and list refresh match the terminal state.
- [ ] Prometheus labels are bounded and never contain run IDs, URLs, titles, or error text.
- [ ] Focused tests, race tests, `make check-go`, and `make check-front` pass.

## Plan Self-Review

- **Spec coverage:** Structured publisher outcomes are Tasks 1 and 3; partial/total/canceled classification is Tasks 1, 3, and 4; transient-only capped retry is Task 2; status/history and cancellation/drain are Task 4; PostgreSQL advisory locking without Redis is Task 5; metrics are Task 6; routes/lifecycle are Task 7; frontend behavior is Task 8; verification and rollback are Task 9.
- **Placeholder scan:** The plan contains no prohibited placeholder marker,
  deferred implementation instruction, unnamed test, unspecified state string,
  unspecified route, or unspecified commit subject. Remaining ellipses are
  executable TypeScript spread syntax or literal UI punctuation.
- **Type consistency:** `domain.ScrapeRun` is the single route/store/frontend contract; `RunReport` is runner-only; `RunStore` persists `domain.ScrapeRun`; `Coordinator` is the only state-transition owner; `Observer` receives domain outcomes; use-case and route signatures use the same domain types.
- **Boundary check:** SQLite does not claim cross-process locking. PostgreSQL uses `pg_try_advisory_lock` plus shared SQL history. No source change in this plan requires Redis or a separate worker.
- **Rollback check:** Every implementation task has one exact commit subject, reverse-order revert commands, and database cleanup limited to the additive PostgreSQL table.
