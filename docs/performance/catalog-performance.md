# Catalog performance

This document tracks list/search pagination responsiveness for the comics catalog UI and Go API.

## Budgets (local dev, verified snapshot)

| Signal | Budget |
| --- | --- |
| UI acknowledges work (pending chrome) | `<50 ms` |
| Cached page paint | `<100 ms` |
| HTTP list p95 | `<100 ms` |
| HTTP search p95 | `<150 ms` |
| Trigram-eligible search repo p95 / p99 | `≤10 ms` / `≤25 ms` (≥3× vs ~25 ms scan baseline) |
| Short (1–2 rune) search p95 | `≤50 ms` |
| Search under scrape-like writes p95 / p99 | `≤25 ms` / `≤75 ms` |
| List during held write tx p95 | `<150 ms` |

Absolute timing checks are **opt-in** (`make perf-catalog`, strict gates via `make perf-backend COMICS_PERF_STRICT=1`). Default CI enforces query plans, FTS parity tests, concurrency behavior, and frontend unit tests to avoid shared-runner flakiness.

## Dataset

- Default perf snapshot: `db/backups/catalog-perf-snapshot.db` (copied from `db/comics.db` on first `make perf-catalog` after `PRAGMA integrity_check`).
- Override with `PERF_SNAPSHOT` (repo Makefile) or `COMICS_PERF_DB` (Go tests).

## Commands

```bash
# Deterministic guards (CI-safe)
make go-test
make front-test

# Opt-in before/after benchmarks (never uses unchecked DB)
make perf-catalog
make perf-backend
make perf-backend COMICS_PERF_STRICT=1
```

Go-only entry point when a snapshot path is already known:

```bash
make -C go_server perf-catalog COMICS_PERF_DB=../db/backups/catalog-perf-snapshot.db
make -C go_server db-rebuild-catalog-search COMICS_SQLITE_PATH=../db/backups/catalog-perf-snapshot.db
```

Rollback indexed search: `COMICS_SEARCH_MODE=scan` (legacy `LIKE` scan). Require index: `COMICS_SEARCH_MODE=trigram` (readiness fails if FTS is empty).

## Implementation notes

- **Frontend**: `AbortSignal` on list fetches, 24-entry LRU cache (2-minute TTL), in-flight deduplication, idle adjacent-page prefetch, stale-result banner, and immediate pagination/search pending state without clearing cards.
- **SQLite**: single writer connection plus a 4-connection read pool so list/search queries do not queue behind scrape/API writes.
- **Substring search**: SQLite uses FTS5 `tokenize='trigram'` with `fts.titles LIKE '%term%'` for terms with ≥3 runes (no `%`/`_` wildcards). Shorter terms and `COMICS_SEARCH_MODE=scan` keep the legacy `comics.titles LIKE` scan. Offline rebuild: `make -C go_server db-rebuild-catalog-search`.
- **Scrape isolation**: `SCRAPE_STARTUP_GRACE` (default 3m) before the first automatic scrape, bounded fetch concurrency (`SCRAPE_FETCH_CONCURRENCY`, default 6), response body cap, no-op update skip, and a bounded registration queue.
- **Observability**: Prometheus metrics under `comics_catalog_*` and `comics_scrape_*`; see `go_server/.observability/catalog-tail-latency.md`.

## Deployed observations

Record manual checks here after releases (input-to-pending, network, paint, cache hits under throttling). Cold starts on hosted backends may exceed local budgets; the UI should still show pending state within 50 ms.
