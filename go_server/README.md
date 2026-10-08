# Go Server

Gin HTTP server for the full current Go app: Mongo-backed auth/profile routes,
Swagger, metrics, and SQLite/Postgres-backed comics REST routes.

## Requirements

- Go 1.26+
- MongoDB for auth/profile data
- Writable comics DB: SQLite at `COMICS_SQLITE_PATH` or Postgres at
  `COMICS_POSTGRES_URL`

## Local Run

```sh
cd go_server
cp local.env .env
# Edit .env: set DB_PASS for Atlas user esteb
make doctor
make go-run
```

**MongoDB Atlas** (matches `mongosh "mongodb+srv://sandbox.ux3yw.mongodb.net/" --username esteb`):

```env
DB_ADDR=mongodb+srv://sandbox.ux3yw.mongodb.net/
DB_USER=esteb
DB_PASS=your_mongodb_password
DB_NAME=comics
```

**Local Docker** (offline dev): `make mongo-up`, `make mongo-init`, then use
`DB_ADDR=mongodb://localhost:27017` and `DB_PASS=localdev` in `.env`.

Run `make` or `make help` for the full grouped target list. Per-target detail:
`make help TARGET=go-run`.

`make go-run` starts air with Delve on `127.0.0.1:2345` for IDE attach. Use
`make go-run-plain` when you do not need a debugger. After upgrading Go, run
`make setup-tools` once to refresh Delve and other dev tools.

Default local endpoints:

- HTTPS server: `https://localhost:8081`
- Swagger: `https://localhost:8081/swagger/index.html`
- Liveness: `https://localhost:8081/health`
- Full readiness: `https://localhost:8081/ready`
- Comics DB health: `https://localhost:8081/health/db`

Local development uses `../tls/comics.crt` and `../tls/comics.key` when those
files exist. Production deploys should leave `HTTP_TLS_CERT_FILE` and
`HTTP_TLS_KEY_FILE` blank and rely on platform TLS termination.

`/scrape` runs the native Go scraper pipeline against SQLite or Postgres. Python
is no longer required for scrape operations.

- One-shot: `make go-scrape` (from repo root) or `make go-scrape` in `go_server/`.
- Periodic: `SCRAPE_INTERVAL` defaults to **10m** in `go_server/.env` (`0` disables).
  Manual `GET /scrape`, CLI, and automatic runs share a single coordinator (no overlap;
  manual resets the automatic cooldown). See
  `docs/superpowers/specs/2026-10-08-scrape-coordination.md`.

Scrape HTTP uses a Go Chrome TLS client (`tls-client`, same idea as Python
`cloudscraper`). Nelomanga listing is on hold (Cloudflare); see
`docs/superpowers/specs/2026-10-07-scrape-cf-proxy-structure.md`.

FlameScans is inactive (`flamecomics.xyz` currently redirects away from the
catalog).

## Docker Run

**Production / SSH deploy** (repo root):

```sh
cp go_server/local.env go_server/.env   # configure secrets on the host
docker compose up -d --build             # Dockerfile at repo root, port 8081
```

**Local compose from `go_server/`** (same stack, build context inside `go_server/`):

```sh
cd go_server
docker compose up --build
```

Both mount `../src/db` as writable `/data` for SQLite and expose HTTP on port `8081`.

## Comics Database

SQLite remains the rollback/source database at `../src/db/comics.db`. To run
the Go REST API against Postgres, create ignored `go_server/.env` from
`local.env` and set:

```env
COMICS_DB_DRIVER=postgres
COMICS_POSTGRES_URL=postgresql://USER:PASSWORD@HOST/DB?sslmode=require
COMICS_SQLITE_PATH=../src/db/comics.db
```

Import the SQLite source of truth into Postgres:

```sh
cd go_server
make go-import-postgres
```

The importer creates the Python-compatible Postgres schema, backs up any
existing target rows to a timestamped `comics_backup_*` table, replaces target
rows with SQLite contents, resets `comic_id_seq`, and verifies aggregate counts.

Current SQLite source size is about 9.9 MiB for 31,444 comics. Expected
Postgres size with table and index overhead is roughly 10-20 MiB.

Rollback is config-only unless Postgres data must be restored: set
`COMICS_DB_DRIVER=sqlite` and keep `COMICS_SQLITE_PATH=../src/db/comics.db`.

## Frontend Against Go

For local frontend development, point Vite at the Go server with:

```env
VITE_API_SERVER=https://localhost:8081
```

The Go server allows local HTTP and HTTPS Vite origins by default through
`CORS_ALLOWED_ORIGINS`.

## Important Env

- `HTTP_ADDRESS`, `HTTP_PORT`, `HOST_URL`: HTTP bind and public host values.
- `DB_ADDR`, `DB_NAME`, `DB_TABLE_USERS`: MongoDB user/auth store (`DB_USER` / `DB_PASS` for authenticated local dev; see `local.env`).
- `ACCESS_TOKEN_SECRET`, `REFRESH_TOKEN_SECRET`: JWT signing secrets.
- `COMICS_DB_DRIVER`: `sqlite` or `postgres`.
- `COMICS_SQLITE_PATH`: SQLite comics DB path.
- `COMICS_POSTGRES_URL`: Postgres comics DB URL with `sslmode=require` when needed.
- `CORS_ALLOWED_ORIGINS`: comma-separated browser origins allowed to call Go.
- `HTTP_TLS_CERT_FILE`, `HTTP_TLS_KEY_FILE`: local TLS cert/key paths; blank in production.
- `VITE_API_SERVER`: frontend API base URL override for local Go development.

## Comics REST Routes

- `GET /comics`
- `POST /comics`
- `GET /comics/:id`
- `PUT /comics/:id`
- `DELETE /comics/:id`
- `PATCH /comics/:id/cover-visibility`
- `GET /comics/search/:title`
- `PATCH /comics/:base_id/:merging_id`
- `PUT /comics/:base_id/:merging_id`

The merge route is registered in Gin as `/comics/:id/:merging_id` to avoid a
wildcard conflict with `/comics/:id`; the first segment is still the base comic
ID on the wire.

## Architecture

The Go app keeps delivery, use cases, and persistence separated:

- `api/route` and `api/controller`: HTTP binding, cookies, headers, and DTOs.
- `internal/usecase`: comics, auth, and user application rules.
- `domain`: entities, app errors, and use-case/repository interfaces.
- `internal/repo`: Mongo, SQLite, and Postgres persistence adapters.

Comics REST keeps the Python-compatible SQLite/Postgres storage format while
the merge, patch, pagination, and validation rules live in the comics use case.

## Checks

From the repo root:

```sh
make check-go
make contract-test
```

From `go_server/`:

```sh
make help
make doctor
make check-go
```

Common targets:

- `make go-run` — air hot reload with Delve on `127.0.0.1:2345`
- `make go-run-plain` — air hot reload without debugger
- `make go-test` / `make go-vet` / `make go-build`
- `make contract-test` — comics REST parity tests
- `make swag` — regenerate Swagger after annotation changes

If Delve reports a Go version mismatch after upgrading the toolchain, run
`make setup-tools` to refresh dev tools from `tool.go.mod`.

## Generated Docs

When Swagger annotations change:

```sh
make swag
```
