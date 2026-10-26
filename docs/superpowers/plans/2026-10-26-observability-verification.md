# Workstream 9: Observability and Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the architecture program with vendor-neutral operational
telemetry and mandatory verification gates for authorization, discovery
integrity, scrape reliability, readiness, shutdown, repository parity,
migrations, route policy, and frontend mutation behavior.

**Architecture:** Keep Prometheus and OpenTelemetry as the portable telemetry
contracts. Reuse the discovery and scrape observers introduced by Workstreams
2 and 5, register them in one application-owned registry, and add only the
missing authorization, keyless, readiness, shutdown, and last-completed
signals. Correlate metadata-only JSON logs with request, run, trace, and span
IDs. Extend the SQLite/PostgreSQL contract and migration harnesses introduced
by Workstream 3 instead of creating parallel test frameworks.

**Tech Stack:** Go 1.27.3, Gin, Zerolog, Prometheus client,
OpenTelemetry/OTLP, SQLite, PostgreSQL 16 testcontainers, React 19,
TypeScript 6, Vitest 3, Playwright 1.64.1, Docker Compose, Prometheus,
Grafana, Jaeger, Loki, Vector, and GitHub Actions.

---

## Global Constraints

- Keep the Go modular monolith, React REST client, SQLite catalog default,
  MongoDB auth store, and optional PostgreSQL catalog.
- SQLite mode remains one API replica and one periodic scheduler.
- Do not add Redis, a required telemetry SaaS, or a proprietary runtime agent.
- Keep `/health` as liveness and `/ready` as synchronous MongoDB plus catalog
  readiness.
- Keep public catalog reads public. Catalog mutation, scrape trigger, scrape
  status, and scrape history remain bearer-admin operations.
- Preserve the metric names and labels already committed by Workstreams 2 and
  5. Do not create duplicate discovery or scrape series.
- Prometheus labels are bounded allowlists. Request IDs, run IDs, trace IDs,
  span IDs, user IDs, comic IDs, titles, URLs, error text, and tokens are never
  metric labels.
- Logs and spans never contain request/response bodies, query strings,
  credentials, cookies, OAuth material, database URLs, publisher URLs, comic
  titles, user identifiers, or raw errors.
- Logs may contain bounded event names, route templates, HTTP method/status,
  duration, request ID, run ID, trace ID, span ID, publisher, state,
  disposition, stable failure kind, and stable failure code.
- PostgreSQL contract and migration tests are mandatory in CI. They use the
  existing testcontainers pattern and do not require a developer-owned
  database.
- The optional Grafana, Jaeger, Loki, Vector, Prometheus, and Alertmanager
  stack must not be an application startup dependency.
- This workstream adds no application gRPC contract. OTLP over gRPC remains a
  supported telemetry transport after Workstream 8 retires application gRPC.

---

## Verified Baseline

These commands were run from the repository root on 2026-10-26:

```bash
make go-test
make contract-test
make front-test
make check
docker compose -f go_server/.observability/docker-compose.yaml config -q
```

Observed result:

```text
make go-test: PASS
make contract-test: PASS; only HTTP route Contract tests execute
make front-test: PASS; 8 files and 24 tests
make check: PASS
observability Compose config: PASS
```

The green baseline does not prove PostgreSQL repository parity, representative
legacy migration, top-level route policy, frontend mutation settlement, safe
correlation, Prometheus rule validity, or release rollback signals.

---

## Dependency Gate and Exact Upstream Seams

Execute this plan after Workstreams 1 through 8. Before Task 1, verify these
exact seams. Stop at the first missing seam and finish or reconcile its owning
workstream; do not introduce a competing abstraction.

- Workstreams 1 and 6:
  root-path public/admin route groups preserve existing REST paths; route
  policy tests prove anonymous public reads, bearer-authenticated active users,
  and Mongo-backed admin authorization without depending on a particular
  middleware symbol name.
- Workstream 2:
  `domain.ErrDiscoveryKeyConflict`,
  `*domain.DiscoveryKeyConflictError{ComicID, OwnerComicID, Scope, TitleKey}`,
  `discovery.NewPrometheusObserver`, and these series:
  `comics_discovery_resolution_total{outcome,match_kind}` and
  `comics_discovery_key_conflicts_total{operation}`.
- Workstream 3:
  `migrations.Open`, `Runner.Up`, `Runner.Down`, `Runner.State`,
  `migrations.DialectSQLite`, `migrations.DialectPostgres`,
  `migrations.CurrentVersion`, `runRepositoryContract`, and the
  integration-tagged PostgreSQL testcontainer harness.
- Workstream 4:
  `health.NewChecker`, `Checker.SetDraining`, `Checker.Ready`,
  `lifecycle.Manager.Shutdown`, `Coordinator.Shutdown`,
  `PeriodicScheduler.Stop`, and `Application.Close(context.Context) error`.
- Workstream 5:
  `scrape.Observer`,
  `scrape.NewMetricsObserver(prometheus.Registerer)`,
  `domain.ScrapeRun`, `domain.ScrapePublisherOutcome`, and these series:
  `comics_scrape_active`,
  `comics_scrape_runs_total{source,state}`,
  `comics_scrape_run_duration_seconds{source,state}`,
  `comics_scrape_publisher_runs_total{publisher,state}`,
  `comics_scrape_fetch_retries_total{publisher,kind}`, and
  `comics_scrape_items_total{publisher,disposition}`.
- Workstream 6:
  `domain.AccessAuthenticator`,
  `BearerAuthenticationMiddleware(auth domain.AccessAuthenticator)`,
  active-user reload, trusted database role, and token claims without role.
- Workstream 7:
  `RestClient.request<T>(ApiRequest<T>): Promise<ApiResult<T>>`,
  `apiClient.setUnauthorizedHandler`,
  `ApiError`, `comicsApi`, `authApi`, and
  `src/frontend/hooks/useComicActions.ts`.
- Workstream 8:
  application gRPC/protobuf and divergent repositories are retired, while
  OTLP dependencies and `go_server/internal/repo/migrations/` remain.

Workstream 8's cleanup must delete only the obsolete
`go_server/internal/repo/sql/` tree and sqlc tooling. It must preserve
Workstream 3's canonical `go_server/cmd/migrate`,
`go_server/internal/repo/migrations/`, migration Make targets, and
`github.com/golang-migrate/migrate/v4`; reconcile Workstreams 3 and 8 before
continuing if any of those are absent.

Run the gate:

```bash
test -d go_server/internal/repo/migrations
test -f go_server/internal/repo/comics/repository_contract_test.go
test -f go_server/internal/repo/comics/repository_contract_postgres_test.go
test -f src/frontend/api/client.ts
test -f go_server/internal/scrape/metrics.go
test -f go_server/api/route/catalog_security_test.go
rg -n 'ErrDiscoveryKeyConflict' go_server/domain/comic.go
rg -n 'func Open\(db \*sql.DB, dialect Dialect\)' \
  go_server/internal/repo/migrations/migrations.go
rg -n 'type Observer interface' go_server/internal/scrape/types.go
```

Expected: every command exits `0`.

---

## Final Metric Contract

Preserve these existing Workstream 2 series unchanged:

```text
comics_discovery_resolution_total{outcome,match_kind}
comics_discovery_key_conflicts_total{operation}
```

Preserve these existing Workstream 5 series unchanged:

```text
comics_scrape_active
comics_scrape_runs_total{source,state}
comics_scrape_run_duration_seconds{source,state}
comics_scrape_publisher_runs_total{publisher,state}
comics_scrape_fetch_retries_total{publisher,kind}
comics_scrape_items_total{publisher,disposition}
```

Add:

```text
comics_authorization_denials_total{route,reason}
comics_discovery_keyless_active
comics_discovery_keyless_refresh_total{outcome}
comics_scrape_last_completed_timestamp_seconds{source,state}
comics_readiness_checks_total{dependency,outcome}
comics_readiness_duration_seconds{dependency,outcome}
comics_readiness_status{dependency}
comics_shutdown_phases_total{phase,outcome}
comics_shutdown_phase_duration_seconds{phase,outcome}
comics_shutdown_total{outcome}
comics_shutdown_duration_seconds{outcome}
```

Allowed values:

- authorization `route`: `/comics`, `/comics/:id`,
  `/comics/:id/cover-visibility`, `/comics/:id/:merging_id`, `/scrape`,
  `/scrape/status`, `/scrape/history`, `/protected/profile`, `/metrics`, or
  `unknown`;
- authorization `reason`: `missing_token`, `malformed_token`,
  `invalid_token`, `inactive_user`, `missing_role`, `forbidden_role`,
  `metrics_disabled`, `invalid_metrics_token`, or `unknown`;
- discovery `operation`: preserve Workstream 2's `create`, `update`, `merge`,
  `scrape`, `import`, or `unknown`;
- scrape `source`, `state`, `publisher`, `kind`, and `disposition`: preserve
  Workstream 5's domain constants and normalizers;
- readiness `dependency`: `users`, `catalog`, or `aggregate`;
- readiness `outcome`: `ready` or `not_ready`;
- shutdown `phase`: `readiness`, `http`, `scheduler`, `scrape`,
  `application`, `catalog`, `users`, `tracing`, `logging`,
  `instance_lock`, or `unknown`;
- shutdown `outcome`: `success`, `error`, or `timeout`.

`request_id`, `run_id`, `trace_id`, and `span_id` are log/trace fields only.

---

### Task 1: Application-Owned Metrics Registry

**Files:**

- Create: `go_server/internal/metrics/application.go`
- Create: `go_server/internal/metrics/application_test.go`
- Modify: `go_server/internal/discovery/metrics.go`
- Modify: `go_server/internal/discovery/resolver_test.go`
- Modify: `go_server/internal/scrape/metrics.go`
- Modify: `go_server/internal/scrape/metrics_test.go`

**Interfaces:**

```go
func NewApplication(registerer prometheus.Registerer) (*Application, error)

func (m *Application) ObserveAuthorizationDenial(route, reason string)
func (m *Application) SetDiscoveryKeylessActive(count int)
func (m *Application) ObserveDiscoveryKeylessRefresh(outcome string)
func (m *Application) ObserveReadiness(
	dependency,
	outcome string,
	elapsed time.Duration,
)
func (m *Application) SetReadinessStatus(dependency string, ready bool)
func (m *Application) ObserveShutdownPhase(
	phase string,
	elapsed time.Duration,
	err error,
)
func (m *Application) ObserveShutdown(elapsed time.Duration, err error)
```

`Application` owns only the new series. It does not redefine discovery or
scrape collectors.

- [ ] **Step 1: Write isolated-registry tests**

Create `application_test.go` with a fresh `prometheus.NewRegistry()`. Record
one event for every method, pass attacker-controlled labels, gather the
registry, and assert:

```go
metrics.ObserveAuthorizationDenial("/comics/123?token=secret", "token-secret")

if got := testutil.ToFloat64(
	metrics.authorizationDenials.WithLabelValues("unknown", "unknown"),
); got != 1 {
	t.Fatalf("unknown authorization denial = %v, want 1", got)
}
```

Run `testutil.GatherAndLint(registry)` and fail on its error or any returned
problem. Inspect every metric descriptor and fail if a variable label contains
`request_id`,
`run_id`, `trace_id`, `span_id`, `user_id`, `comic_id`, `title`, `url`,
`token`, or `error`.

Add a duplicate-registration test:

```go
registry := prometheus.NewRegistry()
if _, err := NewApplication(registry); err != nil {
	t.Fatal(err)
}
if _, err := NewApplication(registry); err == nil {
	t.Fatal("expected duplicate registration error")
}
```

- [ ] **Step 2: Run the red test**

```bash
cd go_server
go test ./internal/metrics -run TestApplication -count=1
```

Expected:

```text
undefined: NewApplication
```

The command exits `1` because the new constructor does not exist.

- [ ] **Step 3: Implement bounded collectors**

Use explicit constructors and `registerer.Register`; never use `promauto` or
`MustRegister`.

```go
func registerAll(
	registerer prometheus.Registerer,
	collectors ...prometheus.Collector,
) error {
	for _, collector := range collectors {
		if err := registerer.Register(collector); err != nil {
			return err
		}
	}
	return nil
}
```

Use these histogram buckets:

```go
var durationBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30,
}
```

Normalize every label through fixed maps. Classify an error as `timeout` when
`errors.Is(err, context.DeadlineExceeded)`, `error` for any other non-nil
error, and `success` otherwise.

Initialize one zero-valued child for vector families needed by release smoke:

```go
m.authorizationDenials.WithLabelValues("unknown", "unknown")
m.discoveryKeylessRefresh.WithLabelValues("success")
m.readinessChecks.WithLabelValues("aggregate", "ready")
m.readinessStatus.WithLabelValues("aggregate").Set(0)
m.shutdownPhases.WithLabelValues("unknown", "success")
m.shutdowns.WithLabelValues("success")
```

Do not initialize the scrape last-completed timestamp to zero; absence means
no run has completed.

- [ ] **Step 4: Rehome existing observers without renaming metrics**

Change `discovery.DefaultObserver()` to return `NoopObserver{}`. Production
will inject `NewPrometheusObserver(appRegistry)` in Task 4. Keep
`NewPrometheusObserver` and both existing metric names unchanged.
Initialize `resolution_total{outcome="not_found",match_kind="none"}` and
`key_conflicts_total{operation="unknown"}` to zero so both families are
present before the first event.

Keep `scrape.DefaultMetricsObserver` only as a compatibility helper for
non-bootstrap tests; production will inject `NewMetricsObserver(appRegistry)`.
Ensure neither package registers a collector at package initialization.

- [ ] **Step 5: Verify**

```bash
cd go_server
go test -race ./internal/metrics ./internal/discovery ./internal/scrape \
  -run 'Test(Application|ResolverRecordsBounded|MetricsObserver)' -count=1
```

Expected:

```text
ok  	comics/internal/metrics
ok  	comics/internal/discovery
ok  	comics/internal/scrape
```

- [ ] **Step 6: Commit**

```bash
git add go_server/internal/metrics \
  go_server/internal/discovery/metrics.go \
  go_server/internal/discovery/resolver_test.go \
  go_server/internal/scrape/metrics.go \
  go_server/internal/scrape/metrics_test.go
git commit -m "feat(observability): own bounded application metrics"
```

---

### Task 2: Safe Correlation and Optional OpenTelemetry

**Files:**

- Create: `go_server/internal/correlation/context.go`
- Create: `go_server/internal/correlation/context_test.go`
- Modify: `go_server/api/middleware/logging.go`
- Modify: `go_server/api/middleware/logging_test.go`
- Modify: `go_server/internal/logger/logger.go`
- Create: `go_server/internal/logger/logger_test.go`
- Modify: `go_server/internal/tracer/tracer.go`
- Create: `go_server/internal/tracer/tracer_test.go`
- Modify: `go_server/internal/repo/mongo/user_repo.go`
- Modify: `go_server/internal/repo/mongo/user_repo_test.go`
- Modify: `go_server/bootstrap/db.go`
- Modify: `go_server/bootstrap/app.go`
- Modify: `go_server/bootstrap/env.go`
- Modify: `go_server/local.env`

**Interfaces:**

```go
func ValidID(value string) bool
func NewID() string
func WithRequestID(context.Context, string) context.Context
func RequestID(context.Context) string
func WithRunID(context.Context, string) context.Context
func RunID(context.Context) string
func EnrichLogger(context.Context, zerolog.Logger) zerolog.Logger
```

```go
type SpanStarter interface {
	StartSpan(context.Context, string) (context.Context, tracer.Span)
}
```

OTel exporter values are exactly `none`, `stdout`, and `otlp`.

- [ ] **Step 1: Write red correlation/privacy tests**

In `logging_test.go`, send:

```go
req := httptest.NewRequest(
	http.MethodPost,
	"/comics?token=query-secret",
	strings.NewReader(`{"password":"body-secret","title":"private title"}`),
)
req.Header.Set("Authorization", "Bearer header-secret")
req.Header.Set("Cookie", "refresh_token=cookie-secret")
req.Header.Set("X-Request-ID", "request-safe-1")
```

Assert the response returns `X-Request-ID: request-safe-1`; the JSON log
contains `event`, `method`, route, status, duration, and `request_id`; and the
log contains none of:

```text
query-secret
body-secret
private title
header-secret
cookie-secret
```

Add a test that an invalid request ID containing a space is replaced. Add
`context_test.go` assertions that IDs are at most 128 ASCII characters from
`[A-Za-z0-9._:-]` and that valid request/run IDs and valid OTel trace/span IDs
round-trip.

Add tracer tests named `TestNewTracerNoneDoesNotDial`,
`TestNewTracerRejectsUnknownExporter`, and
`TestNewTracerOTLPRequiresEndpoint`. The first uses a deliberately unreachable
endpoint and must still construct and shut down successfully.

- [ ] **Step 2: Run the red tests**

```bash
cd go_server
go test ./api/middleware ./internal/correlation ./internal/tracer \
  -run 'Test(LoggerMiddleware|Correlation)' -count=1
```

Expected:

```text
FAIL	comics/api/middleware
structured log leaked "query-secret"
stat /Users/esteb/dev/maister/comics/go_server/internal/correlation: directory not found
unknown field Exporter in struct literal of type Config
```

- [ ] **Step 3: Add context correlation**

Use a private typed context key and `xid.New().String()`. `EnrichLogger` adds
only nonempty valid IDs:

```go
func EnrichLogger(ctx context.Context, base zerolog.Logger) zerolog.Logger {
	fields := base.With()
	if value := RequestID(ctx); value != "" {
		fields = fields.Str("request_id", value)
	}
	if value := RunID(ctx); value != "" {
		fields = fields.Str("run_id", value)
	}
	span := trace.SpanContextFromContext(ctx)
	if span.HasTraceID() {
		fields = fields.Str("trace_id", span.TraceID().String())
	}
	if span.HasSpanID() {
		fields = fields.Str("span_id", span.SpanID().String())
	}
	return fields.Logger()
}
```

- [ ] **Step 4: Replace request logging with metadata-only JSON**

`LoggerMiddleware` must:

1. validate or generate `X-Request-ID`;
2. place it in the request context before `c.Next()`;
3. return it in the response;
4. log `event=http_request`, method, `c.FullPath()`, status, and duration;
5. use `unknown` when `FullPath()` is empty;
6. skip successful `/health`, `/ready`, and `/metrics`;
7. never read the body, append `RawQuery`, or serialize `c.Errors`.

Delete `sensitiveDataFilterToLog`. Metadata-only logging is the privacy
boundary; an incomplete denylist is not.

Write JSON to stderr by default. Add the rotating file writer only when
`LOG_OUTPUT_FILE` is nonempty.

- [ ] **Step 5: Make tracing explicit and vendor-neutral**

Add:

```go
Exporter string `mapstructure:"OTEL_TRACES_EXPORTER"`
```

Use:

```go
switch strings.ToLower(strings.TrimSpace(cfg.Exporter)) {
case "", "none":
	provider := trace.NewNoopTracerProvider()
	otel.SetTracerProvider(provider)
	return &Tracer{
		provider: provider,
		tracer: provider.Tracer(cfg.ServiceName),
	}, nil
case "stdout":
	exporter, err = stdouttrace.New()
case "otlp":
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, errors.New(
			"OTEL_EXPORTER_GRPC_ENDPOINT is required when OTEL_TRACES_EXPORTER=otlp",
		)
	}
	exporter, err = otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(clientOpts...),
	)
default:
	return nil, fmt.Errorf(
		"unsupported OTEL_TRACES_EXPORTER %q",
		cfg.Exporter,
	)
}
```

Install the composite TraceContext/Baggage propagator for all three modes.
Retain the ratio sampler and five-second provider shutdown. Add:

```env
OTEL_TRACES_EXPORTER=none
OTEL_EXPORTER_GRPC_ENDPOINT=
```

to `go_server/local.env`.

- [ ] **Step 6: Centralize repository tracing**

Construct one tracer after logging and before repositories. Pass its
`SpanStarter` to `mongo.NewUserRepo`; remove tracer creation and shutdown from
`mongo.UserRepo`. The application shuts the provider after repositories.

Delete Mongo span/log fields named `id`, `user_id`, `email`, `username`,
`role`, or `query`. Replace `.Err(err)` with stable `error_code` values:
`not_found`, `invalid_argument`, `canceled`, `timeout`, or `database_error`.

- [ ] **Step 7: Verify**

```bash
cd go_server
go test -race ./api/middleware ./internal/correlation ./internal/logger \
  ./internal/tracer ./internal/repo/mongo ./bootstrap -count=1
```

Expected package lines:

```text
ok  	comics/api/middleware
ok  	comics/internal/correlation
ok  	comics/internal/logger
ok  	comics/internal/tracer
ok  	comics/internal/repo/mongo
ok  	comics/bootstrap
```

- [ ] **Step 8: Commit**

```bash
git add go_server/api/middleware/logging.go \
  go_server/api/middleware/logging_test.go \
  go_server/internal/correlation \
  go_server/internal/logger \
  go_server/internal/tracer \
  go_server/internal/repo/mongo \
  go_server/bootstrap/app.go \
  go_server/bootstrap/db.go \
  go_server/bootstrap/env.go \
  go_server/local.env
git commit -m "feat(observability): correlate metadata-only telemetry"
```

---

### Task 3: Extend the SQLite/PostgreSQL Repository Contract Matrix

**Files:**

- Modify: `go_server/internal/repo/comics/title_keys.go`
- Modify: `go_server/internal/repo/comics/repository_contract_test.go`
- Verify: `go_server/internal/repo/comics/repository_contract_postgres_test.go`
- Verify: `go_server/Makefile`

**Interfaces:**

```go
func (r *SQLComicRepository) CountActiveWithoutDiscoveryKeys(
	context.Context,
) (int, error)
```

Preserve Workstream 3's:

```go
func runRepositoryContract(t *testing.T, repo *SQLComicRepository)
```

Do not add an environment-skipped PostgreSQL test or a second contract
factory. `TestPostgresRepositoryContract` continues to use PostgreSQL 16
testcontainers under the `integration` build tag.

- [ ] **Step 1: Add the failing shared scenarios**

Add these named subtests to `runRepositoryContract`:

```go
t.Run("all fields round trip", contractAllFieldsRoundTrip(repo))
t.Run("merge transaction transfers keys atomically", contractMergeTransaction(repo))
t.Run("active rows always own keys", contractNoActiveKeylessRows(repo))
```

`contractAllFieldsRoundTrip` covers titles, chapter fields, cover and
visibility, timestamps, type, status, publishers, genres, description, author,
tracking, rating, and deletion.

`contractMergeTransaction` creates a base and duplicate, then inside
`WithTx` deletes the duplicate's keys, updates the normalized base with both
title sets, and deletes the duplicate. It asserts the duplicate is not found,
the base owns every expected key, and an injected callback error rolls back
both rows and all keys.

`contractNoActiveKeylessRows` creates active series and novel rows plus one
deleted row, then asserts:

```go
count, err := repo.CountActiveWithoutDiscoveryKeys(context.Background())
if err != nil {
	t.Fatal(err)
}
if count != 0 {
	t.Fatalf("active keyless rows = %d, want 0", count)
}
```

- [ ] **Step 2: Run the red SQLite contract**

```bash
cd go_server
go test ./internal/repo/comics \
  -run '^TestSQLiteRepositoryContract$/active_rows_always_own_keys$' \
  -count=1
```

Expected:

```text
repo.CountActiveWithoutDiscoveryKeys undefined (type *SQLComicRepository has no field or method CountActiveWithoutDiscoveryKeys)
```

- [ ] **Step 3: Implement the dialect-aware count**

SQLite:

```sql
SELECT COUNT(*)
FROM comics AS c
WHERE c.deleted = 0
  AND NOT EXISTS (
    SELECT 1
    FROM comic_title_keys AS k
    WHERE k.comic_id = c.id
  )
```

PostgreSQL uses `c.deleted = false`. Execute with `QueryRowContext`. Wrap
errors as `count active comics without discovery keys`; never log rows, IDs, or
titles.

- [ ] **Step 4: Run both dialects**

```bash
cd go_server
make repo-contract-test
```

Expected:

```text
ok  	comics/internal/repo/comics
ok  	comics/internal/repo/comics
```

No output may contain `COMICS_POSTGRES_URL not set`, `SKIP`, or a connection
string.

- [ ] **Step 5: Commit**

```bash
git add go_server/internal/repo/comics/title_keys.go \
  go_server/internal/repo/comics/repository_contract_test.go
git commit -m "test(db): extend sqlite postgres verification contract"
```

---

### Task 4: Authorization, Discovery, Keyless, and Metrics Endpoint Wiring

**Files:**

- Create: `go_server/internal/metrics/keyless.go`
- Create: `go_server/internal/metrics/keyless_test.go`
- Modify: `go_server/api/middleware/auth.go`
- Modify: `go_server/api/middleware/auth_test.go`
- Modify: `go_server/api/route/route.go`
- Modify: `go_server/api/route/auth_route_test.go`
- Modify: `go_server/api/route/catalog_security_test.go`
- Modify: `go_server/internal/discovery/metrics.go`
- Modify: `go_server/internal/usecase/comics.go`
- Modify: `go_server/internal/scrape/types.go`
- Modify: `go_server/internal/scrape/register.go`
- Modify: `go_server/internal/scrape/runner.go`
- Modify: `go_server/internal/scrape/scrape_test.go`
- Modify: `go_server/bootstrap/app.go`
- Modify: `go_server/bootstrap/env.go`
- Modify: `go_server/bootstrap/sanitize.go`
- Modify: `go_server/bootstrap/sanitize_test.go`
- Modify: `go_server/cmd/server/main.go`
- Modify: `go_server/cmd/scrape/main.go`
- Modify: `go_server/local.env`

**Interfaces:**

```go
type AuthorizationObserver interface {
	ObserveAuthorizationDenial(route, reason string)
}

func BearerAuthenticationMiddleware(
	auth domain.AccessAuthenticator,
	observer AuthorizationObserver,
) gin.HandlerFunc

func RoleMiddleware(
	requiredRole string,
	observer AuthorizationObserver,
) gin.HandlerFunc
```

```go
type KeylessSource interface {
	CountActiveWithoutDiscoveryKeys(context.Context) (int, error)
}

func NewKeylessAuditor(
	source KeylessSource,
	recorder *metrics.Application,
	interval time.Duration,
) *KeylessAuditor

func (a *KeylessAuditor) Refresh(context.Context) error
func (a *KeylessAuditor) Start(context.Context) error
```

- [ ] **Step 1: Add red authorization and keyless tests**

Use a fresh application registry in middleware tests. Make one missing-token
request and one user-role request to an admin route, then compare:

```text
# HELP comics_authorization_denials_total Authorization denials by route and stable reason.
# TYPE comics_authorization_denials_total counter
comics_authorization_denials_total{reason="forbidden_role",route="/comics"} 1
comics_authorization_denials_total{reason="missing_token",route="/comics"} 1
comics_authorization_denials_total{reason="unknown",route="unknown"} 0
```

In `keyless_test.go`, verify an initial count of `346`, a later count of `0`,
and error retention:

```go
recorder.SetDiscoveryKeylessActive(7)
auditor := NewKeylessAuditor(
	keylessSourceStub{err: errors.New("database unavailable")},
	recorder,
	time.Minute,
)
if err := auditor.Refresh(context.Background()); err == nil {
	t.Fatal("expected refresh error")
}
if got := testutil.ToFloat64(recorder.discoveryKeylessActive); got != 7 {
	t.Fatalf("keyless gauge changed after error: %v", got)
}
```

- [ ] **Step 2: Run the red tests**

```bash
cd go_server
go test ./api/middleware ./internal/metrics \
  -run 'Test(Authorization|Keyless)' -count=1
```

Expected:

```text
too many arguments in call to BearerAuthenticationMiddleware
undefined: NewKeylessAuditor
```

- [ ] **Step 3: Observe denials without exposing credentials**

Record immediately before aborting:

- no header: `missing_token`;
- non-Bearer or empty bearer value: `malformed_token`;
- `errors.Is(err, domain.ErrUserInactive)`: `inactive_user`;
- other authentication failure: `invalid_token`;
- no trusted role in context: `missing_role`;
- role mismatch: `forbidden_role`.

Normalize `c.FullPath()` through the metric route allowlist. Do not include the
token, user, role value, or auth error in logs or spans. Update every middleware
callsite in route setup and tests.

- [ ] **Step 4: Reuse the existing discovery observer**

Construct exactly one:

```go
discoveryObserver := discovery.NewPrometheusObserver(appRegistry)
```

Pass it to:

```go
usecase.NewComicServiceWithObserver(repo, coordinator, discoveryObserver)
```

and add it to `scrape.RunnerOptions`:

```go
type RunnerOptions struct {
	LoadJobs         func() ([]PublisherJob, error)
	Retry            RetryPolicy
	Observer         Observer
	DiscoveryObserver discovery.Observer
}
```

The registrar uses the injected observer. Production must not call
`discovery.DefaultObserver()`. Preserve
`comics_discovery_key_conflicts_total{operation}` exactly; do not add a second
conflict counter or add `scope`, title key, or owner ID labels.

- [ ] **Step 5: Implement the keyless auditor**

Use the concrete `SQLComicRepository` as the `KeylessSource`; do not expand
`domain.ComicRepository` solely for telemetry.

Change bootstrap construction to retain that narrow source:

```go
func newComicStore(
	ctx context.Context,
	env *Env,
	discoveryObserver discovery.Observer,
	scrapeObserver scrape.Observer,
) (
	domain.ComicUseCase,
	*scrape.Coordinator,
	metrics.KeylessSource,
	error,
)
```

`Refresh` uses a five-second child context. On success, set the gauge and
increment `{outcome="success"}`. On failure, preserve the previous gauge,
increment `{outcome="error"}`, and log:

```text
event=discovery_keyless_refresh outcome=error error_code=catalog_unavailable
```

`Start` refreshes synchronously before serving and returns that error, then
starts a five-minute ticker only after the initial refresh succeeds. Bootstrap
fails closed on an initial audit error so a default zero cannot be mistaken for
a healthy catalog.

- [ ] **Step 6: Own and protect metrics exposition**

Add:

```go
MetricsEnabled bool   `mapstructure:"METRICS_ENABLED"`
MetricsToken   string `mapstructure:"METRICS_TOKEN"`
```

Defaults:

```text
METRICS_ENABLED=false
METRICS_TOKEN=
```

Set `METRICS_ENABLED=true` in `go_server/local.env`. Production startup fails
when metrics are enabled and the token is empty. `bootstrap.Sanitize` must
redact `METRICS_TOKEN`.

Create one application registry and register:

```go
operationalMetrics, err := metrics.NewApplication(appRegistry)
discoveryObserver := discovery.NewPrometheusObserver(appRegistry)
scrapeObserver := scrape.NewMetricsObserver(appRegistry)
```

Retain these application-owned values:

```go
MetricsRegistry    *prometheus.Registry
OperationalMetrics *metrics.Application
KeylessAuditor     *metrics.KeylessAuditor
Tracer              *tracer.Tracer
```

Expose:

```go
promhttp.HandlerFor(
	prometheus.Gatherers{
		prometheus.DefaultGatherer,
		appRegistry,
	},
	promhttp.HandlerOpts{EnableOpenMetrics: true},
)
```

The default gatherer retains Go/process and existing bounded Gin metrics. The
application registry owns all domain/runtime metrics.
Delete route-time `prometheus.MustRegister` calls, including repeated build
collector registration; constructing more than one router in tests must not
mutate the global registry or panic.

When disabled, `/metrics` returns `404` and records `metrics_disabled`. In
production, compare:

```go
expected := []byte("Bearer " + env.MetricsToken)
provided := []byte(c.GetHeader("Authorization"))
authorized := len(expected) == len(provided) &&
	subtle.ConstantTimeCompare(expected, provided) == 1
```

Wrong credentials return `401` and record `invalid_metrics_token`. Never log
either byte slice.

- [ ] **Step 7: Verify**

```bash
cd go_server
go test -race ./api/middleware ./internal/metrics ./internal/discovery \
  ./internal/usecase ./internal/scrape ./api/route ./bootstrap -count=1
```

Expected: every listed package prints `ok` and no duplicate collector panic
occurs when route fixtures construct more than one application.

- [ ] **Step 8: Commit**

```bash
git add go_server/internal/metrics/keyless.go \
  go_server/internal/metrics/keyless_test.go \
  go_server/api/middleware/auth.go \
  go_server/api/middleware/auth_test.go \
  go_server/api/route \
  go_server/internal/discovery/metrics.go \
  go_server/internal/usecase/comics.go \
  go_server/internal/scrape/types.go \
  go_server/internal/scrape/register.go \
  go_server/internal/scrape/runner.go \
  go_server/internal/scrape/scrape_test.go \
  go_server/bootstrap \
  go_server/cmd/server/main.go \
  go_server/cmd/scrape/main.go \
  go_server/local.env
git commit -m "feat(observability): measure auth and catalog integrity"
```

---

### Task 5: Scrape Outcomes, Duration, Completion, and Correlation

**Files:**

- Modify: `go_server/internal/scrape/metrics.go`
- Modify: `go_server/internal/scrape/metrics_test.go`
- Create: `go_server/internal/scrape/observability_test.go`
- Modify: `go_server/internal/scrape/coordinator.go`
- Modify: `go_server/internal/scrape/coordinator_test.go`
- Modify: `go_server/internal/scrape/runner.go`
- Modify: `go_server/internal/scrape/register_log.go`
- Modify: `go_server/internal/scrape/periodic.go`
- Modify: `go_server/internal/scrape/periodic_test.go`
- Modify: `go_server/cmd/scrape/main.go`
- Modify: `go_server/bootstrap/app.go`

**Interfaces:**

- Preserve `scrape.Observer`:
  `RunStarted`, `RunFinished`, `PublisherFinished`, and `FetchRetry`.
- Extend `MetricsObserver` with
  `comics_scrape_last_completed_timestamp_seconds{source,state}`.
- Replace Workstream 5's private run-ID context helpers with
  `correlation.WithRunID` and `correlation.RunID`.

- [ ] **Step 1: Add failing observer tests**

Extend `TestMetricsObserverRecordsTerminalRunAndPublisher`:

```go
metrics.RunStarted(run)
metrics.PublisherFinished(outcome)
metrics.RunFinished(run)

if got := testutil.ToFloat64(
	metrics.lastCompleted.WithLabelValues("manual", "complete"),
); got != float64(finished.Unix()) {
	t.Fatalf("last completed = %v, want %v", got, finished.Unix())
}
```

Add table cases for `partial_failure`, `total_failure`, and `canceled`; assert
run counters and durations are recorded once. A run without `FinishedAt` must
not update the timestamp.

In `observability_test.go`, run a fixture with:

```text
title: Private scrape title
url: https://publisher.invalid/private-token
error: database password leaked
```

Assert logs contain the run ID, publisher, terminal state, stable failure kind
and code, but none of those three values.

- [ ] **Step 2: Run the red tests**

```bash
cd go_server
go test ./internal/scrape \
  -run 'Test(MetricsObserverRecordsTerminal|ScrapeLogs)' -count=1
```

Expected:

```text
metrics.lastCompleted undefined
```

- [ ] **Step 3: Extend the existing observer**

Create the gauge vector with labels `source,state`. In `RunFinished`, after
the existing counter/duration logic, set it to `FinishedAt.Unix()` only for a
terminal run with non-nil `FinishedAt`.

Continue to enforce:

- coordinator owns `RunStarted` and `RunFinished`;
- runner owns `PublisherFinished` and `FetchRetry`;
- routes, use cases, scheduler, and CLI never duplicate terminal observation.

Preinitialize bounded scrape counter/histogram families with existing domain
constants so release smoke can see them before the first run. Do not
preinitialize the timestamp.

- [ ] **Step 4: Correlate and sanitize scrape logs**

The coordinator attaches the accepted `domain.ScrapeRun.ID`:

```go
runCtx = correlation.WithRunID(runCtx, run.ID)
```

Use `correlation.EnrichLogger` in coordinator, runner, registrar logging,
periodic scheduler, and CLI.

Allowed scrape fields:

```text
event
run_id
request_id
trace_id
span_id
source
publisher
state
failure_kind
failure_code
disposition
items_seen
created
updated
unchanged
skipped
failed
duration_ms
```

Delete title, URL, raw error, and response-body fields. Stable fetch/parse/
registration failure codes remain available through Workstream 5's domain
report and metrics.

- [ ] **Step 5: Verify**

```bash
cd go_server
go test -race ./internal/scrape ./cmd/scrape ./bootstrap -count=1
```

Expected:

```text
ok  	comics/internal/scrape
ok  	comics/cmd/scrape
ok  	comics/bootstrap
```

- [ ] **Step 6: Commit**

```bash
git add go_server/internal/scrape \
  go_server/internal/correlation/context.go \
  go_server/cmd/scrape/main.go \
  go_server/bootstrap/app.go
git commit -m "feat(observability): correlate scrape outcome telemetry"
```

---

### Task 6: Readiness and Shutdown Signals

**Files:**

- Modify: `go_server/internal/health/health.go`
- Modify: `go_server/internal/health/health_test.go`
- Modify: `go_server/internal/lifecycle/shutdown.go`
- Modify: `go_server/internal/lifecycle/shutdown_test.go`
- Modify: `go_server/api/route/route.go`
- Modify: `go_server/bootstrap/app.go`
- Modify: `go_server/bootstrap/app_test.go`
- Modify: `go_server/cmd/server/main.go`

**Interfaces:**

```go
type health.Dependency struct {
	Name   string
	Pinger health.Pinger
}

type health.Observer interface {
	ObserveReadiness(dependency, outcome string, elapsed time.Duration)
	SetReadinessStatus(dependency string, ready bool)
}

func health.NewObservedChecker(
	timeout time.Duration,
	observer health.Observer,
	dependencies ...health.Dependency,
) *health.Checker
```

Keep `health.NewChecker(timeout, pingers...)` as a compatibility wrapper.

```go
type lifecycle.Observer interface {
	ObserveShutdownPhase(phase string, elapsed time.Duration, err error)
	ObserveShutdown(elapsed time.Duration, err error)
}
```

- [ ] **Step 1: Write failing readiness observation tests**

Create named `users` and `catalog` pingers. Make users succeed and catalog
fail. Assert one observation for each checked dependency plus aggregate:

```go
want := []observedReadiness{
	{dependency: "users", outcome: "ready"},
	{dependency: "catalog", outcome: "not_ready"},
	{dependency: "aggregate", outcome: "not_ready"},
}
```

Assert `/ready` returns `503`, `/health` returns `200` without pinging either
dependency, and `SetDraining` immediately sets aggregate readiness to zero
without a datastore call.

- [ ] **Step 2: Write failing shutdown observation tests**

Extend lifecycle tests with an HTTP drain timeout and repository close error.
Assert:

- `readiness` is observed before drain work;
- `http`, `scheduler`, and `scrape` retain Workstream 4's concurrent drain;
- `application` starts only after the drain group finishes;
- timeout maps to `timeout`, not `error`;
- aggregate shutdown is observed once;
- final log contains `event=shutdown_complete`, `shutdown_id`, `outcome`, and
  `duration_ms`, but not raw error text.

- [ ] **Step 3: Run red tests**

```bash
cd go_server
go test ./internal/health ./internal/lifecycle ./bootstrap \
  -run 'Test(ObservedReadiness|ManagerObserves|ApplicationObserves)' -count=1
```

Expected:

```text
undefined: NewObservedChecker
unknown field Observer in struct literal of type Manager
```

- [ ] **Step 4: Instrument synchronous readiness**

`Ready` keeps Workstream 4's one bounded context and checks named dependencies
in order. Observe each duration and result. Stop after the first failed
dependency, set unchecked later dependencies to status `0`, and record
aggregate once.

Construct production readiness as:

```go
health.NewObservedChecker(
	env.ReadinessTimeout,
	readinessObserver,
	health.Dependency{Name: "users", Pinger: authStore},
	health.Dependency{Name: "catalog", Pinger: comics},
)
```

Log only:

```text
event=readiness_check dependency=catalog outcome=not_ready error_code=unavailable
```

- [ ] **Step 5: Instrument shutdown without changing ordering**

Wrap the existing Workstream 4 phases with elapsed timing. Keep:

1. readiness enters draining;
2. HTTP, scheduler, and scrape drain concurrently under `DrainTimeout`;
3. forced HTTP close occurs only after drain timeout;
4. application resources close under their separate `CloseTimeout`.

Change bootstrap shutters to named values:

```go
type Shutter struct {
	Name  string
	Close func(context.Context) error
}
```

Use names `catalog`, `users`, `tracing`, `logging`, and `instance_lock`.
Observe each exactly once while preserving the existing close order and
`errors.Join` behavior. Replace shutdown `.Err(err)` logging with bounded
`error_code=timeout` or `error_code=close_failed`; return the joined error to
the caller without serializing it.

Because Prometheus may not scrape after process exit, emit the bounded
`shutdown_complete` JSON event. That log is authoritative for shutdown
timeouts; the counter is useful for in-process tests and graceful restarts.

- [ ] **Step 6: Verify**

```bash
cd go_server
go test -race ./internal/health ./internal/lifecycle ./bootstrap ./cmd/server \
  -count=1
```

Expected:

```text
ok  	comics/internal/health
ok  	comics/internal/lifecycle
ok  	comics/bootstrap
ok  	comics/cmd/server
```

- [ ] **Step 7: Commit**

```bash
git add go_server/internal/health \
  go_server/internal/lifecycle \
  go_server/api/route/route.go \
  go_server/bootstrap/app.go \
  go_server/bootstrap/app_test.go \
  go_server/cmd/server/main.go
git commit -m "feat(observability): measure readiness and shutdown"
```

---

### Task 7: Top-Level Route Policy and HTTP Telemetry Integration

**Files:**

- Create: `go_server/api/route/route_policy_integration_test.go`
- Create: `go_server/api/route/observability_integration_test.go`
- Modify: `go_server/api/route/catalog_security_test.go`
- Modify: `go_server/api/route/comics_contract_test.go`
- Modify: `go_server/api/route/route.go`
- Modify: `go_server/api/middleware/auth_test.go`
- Modify: `go_server/cmd/server/main.go`
- Modify: `go_server/Makefile`
- Modify: `Makefile`

**Final route setup interface:**

```go
type Observability struct {
	Gatherer             prometheus.Gatherer
	AuthorizationObserver middleware.AuthorizationObserver
	ReadinessObserver     health.Observer
}

func Setup(
	env *bootstrap.Env,
	authStore domain.AuthStore,
	comics domain.ComicUseCase,
	observability Observability,
	g *gin.Engine,
) (*health.Checker, error)
```

This reconciles Workstream 4's returned readiness checker with Workstream 6's
setup error. `cmd/server` passes the returned checker to the lifecycle manager
and handles the error before listening.

- [ ] **Step 1: Build a production-equivalent route fixture**

Use a migrated temporary SQLite repository, existing in-memory/fake auth store,
trusted user/admin identities, one injected registry, and one log buffer.
Seed comics 1 and 2 in every mutation subtest.

The public matrix is:

```go
var publicRoutes = []routeCase{
	{Method: http.MethodGet, Path: "/health"},
	{Method: http.MethodGet, Path: "/ready"},
	{Method: http.MethodGet, Path: "/comics"},
	{Method: http.MethodGet, Path: "/comics/1"},
	{Method: http.MethodGet, Path: "/comics/search/policy"},
}
```

The bearer-admin matrix is:

```go
var adminRoutes = []routeCase{
	{Method: http.MethodPost, Path: "/comics",
		Body: `{"titles":["Policy create"],"com_type":3}`},
	{Method: http.MethodPut, Path: "/comics/1",
		Body: `{"rating":4}`},
	{Method: http.MethodDelete, Path: "/comics/1"},
	{Method: http.MethodPatch, Path: "/comics/1/cover-visibility",
		Body: `{"cover":"cover.webp","cover_visible":false}`},
	{Method: http.MethodPatch, Path: "/comics/1/2"},
	{Method: http.MethodPut, Path: "/comics/1/2"},
	{Method: http.MethodPost, Path: "/scrape"},
	{Method: http.MethodGet, Path: "/scrape"},
	{Method: http.MethodGet, Path: "/scrape/status"},
	{Method: http.MethodGet, Path: "/scrape/history?limit=10"},
}
```

For every admin route, assert:

- no bearer returns `401`;
- a valid active user bearer returns `403`;
- an active admin reaches the handler and does not return `401`, `403`, or
  `404`;
- `X-Request-ID` is present and valid.

- [ ] **Step 2: Run the current-baseline red policy test**

```bash
cd go_server
go test ./api/route -run TestRoutePolicyIntegration -count=1 -v
```

Expected before Workstream 1:

```text
POST /comics anonymous: got 201, want 401
```

After the dependency gate, the new test must pass without weakening the
assertion.

- [ ] **Step 3: Add end-to-end telemetry assertions**

Send an anonymous mutation with:

```text
X-Request-ID: policy-denial-1
traceparent: 00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01
```

Then trigger an authorized typed create conflict and a catalog readiness
failure. Assert logs contain request ID `policy-denial-1` and trace ID
`0102030405060708090a0b0c0d0e0f10`, with no request body, title, token, or raw
error.

Gather the injected registry and assert:

```text
comics_authorization_denials_total{reason="missing_token",route="/comics"} 1
comics_discovery_key_conflicts_total{operation="create"} 1
comics_readiness_status{dependency="catalog"} 0
comics_readiness_status{dependency="aggregate"} 0
```

Assert request and trace IDs are absent from every metric descriptor and
sample label.

Test `/metrics`:

- testing + enabled: `200`;
- disabled: `404`;
- production + wrong bearer: `401`;
- production + exact metrics bearer: `200`;
- response contains all final metric family names.

- [ ] **Step 4: Keep handler and policy contracts separate**

Rename the two direct-handler tests:

```text
TestHTTPContractListSearchAndDuplicate
TestHTTPContractUpdateMergeDelete
```

They may mount focused handlers. Policy tests must call top-level `Setup`.

Add:

```make
route-policy-test:
	go test ./api/route \
	  -run 'Test(RoutePolicyIntegration|ObservabilityIntegration|HTTPContract)' \
	  -count=1
```

Expose the same root target.

- [ ] **Step 5: Verify**

```bash
make route-policy-test
cd go_server
go test -race ./api/route ./api/middleware -count=1
```

Expected:

```text
ok  	comics/api/route
ok  	comics/api/route
ok  	comics/api/middleware
```

- [ ] **Step 6: Commit**

```bash
git add go_server/api/route \
  go_server/api/middleware/auth_test.go \
  go_server/cmd/server/main.go \
  go_server/Makefile \
  Makefile
git commit -m "test(api): gate route policy and telemetry"
```

---

### Task 8: Frontend Mutation Settlement and Unauthorized Tests

**Files:**

- Modify: `src/frontend/api/client.test.ts`
- Modify: `src/frontend/hooks/useComicActions.test.tsx`
- Modify: `src/frontend/components/Comics/Edition/CreateComic.test.tsx`
- Modify: `src/frontend/components/Comics/Edition/EditComic.test.tsx`
- Modify: `src/frontend/components/Comics/Edition/MergeComic.test.tsx`
- Modify: `src/frontend/components/Comics/Edition/ScrapeButton.test.tsx`
- Modify: `src/frontend/components/Comics/MainPage/MainPage.test.tsx`
- Modify: `Makefile`

**Interfaces reused from Workstream 7:**

```ts
RestClient.request<T>(request: ApiRequest<T>): Promise<ApiResult<T>>
RestClient.setUnauthorizedHandler(
  handler: UnauthorizedHandler | undefined,
): () => void
comicsApi.create
comicsApi.update
comicsApi.delete
comicsApi.merge
comicsApi.runScrape
```

Do not recreate `src/frontend/util/ApiClient.ts`, dispatch a global browser
event, or call `fetch` from a component.

- [ ] **Step 1: Strengthen transport tests**

Retain Workstream 7's bearer and anonymous-read tests. Add:

- an authenticated `401` calls the registered unauthorized callback exactly
  once and rejects with the same `ApiError`;
- `403` rejects but does not call unauthorized handling;
- malformed non-JSON `502` becomes
  `ApiError("Server returned invalid JSON", 502, "INVALID_JSON",
  "invalid-json")`;
- request headers never contain the session-storage key or expose a token in
  an error.

- [ ] **Step 2: Add a complete mutation settlement matrix**

Use deferred promises so each test checks pre-settlement state:

```ts
let resolveMutation!: (comic: Comic) => void;
const pending = new Promise<Comic>((resolve) => {
  resolveMutation = resolve;
});
updateMock.mockReturnValue(pending);
```

For create, edit, delete, merge, and scrape:

- before resolve: list, modal/card state, and fetch count are unchanged;
- success: local state settles and the list invalidation callback runs exactly
  once;
- rejection: current list remains visible, editable input remains intact, no
  success toast is emitted, and invalidation does not run;
- `401`: AuthContext's registered unauthorized handler owns session clearing;
- `partial_failure`: scrape shows partial completion and invalidates once;
- `total_failure` and `canceled`: scrape shows failure and does not invalidate.

The decisive assertions are:

```ts
expect(onMutationSuccess).not.toHaveBeenCalled();
expect(fetchComics).toHaveBeenCalledTimes(1);

resolveMutation(updatedComic);
await pending;

expect(onMutationSuccess).toHaveBeenCalledTimes(1);
expect(fetchComics).toHaveBeenCalledTimes(2);
```

- [ ] **Step 3: Run the red tests**

On the current pre-Workstream-7 baseline:

```bash
npm test -- \
  src/frontend/api/client.test.ts \
  src/frontend/hooks/useComicActions.test.tsx \
  src/frontend/components/Comics/Edition/CreateComic.test.tsx \
  src/frontend/components/Comics/Edition/EditComic.test.tsx \
  src/frontend/components/Comics/Edition/MergeComic.test.tsx \
  src/frontend/components/Comics/Edition/ScrapeButton.test.tsx \
  src/frontend/components/Comics/MainPage/MainPage.test.tsx
```

Expected red output:

```text
Cannot find module './client'
```

After the dependency gate, any new assertion failure identifies a regression
to fix in the existing Workstream 7 client/component, not a reason to add
another transport.

- [ ] **Step 4: Add the focused target**

```make
front-mutation-test:
	npm test -- \
	  src/frontend/api/client.test.ts \
	  src/frontend/hooks/useComicActions.test.tsx \
	  src/frontend/components/Comics/Edition/CreateComic.test.tsx \
	  src/frontend/components/Comics/Edition/EditComic.test.tsx \
	  src/frontend/components/Comics/Edition/MergeComic.test.tsx \
	  src/frontend/components/Comics/Edition/ScrapeButton.test.tsx \
	  src/frontend/components/Comics/MainPage/MainPage.test.tsx
```

Add it to `.PHONY` and grouped help.

- [ ] **Step 5: Verify**

```bash
make front-mutation-test
npm run build
```

Expected:

```text
Test Files  7 passed
vite build completed successfully
```

Do not assert a fixed test count; earlier workstreams may add cases.

- [ ] **Step 6: Commit**

```bash
git add src/frontend/api/client.test.ts \
  src/frontend/hooks/useComicActions.test.tsx \
  src/frontend/components/Comics/Edition/CreateComic.test.tsx \
  src/frontend/components/Comics/Edition/EditComic.test.tsx \
  src/frontend/components/Comics/Edition/MergeComic.test.tsx \
  src/frontend/components/Comics/Edition/ScrapeButton.test.tsx \
  src/frontend/components/Comics/MainPage/MainPage.test.tsx \
  Makefile
git commit -m "test(frontend): verify mutation settlement"
```

---

### Task 9: Representative Migration Rehearsal

**Files:**

- Create:
  `go_server/internal/repo/migrations/testdata/sqlite_pre_title_keys.sql`
- Create: `go_server/internal/repo/migrations/rehearsal_test.go`
- Create:
  `go_server/internal/repo/migrations/rehearsal_postgres_test.go`
- Modify: `go_server/Makefile`
- Modify: `Makefile`

**Interfaces reused from Workstream 3:**

```go
runner, err := migrations.Open(db, migrations.DialectSQLite)
state, err := runner.Up(ctx)
state, err = runner.Down(ctx, 1)
state, err = runner.State()
```

Do not invent `migrations.Apply`, `migrations.Version`, or
`migrations.ConflictError`.

- [ ] **Step 1: Write the red SQLite rehearsal test**

The test opens
`testdata/sqlite_pre_title_keys.sql`, applies it to a temporary SQLite file,
opens a migration runner, and calls `Up`.

Run before adding the fixture:

```bash
cd go_server
go test ./internal/repo/migrations \
  -run '^TestSQLitePreTitleKeyRehearsal$' -count=1
```

Expected:

```text
open testdata/sqlite_pre_title_keys.sql: no such file or directory
```

- [ ] **Step 2: Add the representative fixture**

The fixture contains the full pre-title-key `comics` table shape and:

```sql
PRAGMA foreign_keys = OFF;

CREATE TABLE comics (
  id            INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  titles        TEXT    NOT NULL,
  current_chap  INTEGER NOT NULL DEFAULT 0,
  cover         TEXT    NOT NULL DEFAULT '',
  last_update   INTEGER NOT NULL DEFAULT 0,
  com_type      INTEGER NOT NULL DEFAULT 0,
  status        INTEGER NOT NULL DEFAULT 0,
  published_in  TEXT    NOT NULL DEFAULT '',
  genres        TEXT    NOT NULL DEFAULT '',
  description   TEXT    NOT NULL DEFAULT '',
  author        TEXT    NOT NULL DEFAULT '',
  track         INTEGER NOT NULL DEFAULT 0,
  viewed_chap   INTEGER NOT NULL DEFAULT 0,
  rating        INTEGER NOT NULL DEFAULT 0,
  deleted       INTEGER NOT NULL DEFAULT 0,
  cover_visible INTEGER NOT NULL DEFAULT 1
);

INSERT INTO comics (
  id, titles, current_chap, cover, last_update, com_type, status,
  published_in, genres, description, author, track, viewed_chap,
  rating, deleted, cover_visible
) VALUES
  (
    1, 'The rehearsal hero|Rehearsal hero', 12, '',
    1700000000, 3, 2, '1', '2', 'series row', 'author',
    1, 4, 3, 0, 1
  ),
  (
    2, 'Rehearsal novel (novel)', 40, '',
    1700000001, 5, 2, '', '', 'novel row', 'author',
    0, 10, 2, 0, 1
  ),
  (
    3, 'Deleted rehearsal', 1, '',
    1700000002, 3, 2, '', '', 'deleted row', 'author',
    0, 0, 0, 1, 1
  );
```

It deliberately has no `comic_title_keys` or
`catalog_schema_migrations` table.

- [ ] **Step 3: Assert migration properties**

The SQLite rehearsal asserts:

1. `Runner.Up` returns
   `State{Version: CurrentVersion, Dirty: false}`;
2. all comic columns and values are byte-for-byte equivalent;
3. `PRAGMA integrity_check` returns `ok`;
4. `PRAGMA foreign_keys` returns `1`;
5. `PRAGMA foreign_key_list(comic_title_keys)` has
   `comic_id -> comics(id) ON DELETE CASCADE`;
6. a second `Up` returns the same clean state;
7. the two active fixture rows are reported as keyless, documenting that
   schema migration does not guess repair decisions;
8. deleting comic 1 after inserting a fixture key cascades that key.

The keyless count of `2` is expected only for this synthetic pre-repair
fixture. Production rollout remains blocked until Workstream 2's reviewed
repair reports `0`.

- [ ] **Step 4: Add PostgreSQL down/up rehearsal**

`rehearsal_postgres_test.go` uses:

```go
//go:build integration
```

Start `postgres:16-alpine` with the already-pinned testcontainers module.
Against a fresh database:

```text
Up -> State current/clean -> Down(1) -> State version 1/clean
-> Up -> State current/clean -> ValidateCurrent
```

No test may skip because an environment variable is absent.

- [ ] **Step 5: Add exact Make targets**

```make
migration-rehearsal:
	go test ./internal/repo/migrations \
	  -run '^TestSQLitePreTitleKeyRehearsal$$' -count=1
	go test -tags=integration ./internal/repo/migrations \
	  -run '^TestPostgresMigrationRehearsal$$' -count=1
```

Add the root delegate and `.PHONY` entries.

- [ ] **Step 6: Verify**

```bash
make migration-rehearsal
```

Expected:

```text
ok  	comics/internal/repo/migrations
ok  	comics/internal/repo/migrations
```

- [ ] **Step 7: Commit**

```bash
git add go_server/internal/repo/migrations/rehearsal_test.go \
  go_server/internal/repo/migrations/rehearsal_postgres_test.go \
  go_server/internal/repo/migrations/testdata/sqlite_pre_title_keys.sql \
  go_server/Makefile \
  Makefile
git commit -m "test(migrations): rehearse legacy catalog upgrade"
```

---

### Task 10: Mandatory CI Quality Gate

**Files:**

- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/ci-cd.yml`
- Modify: `ops/tests/deploy_config_test.sh`
- Modify: `Makefile`
- Modify: `go_server/Makefile`

**Interfaces:**

- Produce: `make observability-test`
- Produce: `make observability-check`
- Produce: `make check-ci`
- Preserve Workstream 4's `workflow_run` deployment of the exact successful
  `main` CI SHA.

- [ ] **Step 1: Add aggregate targets**

Go server:

```make
observability-test:
	go test -race \
	  ./internal/metrics \
	  ./internal/correlation \
	  ./internal/health \
	  ./internal/lifecycle \
	  ./internal/scrape \
	  ./api/middleware \
	  ./api/route \
	  ./bootstrap \
	  -count=1
```

Root:

```make
check-ci: check-go check-front repo-contract-test migration-rehearsal \
	route-policy-test front-mutation-test observability-test \
	observability-check ops-test front-e2e
	@echo "check-ci passed"
```

`check-front` remains browser-download-free for local development.
`check-ci` is the release gate.

- [ ] **Step 2: Run the red aggregate**

```bash
make check-ci
```

Expected before the configuration target is added:

```text
make: *** No rule to make target 'observability-check'.  Stop.
```

- [ ] **Step 3: Add configuration validation and require all gates in CI**

Add the root validator:

```make
observability-check:
	docker compose -f go_server/.observability/docker-compose.yaml config -q
	docker run --rm \
	  --entrypoint /bin/promtool \
	  -v "$(CURDIR)/go_server/.observability/prometheus:/etc/prometheus:ro" \
	  prom/prometheus:latest \
	  check config /etc/prometheus/prometheus.yml
```

Add it to `.PHONY` and grouped help. It validates the current stack in this
task; Task 11 replaces the example rules and re-runs the same gate.

The Go job runs:

```yaml
- name: Go unit and race tests
  run: make go-test
- name: SQLite and PostgreSQL repository contract
  run: make repo-contract-test
- name: SQLite and PostgreSQL migration rehearsal
  run: make migration-rehearsal
- name: Route policy and telemetry integration
  run: make route-policy-test
- name: Observability tests
  run: make observability-test
- name: Vet and production build
  run: make check-go
```

Do not add a PostgreSQL service container. Repository and migration gates use
the Workstream 3 testcontainers pattern, preventing silent local/CI skips.

The frontend job retains `npm ci`, Vitest, build, Playwright Chromium install,
and Playwright. Add `make front-mutation-test`.

Add a configuration job:

```yaml
observability_config:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - run: make observability-check
```

- [ ] **Step 4: Preserve tested-SHA deployment**

The deployment trigger remains:

```yaml
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches: [main]
```

The job condition remains:

```yaml
if: >-
  github.event.workflow_run.conclusion == 'success' &&
  github.event.workflow_run.event == 'push'
```

Pass:

```yaml
env:
  DEPLOY_SHA: ${{ github.event.workflow_run.head_sha }}
```

to the existing `ops/deploy-production.sh`. Preserve Workstream 4's verified
SQLite backup, candidate readiness smoke, previous-image rollback, and exact
SHA checkout.

Extend `ops/tests/deploy_config_test.sh` to assert:

```bash
rg -q 'make observability-check' .github/workflows/ci.yml
rg -q 'make repo-contract-test' .github/workflows/ci.yml
rg -q 'make migration-rehearsal' .github/workflows/ci.yml
rg -q 'make route-policy-test' .github/workflows/ci.yml
rg -q 'make front-mutation-test' .github/workflows/ci.yml
```

- [ ] **Step 5: Verify workflow policy**

```bash
bash ops/tests/deploy_config_test.sh
make ops-test
make observability-check
```

Expected:

```text
deploy config tests passed
operations tests passed
 SUCCESS: /etc/prometheus/prometheus.yml is valid prometheus config file syntax
  SUCCESS: 1 rules found
```

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/ci.yml \
  .github/workflows/ci-cd.yml \
  ops/tests/deploy_config_test.sh \
  Makefile \
  go_server/Makefile
git commit -m "ci: require cross-workstream quality gates"
```

---

### Task 11: Optional Observability Stack, Release Checklist, and Rollback SLOs

**Files:**

- Create: `go_server/.observability/grafana/dashboards.yml`
- Modify: `go_server/.observability/docker-compose.yaml`
- Modify: `go_server/.observability/prometheus/prometheus.yml`
- Modify: `go_server/.observability/prometheus/rules.yml`
- Modify: `go_server/.observability/prometheus/alertmanager.yml`
- Modify: `go_server/.observability/grafana/datasources.yml`
- Modify: `go_server/.observability/grafana/grafana.json`
- Modify: `go_server/.observability/vector-config.yml`
- Modify: `docs/operations.md`
- Modify: `docs/deploy-render.md`
- Modify: `README.md`
- Modify: `go_server/README.md`
- Modify: `.gitignore`
- Modify: `Makefile`

**Interfaces:**

- Produce: `make observability-up`
- Produce: `make observability-down`
- Produce: `make observability-check`
- Reuse: `ops/postdeploy-smoke.sh`
- Reuse: `npm run smoke:frontend-api`
- Keep every observability service optional.

- [ ] **Step 1: Replace example rules with release signals**

First run:

```bash
rg -q 'alert: ComicsNotReady' \
  go_server/.observability/prometheus/rules.yml
```

Expected: exit `1` with no output because the current file contains only the
example `InstanceDown` rule.

Use:

```yaml
groups:
  - name: comics-release
    rules:
      - alert: ComicsNotReady
        expr: comics_readiness_status{dependency="aggregate"} == 0
        for: 2m
        labels:
          severity: page
        annotations:
          summary: Comics API has been unready for two minutes

      - alert: ComicsHTTPErrorBudgetBurn
        expr: |
          sum(rate(gin_uri_request_total{code=~"5.."}[5m]))
          /
          clamp_min(sum(rate(gin_uri_request_total[5m])), 0.001)
          > 0.01
        for: 5m
        labels:
          severity: page
        annotations:
          summary: Five-minute HTTP 5xx ratio exceeds one percent

      - alert: ComicsHTTPP95High
        expr: |
          histogram_quantile(
            0.95,
            sum by (le) (
              rate(gin_request_duration_bucket{
                uri!~"/metrics|/health|/ready"
              }[5m])
            )
          ) > 2
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: Non-health HTTP p95 exceeds two seconds

      - alert: ComicsKeylessCatalogRows
        expr: comics_discovery_keyless_active > 0
        for: 5m
        labels:
          severity: page
        annotations:
          summary: Active comics are missing discovery keys

      - alert: ComicsDiscoveryConflict
        expr: increase(comics_discovery_key_conflicts_total[15m]) > 0
        labels:
          severity: warning
        annotations:
          summary: A typed discovery-key conflict occurred

      - alert: ComicsScrapeTotalFailure
        expr: increase(
          comics_scrape_runs_total{state="total_failure"}[30m]
        ) > 0
        labels:
          severity: page
        annotations:
          summary: A scrape run had total failure

      - alert: ComicsScrapeStale
        expr: |
          max(comics_scrape_last_completed_timestamp_seconds) > 0
          and
          time() - max(
            comics_scrape_last_completed_timestamp_seconds
          ) > 1800
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: No scrape run completed in thirty minutes

      - alert: ComicsAuthorizationDenialBurst
        expr: sum(
          increase(comics_authorization_denials_total[5m])
        ) > 20
        labels:
          severity: warning
        annotations:
          summary: Authorization denials exceeded twenty in five minutes
```

Authorization bursts are an investigation signal, not an automatic rollback
signal.

- [ ] **Step 2: Make Compose internally consistent**

Update the optional stack so:

- Prometheus scrapes only `host.docker.internal:8081`;
- Linux receives `host.docker.internal:host-gateway`;
- Jaeger receives OTLP on `4317`;
- Grafana mounts `grafana.json` and `dashboards.yml`;
- Vector tails JSON files under `go_server/.observability/log/`;
- each stateful service uses its own named volume;
- each service has a health check;
- application startup does not depend on this Compose file.

The checked-in Prometheus target is for a development-mode API, where metrics
are enabled without a production bearer. A production Prometheus deployment
must inject its bearer through secret-managed scrape configuration outside the
repository; never commit `METRICS_TOKEN`.

To opt into local log/trace export while running the API on the host:

```bash
LOG_OUTPUT_FILE=.observability/log/comics.jsonl \
OTEL_TRACES_EXPORTER=otlp \
OTEL_EXPORTER_GRPC_ENDPOINT=localhost:4317 \
make -C go_server go-run
```

Provision the dashboard:

```yaml
apiVersion: 1
providers:
  - name: comics
    orgId: 1
    folder: Comics
    type: file
    disableDeletion: false
    updateIntervalSeconds: 30
    options:
      path: /var/lib/grafana/dashboards
```

Give the Prometheus datasource the UID referenced by `grafana.json`:

```yaml
uid: PE34D4A6585786496
```

Add panels for:

```promql
comics_discovery_keyless_active
```

```promql
sum by (operation) (
  increase(comics_discovery_key_conflicts_total[1h])
)
```

```promql
sum by (source, state) (
  increase(comics_scrape_runs_total[1h])
)
```

```promql
histogram_quantile(
  0.95,
  sum by (le, source) (
    rate(comics_scrape_run_duration_seconds_bucket[1h])
  )
)
```

```promql
comics_readiness_status
```

```promql
sum by (route, reason) (
  increase(comics_authorization_denials_total[1h])
)
```

Remove `trace_id`, `span_id`, `request_id`, `run_id`, and `error` from Vector's
Loki labels. Keep correlation IDs in the JSON event body.

Replace fake SMTP with:

```yaml
route:
  receiver: local-null
receivers:
  - name: local-null
```

- [ ] **Step 3: Add optional-stack lifecycle targets**

```make
observability-up:
	mkdir -p go_server/.observability/log
	docker compose -f go_server/.observability/docker-compose.yaml up -d

observability-down:
	docker compose -f go_server/.observability/docker-compose.yaml down
```

Add both to `.PHONY` and grouped help. Retain Task 10's
`observability-check` recipe unchanged.

Add:

```gitignore
go_server/.observability/log/
```

so local JSONL output is never committed.

- [ ] **Step 4: Write the exact release checklist**

Extend `docs/operations.md`.

Pre-release:

```bash
make check-ci
make db-audit-discovery DB_FILE=db/comics.db
make db-backup BACKUP_LABEL=release
```

Required audit output after Workstream 2 repair:

```text
keyless=0
deleted_keys=0
orphan_keys=0
```

Deploy only through the successful-main-CI workflow. Immediate post-deploy:

```bash
SERVICE_URL=http://127.0.0.1:8081 ./ops/postdeploy-smoke.sh
```

Run the existing authenticated create/update/delete smoke with a disposable
admin:

```bash
COMICS_SMOKE_API=https://comic-tracker-server.onrender.com \
COMICS_SMOKE_ADMIN_EMAIL="$PRODUCTION_SMOKE_ADMIN_EMAIL" \
COMICS_SMOKE_ADMIN_PASSWORD="$PRODUCTION_SMOKE_ADMIN_PASSWORD" \
npm run smoke:frontend-api
```

Verify metrics without printing the token:

```bash
metrics_file="$(mktemp)"
trap 'rm -f "$metrics_file"' EXIT
curl --fail --silent --show-error \
  -H "Authorization: Bearer ${PRODUCTION_METRICS_TOKEN}" \
  https://comic-tracker-server.onrender.com/metrics \
  > "$metrics_file"
for name in \
  comics_authorization_denials_total \
  comics_discovery_key_conflicts_total \
  comics_discovery_keyless_active \
  comics_scrape_runs_total \
  comics_scrape_run_duration_seconds \
  comics_readiness_status \
  comics_shutdown_total; do
  rg -q "^# (HELP|TYPE) ${name}" "$metrics_file"
done
```

- [ ] **Step 5: Document rollback SLO signals**

Observe a 15-minute release window. Roll back when a signal began with the
release and any condition is true:

- `/ready` fails continuously for 2 minutes;
- 5xx ratio exceeds 1% for 5 minutes;
- non-health HTTP p95 exceeds 2 seconds for 10 minutes;
- `comics_discovery_keyless_active` rises above `0`;
- typed discovery conflicts begin on previously successful write paths;
- authenticated create/update/delete smoke fails or cleanup cannot delete;
- `shutdown_complete` reports `timeout` or `error`.

Investigate without automatic rollback for:

- authorization-denial bursts without a 5xx/readiness regression;
- one publisher `partial_failure`;
- optional exporter, dashboard, Loki, or Jaeger unavailability while
  application readiness is healthy.

Immediate candidate failures continue to use Workstream 4's automatic
`ops/deploy-production.sh` rollback. For a later SQLite rollback:

```bash
cd /home/dev/comics
set -a
source .deploy/current.env
set +a
export COMICS_ENV_FILE=/home/dev/comics/go_server/.env.production
export COMICS_DATA_DIR=/home/dev/comics-data
docker compose stop -t 30 comics_server
SERVICE_STOPPED=1 \
BACKUP="$PREDEPLOY_BACKUP" \
DB_FILE="$COMICS_DATA_DIR/comics.db" \
BACKUP_DIR="$COMICS_DATA_DIR/backups" \
COMPOSE_FILE="$PWD/docker-compose.yaml" \
./ops/sqlite-restore.sh
git switch --detach "$PREVIOUS_SHA"
export COMICS_IMAGE="comics-go-api:${PREVIOUS_SHA}"
docker compose up -d --no-build --wait --wait-timeout 90 comics_server
SERVICE_URL=http://127.0.0.1:8081 ./ops/postdeploy-smoke.sh
```

If the release made no schema/data change, omit database restore and roll back
the image only. PostgreSQL uses a rehearsed down migration only when that
migration is explicitly reversible; otherwise restore the verified provider
backup and deploy the previous SHA.

- [ ] **Step 6: Update active documentation**

Document in `README.md`, `go_server/README.md`, and `docs/deploy-render.md`:

- `/health` versus `/ready`;
- `METRICS_ENABLED`, `METRICS_TOKEN`, `OTEL_TRACES_EXPORTER`, and
  `OTEL_EXPORTER_GRPC_ENDPOINT`;
- `OTEL_TRACES_EXPORTER=none` requires no collector;
- local Grafana/Jaeger/Loki is optional;
- links to the canonical `docs/operations.md` release/rollback procedure;
- SQLite remains one replica.

- [ ] **Step 7: Verify**

```bash
make observability-check
```

Expected:

```text
Checking /etc/prometheus/prometheus.yml
 SUCCESS: /etc/prometheus/prometheus.yml is valid prometheus config file syntax

Checking /etc/prometheus/rules.yml
  SUCCESS: 8 rules found
```

The Compose validation command exits `0` without output.

Optionally start the local stack:

```bash
make observability-up
curl --fail http://localhost:9090/-/ready
curl --fail http://localhost:3000/api/health
make observability-down
```

Both curl commands return HTTP `200`.

- [ ] **Step 8: Commit**

```bash
git add go_server/.observability \
  docs/operations.md \
  docs/deploy-render.md \
  README.md \
  go_server/README.md \
  .gitignore \
  Makefile
git commit -m "docs(ops): add observable release rollback gates"
```

---

## Final Verification

- [ ] Format changed Go files:

```bash
gofmt -w \
  go_server/internal/metrics/*.go \
  go_server/internal/correlation/*.go \
  go_server/internal/health/*.go \
  go_server/internal/lifecycle/*.go \
  go_server/internal/scrape/*.go \
  go_server/internal/repo/comics/*.go \
  go_server/internal/repo/migrations/*.go \
  go_server/api/middleware/*.go \
  go_server/api/route/*.go \
  go_server/bootstrap/*.go \
  go_server/cmd/server/*.go \
  go_server/cmd/scrape/*.go
```

- [ ] Run the complete release gate:

```bash
npx playwright install chromium
make check-ci
```

Expected final line:

```text
check-ci passed
```

- [ ] Prove no telemetry vendor is required:

```bash
cd go_server
OTEL_TRACES_EXPORTER=none \
METRICS_ENABLED=false \
go test ./bootstrap ./cmd/server -count=1
```

Expected:

```text
ok  	comics/bootstrap
ok  	comics/cmd/server
```

- [ ] Prove labels and logs contain no sensitive values:

```bash
cd go_server
go test ./internal/metrics ./api/middleware ./internal/scrape \
  -run 'Sensitive|Cardinality|Correlation|Application' -count=1
```

Expected: all packages pass.

- [ ] Check whitespace and the final diff:

```bash
git diff --check
git status --short
```

Expected: `git diff --check` prints nothing. Status contains only the intended
implementation files and no generated database backup, telemetry data volume,
token, or production environment file.

---

## Rollout Order

1. Land Workstreams 1 through 8 and pass the dependency gate.
2. Land Tasks 1 through 3: registry, safe correlation, and the extended
   two-dialect contract.
3. Land Tasks 4 through 6: authorization/discovery/keyless, scrape, readiness,
   and shutdown telemetry.
4. Land Tasks 7 through 9: route, frontend mutation, and migration verification.
5. Land Tasks 10 and 11: CI, optional stack, release checklist, and rollback
   signals.
6. Rehearse backup/restore and the representative migration.
7. Complete Workstream 2's reviewed production repair and require
   `comics_discovery_keyless_active == 0`.
8. Deploy the tested CI SHA to one SQLite replica.
9. Run public readiness smoke, authenticated mutation smoke, and metrics
   verification.
10. Observe the 15-minute release window and retain the verified backup and
    previous image until it closes.

PostgreSQL remains optional at runtime but mandatory in contract and migration
CI tests.

---

## Self-Review

- Coverage: authorization denials, discovery conflicts, active keyless count,
  scrape states/durations/completion, readiness, shutdown, correlation,
  SQLite/PostgreSQL contracts, route policy, frontend mutation settlement,
  migration rehearsal, CI, release, SLO, and rollback are each assigned to a
  task and verification command.
- Interface consistency: this plan uses
  `DiscoveryKeyConflictError`, `migrations.Open/Runner`,
  Workstream 5's `scrape.Observer`, Workstream 6's
  `AccessAuthenticator`, and Workstream 7's `RestClient`; it does not use the
  superseded draft names `DiscoveryConflictError`, `migrations.Apply`,
  `PublisherResult`, `apiRequest`, or `UNAUTHORIZED_EVENT`.
- Metrics consistency: existing discovery and scrape names are preserved;
  only missing series are added; IDs and unbounded data are excluded from
  labels.
- Privacy: snippets and tests explicitly reject bodies, queries, credentials,
  URLs, titles, user identifiers, and raw errors in logs/spans.
- Vendor neutrality: `none`, `stdout`, and standard OTLP are supported;
  application readiness does not depend on the optional local stack.
- Data safety: schema rehearsal never guesses discovery repairs; rollout
  requires the reviewed keyless repair, verified SQLite backup, tested SHA,
  readiness smoke, and documented restore.
- Scope: implementation changes are described only; authoring this plan
  modifies no source or configuration file.
