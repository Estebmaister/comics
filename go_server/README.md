# Go Server

Gin HTTP server for the full current Go app: Mongo-backed auth/profile routes,
Swagger, metrics, and SQLite/Postgres-backed comics REST routes.

## Requirements

- Go 1.24.1
- MongoDB for auth/profile data
- Writable comics DB: SQLite at `COMICS_SQLITE_PATH` or Postgres at
  `COMICS_POSTGRES_URL`

## Local Run

```sh
cd go_server
cp local.env .env
go run ./cmd/server
```

Default local endpoints:

- HTTPS server: `https://localhost:8081`
- Swagger: `https://localhost:8081/swagger/index.html`
- Liveness: `https://localhost:8081/health`
- Full readiness: `https://localhost:8081/ready`
- Comics DB health: `https://localhost:8081/health/db`

Local development uses `../tls/comics.crt` and `../tls/comics.key` when those
files exist. Production deploys should leave `HTTP_TLS_CERT_FILE` and
`HTTP_TLS_KEY_FILE` blank and rely on platform TLS termination.

`/scrape` proxies to `PY_BACKEND_URL` when configured. When it is blank, the
route returns `501` so the Python scraper dependency is explicit.

## Docker Run

```sh
cd go_server
docker compose up --build
```

The compose file starts MongoDB, waits for it to be healthy, mounts the root
SQLite DB directory as writable at `/data`, and exposes the HTTP server on
`http://localhost:8081`.

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
go run ./cmd/import_comics_postgres
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
- `DB_ADDR`, `DB_NAME`, `DB_TABLE_USERS`: MongoDB user/auth store.
- `ACCESS_TOKEN_SECRET`, `REFRESH_TOKEN_SECRET`: JWT signing secrets.
- `COMICS_DB_DRIVER`: `sqlite` or `postgres`.
- `COMICS_SQLITE_PATH`: SQLite comics DB path.
- `COMICS_POSTGRES_URL`: Postgres comics DB URL with `sslmode=require` when needed.
- `PY_BACKEND_URL`: optional Python backend for scraper proxy.
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

```sh
go test ./...
go vet ./...
go build ./cmd/server
```

## Generated Docs

When Swagger annotations change:

```sh
make swag
```
