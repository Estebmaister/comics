## Project Overview

A full-stack comic book tracking application with multi-source web scraping, Go-primary backend (Gin), legacy Python tooling, and React frontend. Production targets a single Go application for comics REST, auth, and native scrapers.

## Architecture Overview

- **Frontend**: React 19 + TypeScript + React Router
- **Backend**: Gin (Go, primary) + Flask (Python, legacy) + gRPC
- **Database**: SQLite (primary) with JSON backup storage
- **Data Pipeline**: Multi-site web scrapers with data normalization
- **Deployment**: GitHub Pages (frontend), Koyeb/Heroku/Render (backend)

## Frontend UI Notes

- The comics page is anchored in `src/frontend/components/Comics/MainPage/MainPage.tsx`.
- Cards live in `src/frontend/components/Comics/Card/` and use CSS Modules for card-only chrome.
- Shared modal shell lives in `src/frontend/components/Modal/`; form-specific grid styles live in `src/frontend/components/Comics/Modals/Modal.css`.
- Global toasts are provided by `src/frontend/components/Toast/ToastProvider.tsx`.
- The floating desktop/mobile utility dock lives in `src/frontend/components/Comics/Actions/FloatingActionRail.tsx`.

### Current UI Patterns

- Default theme is premium dark; preserve the dark-first visual system even when using light-theme references for inspiration.
- Card density is hybrid:
  - `1` column below `900px`
  - `2` columns from `900px` to `1599px`
  - `3` columns at `1600px+`
- Hover-only chrome on cards should be CSS-driven with `:hover` / `:focus-within`, not React hover state.
- Card stacking contract:
  - card overlays `<` navbar `<` floating action rail `<` toast viewport `<` modal
- Use the toast provider for create/edit/merge/scrape feedback instead of fixed inline message boxes.
- Keep create/edit modals roomy with sticky footers; keep merge compact and task-focused.
- Avoid loading modal code until the modal is opened. The create/edit flows already gate lazy modal rendering to keep React 19 tests quiet.

## Key Development Commands

Run `make` or `make help` for the grouped target list. Detail: `make help TARGET=<name>`.

### Setup

```bash
make setup                  # venv + Python + Go + frontend + protobuf + git hooks
make setup-venv             # Create Python virtual environment
make setup-front            # npm ci
make setup-go               # go mod tidy
make doctor                 # Validate venv, node_modules, env files, DB integrity
make status                 # Daemon PID, log path, default dev URLs
```

### Frontend Development

```bash
make front-dev              # Vite HTTP dev server (localhost:3000)
make front-dev-https        # Vite HTTPS dev server (requires TLS certs in ./tls/)
make front-test             # Vitest single run
make front-build            # Production build to build/
make front-deploy           # Deploy to GitHub Pages
```

### Python (DB tools + tests)

```bash
make db-backup              # Export SQLite to timestamped JSON backup
make db-restore             # Rebuild SQLite from comics.json
make db-check               # PRAGMA integrity_check
make db-repair-identity     # Dry-run duplicate repair (APPLY=1 to merge)
make db-audit-covers        # Probe cover URLs (ARGS="--apply" to persist)
```

### Go Backend (via Makefile)

```bash
make go-run                 # Gin server with air hot reload (:8081); SCRAPE_INTERVAL default 10m
make go-scrape              # One native scrape pass (no HTTP server)
make go-test                # Go tests
make go-import-postgres     # Import SQLite comics into Postgres
make go-migrate-up          # Run SQL migrations forward
make go-migrate-down        # Roll back SQL migrations
```

### Protocol Buffer Generation

```bash
make proto-py               # Generate Python gRPC bindings
make proto-js               # Generate JavaScript/TypeScript bindings
make proto-go               # Generate Go gRPC bindings
```

### Testing and release checks

```bash
make py-test                # Python pytest suite
make front-test             # Vitest
make go-test                # Go tests
make check                  # All release-path checks (Go + frontend + Python)
make check-go               # go-test + go-vet + go-build
make check-front            # front-test + front-build
```

### Docker

```bash
make docker-build           # Build Go API image (comics-go-api)
make docker-run             # Compose: API + Mongo (:8081)
make docker-dev             # Foreground compose stack
make docker-down            # Stop compose stack
```

## Database Structure

### Comic Schema (SQLite + JSON)

- **Primary DB**: `src/db/comics.db` (SQLite)
- **Backup/Export**: `src/db/comics.json` (derived from SQLite after successful batch operations)
- **Key Fields**:
  - `id`: Unique comic identifier
  - `titles[]`: Multiple title variations for search flexibility
  - `identity_key`: Normalized primary-title identity used for duplicate prevention and novel/comic separation
  - `com_type`: MANGA, MANHUA, MANHWA, WEBTOON, NOVEL
  - `status`: COMPLETED, ON_AIR, BREAK, DROPPED
  - `current_chap`/`viewed_chap`: Progress tracking
  - `track`: User tracking preference
  - `rating`: F to SSS rating system
  - `published_in[]`: Scan groups/publishers

### Data Flow

1. **Web Scraping**: Multiple specialized scrapers extract data
2. **Processing**: `normalize_text()` and identity-key normalization run before duplicate checks
3. **Storage**: SQLite is canonical for runtime; `comics.json` is regenerated/persisted from the DB state after successful writes
4. **API**: REST via Go Gin (primary); gRPC partial

### Discovery Identity Rules

- Discovery uniqueness is based on `identity_key`, not `LIKE '%title%'`.
- `identity_key` is built from the normalized primary title plus a novel/comic scope:
  - `series:<normalized-title>` for non-novels
  - `novel:<normalized-title>` for novels
- Scrapers must pass titles through `normalize_text()` before model creation. `ComicDB` also normalizes titles and refreshes `identity_key` on insert/update as a safety net.
- Stored titles use sentence-case normalization via `.capitalize()`: first character uppercase, remaining characters lowercase. Preserve that convention when fixing data or adding alternate titles.
- The scraper runs as a batched transaction:
  - per-comic work uses savepoints (`begin_nested`) so one bad entry does not invalidate the whole scrape
  - the real DB commit happens once at the end of the full scrape run
  - `comics.json` should only be written after that final commit succeeds
- Periodic scraping: `SCRAPE_INTERVAL` on `make go-run`; manual `GET /scrape` and `GET /scrape/status` share one coordinator (see `docs/superpowers/specs/2026-10-08-scrape-coordination.md`).
- Current duplicate repair entry point: `make db-repair-identity` (dry-run) or `make db-repair-identity APPLY=1`. Under the hood: `src/db/repair_identity_duplicates.py`.
- When a historical duplicate group still conflicts on non-novel type, repair policy is `lowest id wins`. Use `--merge-ambiguous` to apply that rule and finish the dedupe pass.
- Use `--normalize-all-titles` when you need a full-catalog title storage cleanup after dedupe. That pass rewrites stored title variants to the repo’s sentence-case convention and rebuilds `comics.json`.

## Web Scraping Sources

### Currently Supported Scrapers

- Asura Scans
- Manhua Plus
- Flame Scans
- Realm Scans
- Demonic Scans
- Manganato

### Scraper Architecture

- Location: `go_server/internal/scrape/` (production); `src/scrape/url_switch.json` for publisher URLs (frontend + Go embed)
- Each Go publisher module handles site-specific HTML parsing
- Built-in error handling and retry logic
- Automatic data normalization to common schema
- Runtime dedupe is centralized in `src/scrape/scrapper.py`; publisher-specific modules should stay extraction-only

## Development Environment Setup

### Prerequisites

- Python 3.8+ with virtual environment support
- Node.js 16+ and npm
- Go 1.26.5+ (for Go backend; `GOTOOLCHAIN=auto` if local Go is older)
- Protocol Buffers compiler (`protoc`)
- Local AI service (localhost:11434) for pre-commit hooks

### Initial Setup

```bash
make setup                  # venv, Python, Go, frontend, protobuf, git hooks
cp local.env .env           # Python backend env (optional but recommended)
make doctor                 # Validate the environment
```

Manual steps if needed:

```bash
make setup-venv
make setup-front
make setup-go
```

### HTTPS Development (Frontend)

- Requires TLS certificates in `./tls/` directory
- `comics.crt` and `comics.key` files needed for `make front-dev-https`
- Allows testing features requiring secure context

## Pre-commit Hooks and AI Integration

### Automated Hooks

- **Location**: `.githooks/pre-commit`
- **Features**:
  - AI-powered security/performance analysis
  - Automated changelog generation
  - Code review assistance
- **Requirements**:
  - Local AI service running on `localhost:11434` (phi4 model)
  - `jq`, `curl` command-line tools

### Hook Configuration

```bash
chmod +x .githooks/pre-commit  # Ensure hooks are executable
```

## Current Technical Debt and Known Issues

1. **Data Consistency**: Historical duplicate identity groups still need to be repaired before the unique `identity_key` index can be enforced on older DB files
2. **Search Functionality**: Special character handling still needs improvement
3. **gRPC Implementation**: Partially implemented, needs completion
4. **Database Migration**: MongoDB/Neo4j migration is still exploratory
5. **Frontend Typing**: Some modal/network helpers still rely on permissive object shapes and should continue moving toward stricter shared types

## Environment Variables

### Frontend (.env)

- `VITE_API_SERVER`: preferred API base URL for local/production frontend builds
- `VITE_PY_SERVER`: legacy fallback Python server URL (default: http://localhost:5001)
- `VITE_EXTERNAL_HOST`: External host for deployment configurations

### Backend

- `DB_ENGINE`: Database engine selection (`sqlite`, `postgresql`, `postgres`, `mysql`)
- `PRODUCTION`: Set `true` on Render/production to use gevent instead of Flask dev server
- `DEBUG`: Enable debug mode
- `PORT`: HTTP listen port (default 5001)
- `DB_FILE`: SQLite path (default `src/db/comics.db`)
- Database-specific credentials for PostgreSQL/MySQL support

Corrupt SQLite on startup is rebuilt automatically from `src/db/comics.json`.

## Container Support

```bash
make docker-build           # Build Docker image
make docker-run             # Run Docker container
make docker-dev             # Run with file watching for development
```

## Deployment

### Frontend

- **Target**: GitHub Pages (https://estebmaister.github.io/comics)
- **Command**: `make front-deploy`
- **Configuration**: `homepage` field in package.json

### Backend Options

- **Render / Docker**: root `Dockerfile` + `docker compose` (Go API on `:8081`)
- **Heroku**: Via `git push heroku`
- **Docker/SSH production**: `docker compose up -d` on host after merge to main

## File Structure Highlights

```
├── src/
│   ├── frontend/           # React application
│   │   ├── components/     # React components (Comics/, Loaders/, Modal/, Toast/)
│   │   ├── pb/            # Generated TypeScript protobuf bindings
│   │   └── css/           # Styling
│   ├── db/                # Database operations and migrations
│   ├── scrape/            # Web scraping modules
│   └── pb/                # Generated Python protobuf bindings
├── go_server/             # Go backend alternative
│   ├── cmd/               # Go application entry points
│   ├── pkg/pb/            # Generated Go protobuf bindings
│   └── migrations/        # Database migration files
├── proto/                 # Protocol Buffer definitions (.proto files)
└── tls/                   # TLS certificates for HTTPS development
```

## Common Development Patterns

### Adding New Comic Scrapers

Go (production): add publisher module under `go_server/internal/scrape/`, register in
`runner.go`, and add fixture tests beside the extractor.

Python registration tests use `src/db/scraped_register.py` (no HTTP scrapers).

### Frontend Component Development

- Components are organized by feature in `src/frontend/components/`
- Prefer feature-local CSS Modules for self-contained visuals like cards; use shared CSS only for cross-cutting primitives such as modals or global tokens
- Use the existing `Modal` base component for dialogs and the toast provider for transient feedback
- Preserve the premium dark card language: glass overlay actions, strong poster framing, and explicit footer action lanes
- When changing card layout, update pagination helpers in `src/frontend/components/Comics/utils.ts` to keep page sizes aligned with the visual density
- Generated protobuf bindings remain in `src/frontend/pb/`

### Database Operations

- Use the SQLAlchemy model in `src/db/__init__.py`
- Exact duplicate prevention must go through `identity_key` helpers in `src/db/identity.py` and repo helpers in `src/db/repo.py`
- Use `comics_like_title()` only for fuzzy search/UI flows, not for discovery or duplicate prevention
- `src/db/repair_identity_duplicates.py` is the maintenance script for auditing and merging exact duplicate groups
- Both SQLite and PostgreSQL support available

## Testing Notes

- `npm test -- --watchAll=false` or `make front-test` is the quickest frontend regression pass.
- React 19 will warn if lazily loaded modal components resolve during tests without `act(...)`; avoid that by only rendering lazy modal components when they are open, or by awaiting them explicitly in tests.
- Add focused tests for layout math (`utils.ts`), toast timing, and card fallback states when touching the comics UI.
- Backend duplicate-prevention tests live alongside scraper tests; cover smart-quote normalization, novel/comic identity splits, and scrape normalization boundaries when changing discovery logic.
