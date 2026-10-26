# Workstream 8: Contract and Legacy Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make React + REST JSON the only supported application contract, then remove the unsupported gRPC stack, generated protobuf clients, divergent repositories, stale Python-era configuration, ambiguous JSON catalog, and empty legacy SQLite users table without losing catalog or user data.

**Architecture:** Start with a hard consumer-inventory gate. The recommended path is retirement because the production Docker image, CI, Make build, React app, and current deployment all use `cmd/server` and REST; the gRPC binary instead owns a separate PostgreSQL-only repository and bypasses the production use case, discovery, transaction, and authorization paths. If the gate finds a real consumer, stop the gRPC/protobuf deletion and open a parity-and-migration plan that routes gRPC through `domain.ComicUseCase`, `domain.ComicRepository`, and the same auth policy before any source is preserved.

**Tech Stack:** Go 1.27.3, Gin REST, React 19, TypeScript 6, Vite 7, Vitest 3, SQLite, optional PostgreSQL, MongoDB, Make, Docker Compose, GitHub CLI

## Global Constraints

- Keep the Go modular monolith as the production service.
- Keep React + REST JSON as the production client contract.
- Keep SQLite as the default catalog store and its single-instance constraint.
- Keep MongoDB as the user/auth store.
- Keep PostgreSQL as an optional catalog backend and scale-up path.
- Do not change REST routes, REST payloads, catalog semantics, or auth policy in this workstream.
- Do not treat OTLP-over-gRPC (`OTEL_EXPORTER_GRPC_ENDPOINT`) as an application gRPC API; tracing remains supported.
- Preserve `db/db_classes.json`: the React app imports it for enum labels, so it is not one of the dead schema artifacts.
- Preserve historical changelogs and approved specs. Remove stale claims only from active setup, deployment, build, and architecture documentation.
- Do not retain `.proto` files as speculative documentation. Preserve them only when the gate identifies a named active consumer, an owner, recent use, and a committed migration/parity path.
- Stop the API before destructive SQLite maintenance, create and verify a `.db` backup first, and refuse to drop a non-empty legacy `users` table.
- Never use `db/comics.json` for runtime, backup, or restore. SQLite `.db` files are canonical.
- Do not commit generated SQLite backups.
- Run Tasks 1–6 in order. Workstreams 3 and 7 touch nearby repository/frontend files; if either lands first, rebase and rerun Gate 0 before executing this plan.
- Workstream 5 must land before Task 3 so its
  `ScrapeButton.test.ts`-to-`.tsx` rename is present; do not recreate the old
  test path.
- Workstream 3's `go_server/cmd/migrate`,
  `go_server/internal/repo/migrations/`, `go-migrate-up`, and
  `go-migrate-down` are the canonical catalog migration path and must remain.
  This workstream removes only the unused sqlc-generated
  `go_server/internal/repo/sql/` tree and sqlc tooling.

## Verified Baseline and Recommendation

Evidence verified on 2026-10-26:

- `make go-test` passes. Its package list includes the unsupported `cmd/server_grpc`, `grpc`, `internal/repo/sql/*`, `pkg/pb`, and sampler packages, which proves they compile but not that production uses them.
- `make front-test` passes with 8 files and 24 tests.
- `go list -deps ./cmd/server ./cmd/scrape` includes `comics/pkg/pb` only because `api/route/oauth2.go` imports the `sampler` package and every file in that package is compiled. It does not include `comics/grpc`.
- Outside the gRPC/repository definitions, protobuf imports are confined to the unused sampler comic/user generators and serializers.
- No React source imports `src/frontend/pb`; only TypeScript/Vite aliases and the generator target refer to it.
- No production package imports the sqlc-generated `internal/repo/sql/sqlite` or `internal/repo/sql/postgres` packages.
- Root Docker and CI build `./cmd/server`; neither builds nor starts `./cmd/server_grpc`.
- The gRPC server loads TLS certificates from hard-coded dummy paths in
  `go_server/grpc/config.go`, uses a PostgreSQL-only `ComicsRepo`, and bypasses
  the production `ComicService`.
- GitHub code search under `Estebmaister` finds protobuf/gRPC symbols only in `Estebmaister/comics`.
- `db/comics.db` contains a legacy `users` table with zero rows.
- The repository has no tracked Python source, Python package manifest, generated Python protobuf package, or Python test suite.
- `db/comics.json` is tracked, approximately 15 MiB, and has no runtime or Make backup/restore consumer.

**Recommendation:** choose retirement. Git history is the archive for the removed protobuf definitions; it is not an active contract. Retaining the source without a supported server would continue the current ambiguity.

## Decision Options

1. **Retire the unsupported surface — recommended.** When Gate 0 finds no
   consumer or deployed endpoint, remove the gRPC binary, divergent repository,
   protobuf sources and generated clients, sqlc output, stale configuration,
   and ambiguous data artifacts in Tasks 1–6.
2. **Preserve temporarily and converge.** If Gate 0 identifies a named active
   consumer, do not execute Task 1. Keep the protobuf source and generated
   bindings only while a separately approved plan gives that consumer an owner,
   routes RPCs through the production use-case/repository/auth path, adds
   deployment and contract tests, and commits to either a dated REST cutover or
   long-term versioned support. The divergent repository cannot remain the
   implementation.

## Gate 0: Prove Whether a Consumer Exists

This gate is read-only and has no commit. Run it immediately before Task 1, even if the commands were run during planning.

- [ ] **Step 1: Record the exact revision and baseline health**

Run:

```bash
git rev-parse HEAD
make go-test
make front-test
```

Expected:

- `git rev-parse HEAD` prints one 40-character commit SHA.
- Go exits `0`.
- Vitest exits `0` and reports all test files passing.

- [ ] **Step 2: Trace production binaries and in-repository Go consumers**

Run:

```bash
cd go_server
go list -deps ./cmd/server ./cmd/scrape \
  | rg '^comics/(grpc|grpc/middleware|pkg/pb)$'
cd ..

git grep -n 'comics/pkg/pb' -- \
  'go_server/**/*.go' \
  ':!go_server/grpc/**' \
  ':!go_server/internal/repo/comics_repo.go' \
  ':!go_server/pkg/pb/**'

git grep -n 'comics/internal/repo/sql/' -- \
  'go_server/**/*.go' \
  ':!go_server/internal/repo/sql/**' || true
```

Expected:

- The dependency query prints `comics/pkg/pb` and no `comics/grpc` package.
- The protobuf import query prints only sampler generator/serializer files and their tests.
- The sqlc import query prints nothing.

Interpretation: protobuf is linked accidentally through package-level compilation, not consumed as the production API contract.

- [ ] **Step 3: Prove the React app has no protobuf consumer**

Run:

```bash
if git grep -nE \
  "from ['\"](@pb|.*frontend/pb)|import\\(['\"](@pb|.*frontend/pb)" -- \
  'src/frontend/**' ':!src/frontend/pb/**'; then
  exit 1
else
  echo "frontend protobuf imports: none"
fi
```

Expected:

```text
frontend protobuf imports: none
```

- [ ] **Step 4: Inventory build and deployment entrypoints**

Run:

```bash
git grep -nE 'cmd/server_grpc|server_grpc|GRPC_(ADDRESS|PORT)|50051|8082' -- \
  Dockerfile go_server/Dockerfile Makefile go_server/Makefile \
  docker-compose.yaml go_server/docker-compose.yaml \
  .github/workflows go_server/local.env \
  ':!go_server/cmd/server_grpc/**' ':!go_server/grpc/**' || true

git grep -nE 'go build .*cmd/server|CMD \["\./server"\]' -- \
  Dockerfile go_server/Dockerfile .github/workflows
```

Expected:

- The first query shows only phantom `GRPC_ADDRESS`, `GRPC_PORT`, and `8082` Compose/env configuration; it shows no build or launch of `cmd/server_grpc`.
- The second query shows the root image and CI building `cmd/server`, and Docker running `./server`.

- [ ] **Step 5: Search every repository visible under the owner account**

Run:

```bash
for query in 'ComicService' 'comics_service.proto' 'comics/pkg/pb'; do
  gh search code "$query" \
    --owner Estebmaister \
    --limit 100 \
    --json repository,path,url \
    | jq -r '.[].repository.nameWithOwner' \
    | sort -u
done
```

Expected: each non-empty result set contains only:

```text
Estebmaister/comics
```

Also verify that every private client/deployment repository is visible to the authenticated `gh` identity:

```bash
gh auth status
gh repo list Estebmaister --limit 100 --json nameWithOwner,isPrivate \
  | jq -e 'length > 0'
```

Expected: authentication succeeds and `jq` prints `true`.

- [ ] **Step 6: Verify the deployed process, image, and ingress**

On the SSH/Docker production host described by `.github/workflows/ci-cd.yml`, run:

```bash
cd /home/dev/comics
docker compose ps --format json \
  | jq -r '[.Service, .Image, .Command, .Publishers] | @tsv'
docker compose exec -T comics_server sh -c \
  'test ! -e /app/server_grpc && echo "server_grpc binary: absent"'
docker compose exec -T comics_server sh -c \
  'test -x /app/server && echo "REST server binary: present"'
```

Expected:

```text
server_grpc binary: absent
REST server binary: present
```

The `comics_server` command must be `./server`. Render/Koyeb/Heroku settings must likewise point at the root Dockerfile or `cmd/server`, with no separate gRPC service, DNS name, health check, or ingress rule.

- [ ] **Step 7: Verify the data and Python-era baselines**

Run:

```bash
sqlite3 -readonly db/comics.db \
  "SELECT COUNT(*) FROM users;"

git ls-files '*.py' 'requirements*.txt' pyproject.toml setup.py setup.cfg \
  'Pipfile*' poetry.lock 'src/pb/**'

git grep -nE 'db/comics\.json|COMICS_JSON' -- \
  ':!db/comics.json' \
  ':!changelog/**' \
  ':!docs/superpowers/specs/**' || true
```

Expected:

- SQLite prints `0`.
- The Python file inventory prints nothing.
- The JSON query shows only active documentation/configuration ambiguity, not a runtime reader.

- [ ] **Step 8: Apply the decision rule**

The contract-retirement decision passes only when all of the following are true:

- No repository other than `Estebmaister/comics` imports or vendors this protobuf contract.
- No mobile, web, CLI, automation, or partner client uses a gRPC endpoint.
- No deployed process contains `server_grpc`.
- No ingress, DNS record, service definition, or monitoring target routes traffic to application gRPC.

If these conditions hold, retirement is authorized. The separate data
precondition for Task 4 is that the legacy SQLite `users` table has zero rows.
Execute Tasks 1–6 in order only when both decisions pass.

If any contract-retirement condition fails, stop before Task 1. Do not delete
`proto/`, `go_server/pkg/pb/`, `go_server/grpc/`,
`go_server/cmd/server_grpc/`, or `go_server/internal/repo/comics_repo.go`.
Preserve protobuf source only while a separate parity-and-migration plan does
all of the following:

- names the consumer repository, owner, endpoint, and last observed use;
- routes RPCs through `domain.ComicUseCase` and `domain.ComicRepository`;
- applies the same normalization, discovery, transaction, merge, and admin-auth rules as REST;
- adds gRPC contract tests and a supported deployment target;
- assigns generated-client ownership and a versioning policy;
- defines a dated REST cutover or an explicit long-term support commitment.

The divergent `ComicsRepo` is not an acceptable preservation path.

If the consumer decision passes but the legacy table contains rows, Tasks 1–3
may proceed. Stop before Task 4 and create a data-migration plan that proves
those rows exist in MongoDB before resuming Tasks 4–6.

---

## Task 1: Retire the Unsupported gRPC and Protobuf Contract

**Files:**

- Delete: `go_server/cmd/server_grpc/main.go`
- Delete: `go_server/grpc/comics_service.go`
- Delete: `go_server/grpc/config.go`
- Delete: `go_server/grpc/metrics.go`
- Delete: `go_server/grpc/middleware.go`
- Delete: `go_server/grpc/middleware/logging.go`
- Delete: `go_server/grpc/service.go`
- Delete: `go_server/grpc/validation.go`
- Delete: `go_server/internal/repo/comics_repo.go`
- Delete: `go_server/pkg/pb/auth.pb.go`
- Delete: `go_server/pkg/pb/auth.pb.validate.go`
- Delete: `go_server/pkg/pb/auth_grpc.pb.go`
- Delete: `go_server/pkg/pb/comics.pb.go`
- Delete: `go_server/pkg/pb/comics.pb.validate.go`
- Delete: `go_server/pkg/pb/comics_service.pb.go`
- Delete: `go_server/pkg/pb/comics_service.pb.validate.go`
- Delete: `go_server/pkg/pb/comics_service_grpc.pb.go`
- Delete: `go_server/pkg/pb/staticcheck.conf`
- Delete: `proto/README.md`
- Delete: `proto/auth.proto`
- Delete: `proto/comics.proto`
- Delete: `proto/comics_service.proto`
- Delete: `proto/validate/validate.proto`
- Delete: `src/frontend/pb/comics.ts`
- Delete: `src/frontend/pb/google/protobuf/timestamp.ts`
- Delete: `go_server/sampler/comic_generator.go`
- Delete: `go_server/sampler/comic_generator_test.go`
- Delete: `go_server/sampler/serializer.go`
- Delete: `go_server/sampler/serializer_test.go`
- Delete: `go_server/sampler/user_generator.go`
- Delete: `go_server/sampler/user_generator_test.go`
- Preserve: `go_server/sampler/random_generator.go`
- Preserve: `go_server/sampler/random_generator_test.go`
- Modify: `Makefile`
- Modify: `package.json`
- Modify: `package-lock.json`
- Modify: `tsconfig.json`
- Modify: `vite.config.mts`
- Modify: `.githooks/common.sh`
- Modify: `go_server/.golangci.yml`
- Modify: `go_server/Makefile`
- Modify: `go_server/go.mod`
- Modify: `go_server/go.sum`
- Modify: `go_server/tool.go.mod`
- Modify: `go_server/tool.go.sum`

**Interfaces:**

- Consumes: Gate 0 retirement authorization.
- Produces: production Go binaries and the React app with no application protobuf/gRPC package dependency.
- Preserves: OTLP tracing over gRPC and `sampler.RandomString()` used by the current OAuth route.

- [ ] **Step 1: Run the retirement assertions and verify they fail before deletion**

Run:

```bash
test ! -e go_server/cmd/server_grpc
test ! -e go_server/grpc
test ! -e go_server/pkg/pb
test ! -e proto
test ! -e src/frontend/pb
```

Expected: each assertion exits `1` because the legacy path still exists.

- [ ] **Step 2: Remove the accidental production protobuf dependency**

Delete the six protobuf-dependent sampler implementation/test files listed above. Keep `random_generator.go` and its test because `api/route/oauth2.go` still imports `sampler.RandomString()`.

Run:

```bash
cd go_server
go list -deps ./cmd/server ./cmd/scrape \
  | rg '^comics/(grpc|grpc/middleware|pkg/pb)$' || true
cd ..
```

Expected: no output.

- [ ] **Step 3: Remove protobuf generation from onboarding and cleanup**

In the root `Makefile`:

- remove `proto-go` and `proto-js` from `.PHONY`;
- change `setup` to `setup-go setup-front setup-hooks`;
- change the setup description to `Onboarding: Go, frontend, git hooks`;
- delete `PROTO_DIR`, `PROTO_FILES`, `JS_OUT`, `GO_OUT`, `GO_PROTO_OUT`, `GO_SERVICE_OUT`, and `PROTOC`;
- delete both protobuf recipes and the protobuf help section;
- replace the current `clean` recipe with:

```make
## @util:clean                  Remove frontend and Go build artifacts
clean:
	rm -rf build
	$(MAKE) -C go_server clean
```

In `.githooks/common.sh`, remove only these exclusions:

```sh
':!go_server/pkg/pb/*'
':!src/pb/*'
':!src/frontend/pb/*'
```

In `go_server/.golangci.yml`, remove the `issues.exclude-dirs` entry for `pkg/pb`.

In `go_server/Makefile`, change the security scan to:

```make
	$(TOOL_GO) gosec -quiet ./...
```

- [ ] **Step 4: Remove the unused TypeScript generator and aliases**

Run:

```bash
npm uninstall ts-proto --ignore-scripts
```

Expected: `package.json` and `package-lock.json` no longer contain a top-level `ts-proto` dependency.

Delete the entire `paths` block from `tsconfig.json`; no aliases remain there.

Delete only the `@pb` entry from `vite.config.mts`, leaving `@config` intact:

```ts
resolve: {
  alias: {
    '@config': path.resolve(rootDir, 'config'),
  },
},
```

- [ ] **Step 5: Delete gRPC, protobuf source, and generated bindings**

Delete every gRPC, protobuf, generated Go binding, generated frontend binding, and divergent `comics_repo.go` file listed in this task.

Do not copy `.proto` files into another active directory. The preceding gate and Git history are the preservation mechanism.

- [ ] **Step 6: Reconcile Go dependencies**

Run:

```bash
cd go_server
go mod tidy
go mod tidy -modfile=tool.go.mod
go mod verify
cd ..
```

Expected:

- `github.com/envoyproxy/protoc-gen-validate` is absent from `go.mod`.
- `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` is absent from `go.mod`.
- `google.golang.org/grpc` and `google.golang.org/protobuf` may remain indirect dependencies because the supported OTLP trace exporter uses them.
- `go mod verify` prints `all modules verified`.

- [ ] **Step 7: Verify no contract code or generated client remains**

Run:

```bash
for path in \
  go_server/cmd/server_grpc \
  go_server/grpc \
  go_server/pkg/pb \
  proto \
  src/frontend/pb \
  go_server/internal/repo/comics_repo.go; do
  test ! -e "$path"
done

cd go_server
go list -deps ./cmd/server ./cmd/scrape \
  | rg '^comics/(grpc|grpc/middleware|pkg/pb)$' || true
go test ./...
go vet ./...
cd ..

npm ci
npm test
npm run build
```

Expected:

- All path checks pass.
- The dependency query prints nothing.
- Go test/vet exit `0`.
- Vitest and the TypeScript/Vite build exit `0`.

- [ ] **Step 8: Commit the contract retirement**

```bash
git add \
  Makefile package.json package-lock.json tsconfig.json vite.config.mts \
  .githooks/common.sh go_server/.golangci.yml go_server/Makefile \
  go_server/go.mod go_server/go.sum go_server/tool.go.mod go_server/tool.go.sum \
  go_server/cmd/server_grpc go_server/grpc go_server/internal/repo/comics_repo.go \
  go_server/pkg/pb proto src/frontend/pb \
  go_server/sampler/comic_generator.go \
  go_server/sampler/comic_generator_test.go \
  go_server/sampler/serializer.go \
  go_server/sampler/serializer_test.go \
  go_server/sampler/user_generator.go \
  go_server/sampler/user_generator_test.go
git commit -m "chore: retire unsupported grpc contract"
```

Expected: one commit that removes the unsupported contract while all supported builds pass.

Rollback for this task:

```bash
git revert --no-edit "$(
  git log -1 --format=%H --fixed-strings \
    --grep='chore: retire unsupported grpc contract'
)"
```

Expected: the removed source returns. This is source rollback only; it does not make gRPC a supported or deployed contract.

---

## Task 2: Remove sqlc and Divergent Schema/Migration Paths

**Files:**

- Delete: `go_server/sqlc.yml`
- Delete: `go_server/internal/repo/sql/migrations/001_initial_schema.sql`
- Delete: `go_server/internal/repo/sql/migrations/002_add_triggers.sql`
- Delete: `go_server/internal/repo/sql/migrations/003_add_cover_visible.sql`
- Delete: `go_server/internal/repo/sql/migrations/004_comic_title_keys.sql`
- Delete: `go_server/internal/repo/sql/migrations/005_drop_comics_identity_key.sql`
- Delete: `go_server/internal/repo/sql/postgres/comics_postgre.sql.go`
- Delete: `go_server/internal/repo/sql/postgres/db.go`
- Delete: `go_server/internal/repo/sql/postgres/models.go`
- Delete: `go_server/internal/repo/sql/query/comics_normalized.sql`
- Delete: `go_server/internal/repo/sql/query/comics_postgre.sql`
- Delete: `go_server/internal/repo/sql/query/comics_sqlite.sql`
- Delete: `go_server/internal/repo/sql/query/create_normalized.go`
- Delete: `go_server/internal/repo/sql/schema/comics_normalized.erd.json`
- Delete: `go_server/internal/repo/sql/schema/comics_postgre.sql`
- Delete: `go_server/internal/repo/sql/schema/comics_sqlite.sql`
- Delete: `go_server/internal/repo/sql/sqlite/comics_sqlite.sql.go`
- Delete: `go_server/internal/repo/sql/sqlite/db.go`
- Delete: `go_server/internal/repo/sql/sqlite/models.go`
- Delete: `go_server/internal/repo/sql/sqlite/user_repo.go`
- Modify: `Makefile`
- Modify: `go_server/Makefile`
- Modify: `go_server/tool.go.mod`
- Modify: `go_server/tool.go.sum`
- Modify: `go_server/go.mod`
- Modify: `go_server/go.sum`
- Modify: `go_server/bootstrap/app.go`
- Modify: `go_server/bootstrap/db.go`
- Modify: `go_server/bootstrap/env.go`
- Modify: `go_server/internal/repo/db.go`
- Modify: `go_server/local.env`

**Interfaces:**

- Consumes: `comicrepo.NewSQLiteComicRepository` and `comicrepo.NewPostgresComicRepository` as the supported catalog adapters.
- Produces: one catalog repository path under `internal/repo/comics`, backed by Workstream 3's canonical `internal/repo/migrations` package and migration CLI.
- Preserves: MongoDB `DBConfig`, SQLite default, and optional PostgreSQL.

- [ ] **Step 1: Prove generated repositories are not consumers and the canonical migration path exists**

Run:

```bash
git grep -n 'comics/internal/repo/sql/' -- \
  'go_server/**/*.go' ':!go_server/internal/repo/sql/**' || true

test -d go_server/internal/repo/migrations
test -f go_server/cmd/migrate/main.go
```

Expected:

- The import search prints nothing.
- Both canonical migration assertions exit `0`; they protect Workstream 3's
  replacement from this cleanup.

- [ ] **Step 2: Simplify user-repository bootstrap**

Replace `go_server/bootstrap/db.go` with:

```go
package bootstrap

import (
	"context"

	"comics/internal/repo/mongo"
)

func newUserRepo(ctx context.Context, env *Env) (ClosableUserStore, error) {
	return mongo.NewUserRepo(ctx, env.DBConfig, env.TracerConfig)
}
```

In `go_server/bootstrap/app.go`, replace:

```go
userRepo, err := newRepo(ctx, env, mongoUserRepoType)
```

with:

```go
userRepo, err := newUserRepo(ctx, env)
```

This removes commented selectors that imply SQLite/PostgreSQL user stores or Mongo comic stores are supported.

- [ ] **Step 3: Remove DB configuration fields owned only by deleted code**

In `go_server/internal/repo/db.go`, make the connection/collection portion of `DBConfig`:

```go
type DBConfig struct {
	Addr       string `mapstructure:"DB_ADDR" default:""`
	User       string `mapstructure:"DB_USER" default:"user"`
	Pass       string `mapstructure:"DB_PASS" default:"pass"`
	Name       string `mapstructure:"DB_NAME" default:"comics"`
	TableUsers string `mapstructure:"DB_TABLE_USERS" default:"users"`

	MaxPoolSize     uint          `mapstructure:"DB_MAX_POOL_SIZE" default:"100"`
	MinPoolSize     uint          `mapstructure:"DB_MIN_POOL_SIZE" default:"0"`
	MaxConnIdleTime time.Duration `mapstructure:"DB_MAX_CONN_IDLE_TIME" default:"5m"`
	MaxConnLifeTime time.Duration `mapstructure:"DB_MAX_CONN_LIFE_TIME" default:"60m"`
	ConnectTimeout  time.Duration `mapstructure:"DB_CONN_TIMEOUT" default:"30s"`
	BackoffTimeout  time.Duration `mapstructure:"DB_BACKOFF_TIMEOUT" default:"15s"`
}
```

Remove `DB_HOST`, `DB_PORT`, and `DB_TABLE_COMICS` from `bindEnvKeys()` in `go_server/bootstrap/env.go`. Remove `DB_TABLE_COMICS` from `go_server/local.env`.

- [ ] **Step 4: Remove dead sqlc tooling**

Keep `go-migrate-up`, `go-migrate-down`, and their `.PHONY` entries in both
Makefiles. They invoke Workstream 3's canonical migration CLI.

In `go_server/Makefile`:

- remove `sqlc` from `.PHONY`;
- change the setup-tools help text to `Installs air, delve, swag, linters, and scanners into module cache`;
- delete the `sqlc` recipe.

Remove this tool directive from `go_server/tool.go.mod`:

```go
github.com/sqlc-dev/sqlc/cmd/sqlc
```

- [ ] **Step 5: Delete generated SQL and duplicate schemas**

Delete `go_server/sqlc.yml` and the entire `go_server/internal/repo/sql/` tree
listed above. Preserve `go_server/cmd/migrate` and
`go_server/internal/repo/migrations`.

The supported repository remains:

```text
go_server/internal/repo/comics/sql_repository.go
go_server/internal/repo/comics/title_keys.go
```

Do not delete `db/db_classes.json`; React imports it.

- [ ] **Step 6: Tidy modules and verify the supported repositories**

Run:

```bash
cd go_server
go mod tidy
go mod tidy -modfile=tool.go.mod
go mod verify
go test ./internal/repo/comics/... ./bootstrap/... ./api/route/...
go test ./...
go vet ./...
cd ..

make help
make -C go_server help
```

Expected:

- `github.com/golang-migrate/migrate/v4` remains in `go_server/go.mod`.
- `github.com/sqlc-dev/sqlc/cmd/sqlc` is absent from the `tool` block.
- Tests and vet exit `0`.
- Both help outputs still list `go-migrate-up` and `go-migrate-down`; neither
  lists `sqlc`.

- [ ] **Step 7: Commit repository consolidation**

```bash
git add \
  Makefile go_server/Makefile go_server/tool.go.mod go_server/tool.go.sum \
  go_server/go.mod go_server/go.sum go_server/bootstrap/app.go \
  go_server/bootstrap/db.go go_server/bootstrap/env.go \
  go_server/internal/repo/db.go go_server/local.env \
  go_server/sqlc.yml go_server/internal/repo/sql
git commit -m "chore: remove unused sql repositories"
```

Rollback:

```bash
git revert --no-edit "$(
  git log -1 --format=%H --fixed-strings \
    --grep='chore: remove unused sql repositories'
)"
```

Expected: the deleted generated packages and dead targets return.

---

## Task 3: Remove Phantom gRPC and Python API Configuration

**Files:**

- Modify: `go_server/bootstrap/env.go`
- Modify: `go_server/local.env`
- Modify: `local.env`
- Modify: `docker-compose.yaml`
- Modify: `go_server/docker-compose.yaml`
- Modify: `go_server/.observability/docker-compose.yaml`
- Modify: `src/frontend/util/Config.ts`
- Modify: `src/frontend/util/Config.test.ts`
- Modify: `src/frontend/global.d.ts`
- Modify: `src/frontend/components/Comics/Edition/ScrapeButton.test.tsx`

**Interfaces:**

- Consumes: `VITE_API_SERVER` as the sole explicit browser API override.
- Produces: local fallback `https://localhost:8081` and remote same-origin HTTPS fallback.
- Preserves: `OTEL_EXPORTER_GRPC_ENDPOINT` for trace export.

- [ ] **Step 1: Change the frontend tests first**

Replace `src/frontend/util/Config.test.ts` with:

```ts
import { describe, expect, test } from 'vitest';
import { resolveServerURL } from './Config';

describe('resolveServerURL', () => {
  test('prefers explicit VITE_API_SERVER', () => {
    expect(resolveServerURL({
      hostname: 'localhost',
      env: { VITE_API_SERVER: 'https://api.example.com' },
    })).toBe('https://api.example.com');
  });

  test('uses local Go server by default on localhost', () => {
    expect(resolveServerURL({
      hostname: 'localhost',
      env: {},
    })).toBe('https://localhost:8081');
  });

  test('uses same-origin HTTPS outside local development', () => {
    expect(resolveServerURL({
      hostname: 'comics.example.com',
      env: {},
    })).toBe('https://comics.example.com');
  });
});
```

Run:

```bash
npm test -- src/frontend/util/Config.test.ts
```

Expected: FAIL because the implementation still accepts `VITE_PY_SERVER` and returns port `5001` for a remote host.

- [ ] **Step 2: Remove the Python API fallback**

Replace the type and URL resolver in `src/frontend/util/Config.ts` with:

```ts
type ApiConfigEnv = {
  VITE_API_SERVER?: string;
};

type ResolveServerArgs = {
  hostname?: string;
  env?: ApiConfigEnv;
};

const localHosts = new Set(['localhost', '127.0.0.1', '::1']);

const cleanURL = (value?: string) => {
  const trimmed = value?.trim().replace(/\/+$/, '');
  return trimmed === '' ? undefined : trimmed;
};

export const resolveServerURL = ({
  hostname = window.location.hostname,
  env = import.meta.env,
}: ResolveServerArgs = {}) => (
  cleanURL(env.VITE_API_SERVER)
  ?? (localHosts.has(hostname) ? 'https://localhost:8081' : `https://${hostname}`)
);
```

Keep the existing exported `config` object below this code.

In `src/frontend/global.d.ts`, keep only the supported frontend variable:

```ts
interface ImportMetaEnv {
  readonly VITE_API_SERVER?: string;
}
```

Change the stale failure fixture in `ScrapeButton.test.tsx` to:

```ts
json: () => Promise.resolve({ message: 'scrape failed' }),
```

- [ ] **Step 3: Remove dead backend env fields**

From `go_server/bootstrap/env.go`, delete:

- `defaultGRPCAddr`;
- `defaultGRPCPort`;
- `AddressGRPC`;
- `PortGRPC`;
- `PythonBackendURL`;
- the `GRPC_ADDRESS`, `GRPC_PORT`, and `PY_BACKEND_URL` defaults/bindings.

Do not remove `OTEL_EXPORTER_GRPC_ENDPOINT`.

Rewrite root `local.env` as the frontend/dev-proxy template:

```dotenv
# Frontend
VITE_API_SERVER=/api

# Vite forwards /api to the Go REST server.
VITE_GO_API_TARGET=https://localhost:8081
```

From `go_server/local.env`, remove `GRPC_PORT`, `GRPC_ADDRESS`, the Python scraper-proxy comment, and `PY_BACKEND_URL`.

- [ ] **Step 4: Remove phantom application gRPC ports**

From both Compose files, remove:

```yaml
GRPC_ADDRESS: 0.0.0.0
GRPC_PORT: 8082
```

or their list-form equivalents, and remove the `8082:8082` port mapping.

From `go_server/.observability/docker-compose.yaml`, remove:

```yaml
environment:
  TARGETS: host.docker.internal:8082/metrics
```

The mounted Prometheus configuration already scrapes the supported REST process at `host.docker.internal:8081/metrics`.

- [ ] **Step 5: Run focused and full verification**

Run:

```bash
npm test -- src/frontend/util/Config.test.ts \
  src/frontend/components/Comics/Edition/ScrapeButton.test.tsx
npm test
npm run build

cd go_server
go test ./bootstrap/... ./api/route/...
go test ./...
cd ..

docker compose config > /tmp/comics-compose.yaml
if rg -n '8082|GRPC_(ADDRESS|PORT)|PY_BACKEND_URL' \
  /tmp/comics-compose.yaml go_server/local.env local.env; then
  exit 1
else
  echo "legacy runtime configuration: none"
fi
```

Expected:

- Focused and full frontend suites pass.
- Frontend build passes.
- Go tests pass.
- The final command prints `legacy runtime configuration: none`.

- [ ] **Step 6: Commit configuration cleanup**

```bash
git add \
  go_server/bootstrap/env.go go_server/local.env local.env \
  docker-compose.yaml go_server/docker-compose.yaml \
  go_server/.observability/docker-compose.yaml \
  src/frontend/util/Config.ts src/frontend/util/Config.test.ts \
  src/frontend/global.d.ts \
  src/frontend/components/Comics/Edition/ScrapeButton.test.tsx
git commit -m "refactor: remove legacy api fallbacks"
```

Rollback:

```bash
git revert --no-edit "$(
  git log -1 --format=%H --fixed-strings \
    --grep='refactor: remove legacy api fallbacks'
)"
```

---

## Task 4: Safely Remove the Empty Legacy SQLite Users Table

**Files:**

- Create: `go_server/internal/repo/comics/legacy_cleanup.go`
- Create: `go_server/internal/repo/comics/legacy_cleanup_test.go`
- Modify: `go_server/cmd/db_maint/main.go`
- Modify: `go_server/Makefile`
- Modify: `Makefile`
- Modify: `.gitignore`
- Modify data: `db/comics.db`

**Interfaces:**

- Produces: `comics.DropLegacyUsersTable(db *sql.DB) (bool, error)`.
- Safety contract: absent table is idempotent success; empty table is dropped transactionally; non-empty table is preserved and returns an exact error.
- Rollback input: verified timestamped SQLite backup under `db/backups/`.

- [ ] **Step 1: Write tests for absent, empty, and populated tables**

Create `go_server/internal/repo/comics/legacy_cleanup_test.go`:

```go
package comics

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func openLegacyCleanupDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestDropLegacyUsersTableAlreadyAbsent(t *testing.T) {
	db := openLegacyCleanupDB(t)
	dropped, err := DropLegacyUsersTable(db)
	if err != nil {
		t.Fatal(err)
	}
	if dropped {
		t.Fatal("absent users table reported as dropped")
	}
}

func TestDropLegacyUsersTableDropsEmptyTable(t *testing.T) {
	db := openLegacyCleanupDB(t)
	if _, err := db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	dropped, err := DropLegacyUsersTable(db)
	if err != nil {
		t.Fatal(err)
	}
	if !dropped {
		t.Fatal("empty users table was not dropped")
	}
	var count int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_schema
		WHERE type = 'table' AND name = 'users'
	`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("users table still exists: %d", count)
	}
}

func TestDropLegacyUsersTableRefusesPopulatedTable(t *testing.T) {
	db := openLegacyCleanupDB(t)
	if _, err := db.Exec(`
		CREATE TABLE users (id TEXT PRIMARY KEY);
		INSERT INTO users (id) VALUES ('legacy-user');
	`); err != nil {
		t.Fatal(err)
	}
	dropped, err := DropLegacyUsersTable(db)
	if dropped {
		t.Fatal("populated users table was dropped")
	}
	if err == nil || !strings.Contains(
		err.Error(),
		"legacy users table contains 1 row; migrate it to MongoDB before dropping",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("legacy row count changed: %d", count)
	}
}
```

Run:

```bash
cd go_server
go test ./internal/repo/comics -run DropLegacyUsers -count=1
```

Expected: FAIL with `undefined: DropLegacyUsersTable`.

- [ ] **Step 2: Implement the guarded transactional drop**

Create `go_server/internal/repo/comics/legacy_cleanup.go`:

```go
package comics

import (
	"database/sql"
	"fmt"
)

// DropLegacyUsersTable removes the obsolete SQLite users table only when empty.
func DropLegacyUsersTable(db *sql.DB) (bool, error) {
	var exists int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_schema
		WHERE type = 'table' AND name = 'users'
	`).Scan(&exists); err != nil {
		return false, fmt.Errorf("inspect legacy users table: %w", err)
	}
	if exists == 0 {
		return false, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return false, fmt.Errorf("begin legacy users cleanup: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return false, fmt.Errorf("count legacy users: %w", err)
	}
	if count != 0 {
		noun := "rows"
		if count == 1 {
			noun = "row"
		}
		return false, fmt.Errorf(
			"legacy users table contains %d %s; migrate it to MongoDB before dropping",
			count,
			noun,
		)
	}
	if _, err := tx.Exec(`DROP TABLE users`); err != nil {
		return false, fmt.Errorf("drop legacy users table: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit legacy users cleanup: %w", err)
	}
	return true, nil
}
```

Run:

```bash
cd go_server
go test ./internal/repo/comics -run DropLegacyUsers -count=1
```

Expected: PASS for all three tests.

- [ ] **Step 3: Add the explicit maintenance action**

Replace `go_server/cmd/db_maint/main.go` with:

```go
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	comicrepo "comics/internal/repo/comics"

	_ "modernc.org/sqlite"
)

func main() {
	dropIdentity := flag.Bool(
		"drop-legacy-identity-key",
		false,
		"drop comics.identity_key after comic_title_keys is populated",
	)
	dropUsers := flag.Bool(
		"drop-legacy-users",
		false,
		"drop the empty legacy SQLite users table; refuses when rows exist",
	)
	flag.Parse()

	if *dropIdentity == *dropUsers {
		fmt.Fprintln(
			os.Stderr,
			"usage: db_maint (--drop-legacy-identity-key | --drop-legacy-users)",
		)
		os.Exit(2)
	}

	path := firstNonEmpty(os.Getenv("COMICS_SQLITE_PATH"), "../db/comics.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	if *dropIdentity {
		var keyCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM comic_title_keys`).
			Scan(&keyCount); err != nil {
			fatal(err)
		}
		if keyCount == 0 {
			fatal(fmt.Errorf(
				"%s: comic_title_keys is empty; run the API once or backfill before dropping identity_key",
				path,
			))
		}
		start := time.Now()
		if err := comicrepo.DropLegacyIdentityKeyColumn(db); err != nil {
			fatal(err)
		}
		fmt.Printf(
			"dropped legacy identity_key on %s (%d discovery keys, %s)\n",
			path,
			keyCount,
			time.Since(start).Round(time.Millisecond),
		)
		return
	}

	dropped, err := comicrepo.DropLegacyUsersTable(db)
	if err != nil {
		fatal(err)
	}
	if dropped {
		fmt.Printf("dropped empty legacy users table on %s\n", path)
		return
	}
	fmt.Printf("legacy users table already absent on %s\n", path)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
```

Update `go_server/Makefile` help text:

```make
## @database:go-db-maint          Offline SQLite maintenance; run `make db-backup` first
##? go-db-maint  Actions: --drop-legacy-identity-key or --drop-legacy-users
```

Update the root Make help text for `go-db-maint` to name both actions and the backup prerequisite.

- [ ] **Step 4: Ignore generated backups and the retired JSON export**

Append to `.gitignore`:

```gitignore
# Generated catalog artifacts
/db/backups/*.db
/db/comics.json
```

`db/backups/.gitkeep` remains tracked.

- [ ] **Step 5: Back up and prove catalog counts before maintenance**

Stop local writers and capture the counts:

```bash
make dev-down

before_comics="$(sqlite3 -readonly db/comics.db 'SELECT COUNT(*) FROM comics;')"
before_keys="$(sqlite3 -readonly db/comics.db 'SELECT COUNT(*) FROM comic_title_keys;')"
legacy_users="$(sqlite3 -readonly db/comics.db 'SELECT COUNT(*) FROM users;')"
test "$legacy_users" = "0"

backup_output="$(make db-backup)"
printf '%s\n' "$backup_output"
backup="${backup_output#Backup: }"
[[ "$backup" =~ ^db/backups/comics_backup_[0-9]{8}_[0-9]{6}\.db$ ]]
test -f "$backup"
sqlite3 -readonly "$backup" 'PRAGMA integrity_check;'
mkdir -p .dev
printf '%s\n' "$backup" > .dev/contract-cleanup-backup
```

Expected:

- `legacy_users` is `0`.
- The backup path passes the timestamped-path regular-expression assertion.
- Backup integrity prints `ok`.

- [ ] **Step 6: Drop the table and prove no catalog data changed**

Run:

```bash
make go-db-maint ARGS="--drop-legacy-users"

after_comics="$(sqlite3 -readonly db/comics.db 'SELECT COUNT(*) FROM comics;')"
after_keys="$(sqlite3 -readonly db/comics.db 'SELECT COUNT(*) FROM comic_title_keys;')"
users_tables="$(
  sqlite3 -readonly db/comics.db \
    "SELECT COUNT(*) FROM sqlite_schema WHERE type='table' AND name='users';"
)"

test "$before_comics" = "$after_comics"
test "$before_keys" = "$after_keys"
test "$users_tables" = "0"
make db-check
printf 'catalog counts unchanged: comics=%s keys=%s\n' \
  "$after_comics" "$after_keys"
```

Expected:

```text
dropped empty legacy users table on ../db/comics.db
db/comics.db: ok
```

The final `catalog counts unchanged` line must print the two integer values
captured before maintenance.

- [ ] **Step 7: Run the complete Go suite and commit**

Run:

```bash
cd go_server
go test ./cmd/db_maint ./internal/repo/comics -count=1
go test ./...
go vet ./...
cd ..
```

Expected: all commands exit `0`.

Commit:

```bash
git add \
  .gitignore Makefile go_server/Makefile go_server/cmd/db_maint/main.go \
  go_server/internal/repo/comics/legacy_cleanup.go \
  go_server/internal/repo/comics/legacy_cleanup_test.go \
  db/comics.db
git commit -m "chore: remove empty legacy sqlite users"
```

Operational rollback for the table:

```bash
make dev-down
backup="$(cat .dev/contract-cleanup-backup)"
test -n "$backup"
test -f "$backup"
make db-restore BACKUP="$backup"
make db-check
```

Expected: restore reports the chosen backup and integrity prints `ok`.

Source rollback:

```bash
git revert --no-edit "$(
  git log -1 --format=%H --fixed-strings \
    --grep='chore: remove empty legacy sqlite users'
)"
```

---

## Task 5: Remove JSON Catalog and Python-Era Documentation Ambiguity

**Files:**

- Delete: `db/comics.json`
- Create: `docs/contracts.md`
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `go_server/README.md`
- Modify: `docs/deploy-render.md`
- Modify: `tls/README.md`
- Modify: `Dockerfile`
- Modify: `Procfile`
- Modify: `Makefile`
- Modify: `tsconfig.json`
- Modify: `.gitignore`

**Interfaces:**

- Produces: active documentation that names REST JSON, SQLite, MongoDB, and optional PostgreSQL as the supported architecture.
- Preserves: `db/comics.db`, timestamped `.db` backups, historical specs/changelogs, and `db/db_classes.json`.

- [ ] **Step 1: Remove the unused JSON catalog and Make variable**

Run:

```bash
git rm db/comics.json
```

Remove this line from the root `Makefile`:

```make
COMICS_JSON ?= db/comics.json
```

Do not add a JSON restore/export recipe. `db-backup`, `db-restore`, and `db-check` continue to operate on SQLite `.db` files.

- [ ] **Step 2: Create the supported-contract record**

Create `docs/contracts.md`:

```markdown
# Supported Application Contracts

## Production API

The supported browser/service contract is the Go Gin REST JSON API served by
`go_server/cmd/server`. The React application uses `VITE_API_SERVER` when set,
`https://localhost:8081` for local fallback, and same-origin HTTPS for a
self-hosted production fallback.

Catalog reads and writes flow through `domain.ComicUseCase` and
`domain.ComicRepository`. SQLite is the default catalog store, PostgreSQL is
optional, and MongoDB owns users and authentication.

## Retired Contracts

The standalone application gRPC server, protobuf-generated clients, divergent
PostgreSQL repository, sqlc-generated repositories, and duplicate sqlc
schema/migration path were retired on 2026-10-26 after repository, build,
deployment, account-wide code-search, and database-consumer checks found no
supported consumer. The canonical `cmd/migrate` and
`internal/repo/migrations` path remains supported.

Git history preserves the retired definitions for archaeology. Files recovered
from history are not a supported contract. Reintroducing protobuf requires a
named consumer and owner, use-case/repository/auth parity, contract tests,
deployment ownership, and a versioning policy.

OTLP trace export may still use gRPC internally. That transport is observability
infrastructure, not the application API.

## Data Ownership

`db/comics.db` is the checked-in local SQLite catalog. Runtime backup and
restore use timestamped SQLite `.db` files under `db/backups/`; generated
backups are ignored by Git. The removed JSON snapshot is not a backup or
runtime source.

`db/db_classes.json` remains a frontend enum-label resource. It is not a
database schema or catalog backup.
```

- [ ] **Step 3: Correct root setup, backup, and architecture documentation**

Update `README.md` so:

- quick start remains `make setup`,
  `cp go_server/local.env go_server/.env`, `make dev-up`, and `make check`;
- setup is described as Go + frontend + git hooks;
- all virtualenv, pip, Python test, and protobuf setup commands are removed;
- testing keeps the supported layered targets: `make go-test`,
  `make contract-test`, `make front-test`, `make check-go`,
  `make check-front`, and `make check`; Workstream 9 later adds
  `make check-ci` as the mandatory CI/release aggregate;
- SQLite `db/comics.db` is canonical;
- backup/restore examples capture the path printed by `make db-backup` and pass
  that exact path as `make db-restore BACKUP="$backup"`;
- frontend env documents only `VITE_API_SERVER`;
- deployment text says Go Docker/Go service, without migration-from-Python wording;
- `docs/contracts.md` is linked from the architecture section.

Update `CLAUDE.md` so:

- every documented Make target exists after Tasks 1–2;
- no Python environment, Flask, SQLAlchemy, generated Python binding,
  protobuf generation, gRPC implementation, or sqlc target is presented as
  active; canonical migration targets remain documented;
- data flow ends in SQLite/PostgreSQL and does not claim JSON regeneration;
- backup/restore is described as SQLite `.db` only;
- frontend config names only `VITE_API_SERVER`;
- file structure omits removed `proto/`, `pkg/pb/`, `src/frontend/pb/`, and Python paths;
- catalog repository guidance points to `go_server/internal/repo/comics/`;
- `db/db_classes.json` is described only as frontend enum labels.

- [ ] **Step 4: Correct Go, Render, TLS, and image documentation**

Update `go_server/README.md` to remove:

- comparisons to the Python scraper;
- “Python-compatible” schema wording;
- obsolete sqlc generation commands.

Keep SQLite default, PostgreSQL optional, MongoDB auth, REST routes, scrape commands, and supported checks.

Rewrite the title and introduction of `docs/deploy-render.md` as Go deployment documentation. Remove:

- Python build/start migration instructions;
- `PY_BACKEND_URL`;
- the `requirements.txt` troubleshooting row.

In `tls/README.md`, describe only the frontend and Go backend, and tell readers to restart those two processes after certificate changes.

Change the first Dockerfile comment to:

```dockerfile
# Production image: Go REST API and native scraper.
```

Replace `Procfile` with:

```text
# Heroku: use the root Dockerfile or the Go buildpack.
# The supported process is go_server/cmd/server compiled as the web binary.
```

- [ ] **Step 5: Remove obsolete Python-specific ignore boilerplate**

Delete the Python packaging/framework block at the beginning of `.gitignore`.
Move the required root `.env` ignore into the remaining miscellaneous section:

```gitignore
.env
```

Remove `comics_env` from the `exclude` array in `tsconfig.json`. Do not retain
virtualenv, bytecode, Flask, pytest, packaging, or notebook ignores for a
runtime that no longer exists.

- [ ] **Step 6: Audit active documentation and configuration**

Run:

```bash
if git grep -nE \
  'VITE_PY_SERVER|VITE_REMOTE_PY_SERVER|PY_BACKEND_URL|COMICS_JSON|comics_env|setup-venv|proto-(py|js|go)|py-test|SQLAlchemy|Flask|sqlc generate|db/comics\.json' -- \
  README.md CLAUDE.md go_server/README.md docs/deploy-render.md \
  tls/README.md Dockerfile Procfile Makefile go_server/Makefile \
  local.env go_server/local.env src/frontend; then
  exit 1
else
  echo "active legacy references: none"
fi

test ! -e db/comics.json
test -e db/comics.db
test -e db/db_classes.json
make db-check
```

Expected:

```text
active legacy references: none
db/comics.db: ok
```

- [ ] **Step 7: Commit documentation and catalog cleanup**

```bash
git add \
  db/comics.json docs/contracts.md README.md CLAUDE.md go_server/README.md \
  docs/deploy-render.md tls/README.md Dockerfile Procfile Makefile \
  tsconfig.json .gitignore
git commit -m "docs: define the supported rest architecture"
```

Rollback:

```bash
git revert --no-edit "$(
  git log -1 --format=%H --fixed-strings \
    --grep='docs: define the supported rest architecture'
)"
```

Expected: documentation and the JSON snapshot return to their prior revision.

---

## Task 6: Add a Permanent Legacy-Contract Regression Gate

**Files:**

- Create: `scripts/check-contract-cleanup.sh`
- Modify: `Makefile`
- Modify: `.github/workflows/ci.yml`

**Interfaces:**

- Produces: `make contract-cleanup-check`.
- Enforces: retired paths and active configuration names cannot be reintroduced silently.
- Excludes: historical specs, changelogs, and implementation plans.

- [ ] **Step 1: Add the static cleanup check**

Create `scripts/check-contract-cleanup.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

legacy_paths=(
  "db/comics.json"
  "go_server/cmd/server_grpc"
  "go_server/grpc"
  "go_server/internal/repo/comics_repo.go"
  "go_server/internal/repo/sql"
  "go_server/pkg/pb"
  "go_server/sqlc.yml"
  "proto"
  "src/frontend/pb"
)

failed=0
for path in "${legacy_paths[@]}"; do
  if [[ -e "$path" ]]; then
    echo "legacy path present: $path"
    failed=1
  fi
done

required_paths=(
  "go_server/cmd/migrate/main.go"
  "go_server/internal/repo/migrations"
)
for path in "${required_paths[@]}"; do
  if [[ ! -e "$path" ]]; then
    echo "canonical migration path missing: $path"
    failed=1
  fi
done

pattern='VITE_(PY|REMOTE_PY)_SERVER|PY_BACKEND_URL|COMICS_JSON|comics_env|GRPC_(ADDRESS|PORT)|@pb/|proto-(go|js|py)|sqlc generate|db/comics\.json'
matches="$(
  git grep -nE "$pattern" -- \
    ':!.gitignore' \
    ':!changelog/**' \
    ':!docs/superpowers/specs/**' \
    ':!docs/superpowers/plans/**' \
    ':!scripts/check-contract-cleanup.sh' \
    || true
)"
if [[ -n "$matches" ]]; then
  printf '%s\n' "$matches"
  failed=1
fi

if (( failed != 0 )); then
  echo "contract cleanup check: failed"
  exit 1
fi

echo "contract cleanup check: ok"
```

Run:

```bash
chmod +x scripts/check-contract-cleanup.sh
./scripts/check-contract-cleanup.sh
```

Expected:

```text
contract cleanup check: ok
```

- [ ] **Step 2: Wire the check into Make and CI**

Add `contract-cleanup-check` to root `.PHONY` and add:

```make
## @quality:contract-cleanup-check  Reject retired contract/config artifacts
contract-cleanup-check:
	@./scripts/check-contract-cleanup.sh
```

In the root `check` recipe, run it before `check-go`:

```make
	echo "== contract-cleanup-check =="; \
	$(MAKE) contract-cleanup-check || failed=1; \
	echo ""; \
```

In `.github/workflows/ci.yml`, add this root-level step to the frontend job immediately after checkout:

```yaml
      - run: ./scripts/check-contract-cleanup.sh
```

- [ ] **Step 3: Run final static, test, build, and container verification**

Run:

```bash
./scripts/check-contract-cleanup.sh
make check

docker compose config > /tmp/comics-compose-final.yaml
if rg -n '8082|GRPC_(ADDRESS|PORT)|PY_BACKEND_URL' \
  /tmp/comics-compose-final.yaml; then
  exit 1
else
  echo "compose legacy surface: none"
fi

docker build \
  -t comics-go-api:contract-cleanup \
  -f Dockerfile .

sqlite3 -readonly db/comics.db \
  "PRAGMA integrity_check;
   SELECT COUNT(*) FROM sqlite_schema
   WHERE type='table' AND name='users';"

git diff --check
```

Expected:

- Static check prints `contract cleanup check: ok`.
- `make check` ends with `check passed`.
- Compose check prints `compose legacy surface: none`.
- Docker build exits `0` and builds only `cmd/server`.
- SQLite prints `ok` and `0`.
- `git diff --check` prints nothing.

- [ ] **Step 4: Commit the regression gate**

```bash
git add scripts/check-contract-cleanup.sh Makefile .github/workflows/ci.yml
git commit -m "ci: prevent legacy contract regressions"
```

Expected: one commit with the executable script, Make target, and CI invocation.

Rollback:

```bash
git revert --no-edit "$(
  git log -1 --format=%H --fixed-strings \
    --grep='ci: prevent legacy contract regressions'
)"
```

---

## Full Rollback Order

Revert code/documentation commits in reverse dependency order:

```bash
for message in \
  'ci: prevent legacy contract regressions' \
  'docs: define the supported rest architecture' \
  'chore: remove empty legacy sqlite users' \
  'refactor: remove legacy api fallbacks' \
  'chore: remove unused sql repositories' \
  'chore: retire unsupported grpc contract'; do
  commit="$(
    git log -1 --format=%H --fixed-strings --grep="$message"
  )"
  test -n "$commit"
  git revert --no-edit "$commit"
done
```

Then restore the pre-maintenance SQLite backup if the deployed database was modified:

```bash
make dev-down
backup="$(cat .dev/contract-cleanup-backup)"
test -n "$backup"
test -f "$backup"
make db-restore BACKUP="$backup"
make db-check
```

Expected: every revert succeeds, restore names the exact `.db` backup captured
before maintenance, and integrity prints `ok`.

If rollback is triggered by a newly discovered gRPC consumer, restoring source is only containment. Do not advertise or redeploy gRPC until the parity requirements in Gate 0 are met.

## Plan Self-Review

- [x] Consumer inventory precedes every breaking deletion and covers source, dependency graphs, build/deploy entrypoints, account-wide code search, deployed images, and data.
- [x] Both options are explicit: retirement is the evidence-backed recommendation; a discovered consumer blocks deletion and requires parity/migration work.
- [x] Protobuf source is removed on the recommended path and preserved only under concrete ownership/use/parity criteria.
- [x] The gRPC server, divergent PostgreSQL repository, generated Go/TypeScript protobuf clients, pb-only samplers, sqlc output, and duplicate SQL/ERD artifacts all have exact deletion lists; the canonical migration command/package is explicitly preserved.
- [x] OTLP gRPC tracing and live `db/db_classes.json` usage are explicitly protected from over-broad cleanup.
- [x] Python-era runtime/frontend env names and active docs are removed without deleting historical specs.
- [x] `db/comics.json` is removed only after proving it has no consumer; SQLite `.db` backup/restore remains canonical.
- [x] Legacy SQLite users are dropped only after an offline verified backup, a zero-row check, transactional maintenance, and catalog-count comparison.
- [x] Commands, expected outputs, commit messages, per-task rollback, operational DB rollback, and full rollback order are included.
- [x] Final checks cover Go, frontend, Docker, Compose, SQLite integrity, stale-path searches, and whitespace.
- [x] SQLite + MongoDB remain the default and PostgreSQL remains optional.
