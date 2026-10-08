# Scrape coordination (Go)

Single-flight scheduling for manual, periodic, and CLI scrapes while the comics DB
stays available for normal REST traffic.

## Goals

1. **At most one scrape run** across HTTP `GET /scrape`, `SCRAPE_INTERVAL` background
   loop, and `cmd/scrape` / `make go-scrape`.
2. **Manual vs automatic** do not stack: a manual run while automatic is in progress
   is rejected; an automatic tick while any run is active is skipped (not queued).
3. **Cooldown after any run**: the next automatic run is scheduled for
   `last_completed_at + SCRAPE_INTERVAL`, where `last_completed_at` comes from
   manual, automatic, or CLI runs alike. A manual scrape therefore postpones the
   next automatic pass by a full interval (no immediate auto right after manual).
4. **DB concurrency**: scrapes and user CRUD share the same SQLite/Postgres store.
   Scrapes serialize **writes** inside the register path; readers and unrelated
   comic updates proceed under normal driver locking (SQLite WAL + busy timeout).

## Sources

| Source      | Entry                         | If another run is active      |
|-------------|-------------------------------|-------------------------------|
| `manual`    | `GET /scrape`                 | HTTP **409** + running source |
| `automatic` | periodic scheduler            | Skip tick (log, no queue)     |
| `cli`       | `go run ./cmd/scrape`         | Exit non-zero / log busy      |

Manual and CLI both reset the automatic cooldown when they finish.

## State machine

```text
                    ┌─────────────┐
         idle ─────►│   running   │─────► idle
                    │ (one source)│
                    └─────────────┘
                          │
        second trigger ───┴──► reject (manual/cli) or skip (automatic)
```

While **running**:

- `busy_source` ∈ {manual, automatic, cli}
- `last_completed_at` unchanged

On **finish** (success or failure):

- `busy` cleared
- `last_completed_at = now`
- `last_completed_source` recorded for logs/status

## Periodic scheduler

1. On server start, if `SCRAPE_INTERVAL > 0`, start the scheduler goroutine.
2. Compute sleep: `max(0, last_completed_at + interval - now)`; zero
   `last_completed_at` means **run immediately** (first boot tick).
3. Wake, call `Run(automatic)`; if `ErrScrapeInProgress`, wait until idle
   (short poll), then recompute sleep (manual/cli completion sets the cooldown).
4. Shutdown: cancel context; in-flight scrape respects request context where wired.

Default interval: **10m** (`SCRAPE_INTERVAL=10m`). Disable with `SCRAPE_INTERVAL=0`.

## HTTP contract

- **200** — scrape finished.
- **409** — scrape already running; body includes `source` (`manual` | `automatic` | `cli`).
- **500** — scrape failed mid-run.

`GET /scrape/status` returns `{ running, source, last_completed_at, last_completed_source }`
for the UI scrape button and polling while a run is active.

## Implementation map (Go)

| Piece | Package / file | Role |
|-------|----------------|------|
| `Coordinator` | `internal/scrape/coordinator.go` | Single-flight + timestamps |
| `PeriodicScheduler` | `internal/scrape/periodic.go` | Cooldown-aware loop |
| `ComicService.Scrape` | `internal/usecase/comics.go` | `Run(manual)` |
| `cmd/scrape` | `cmd/scrape/main.go` | `Run(cli)` |
| `runScrape` | `api/route/comics.go` | Map `ErrScrapeInProgress` → 409 |
| Wiring | `bootstrap/app.go`, `cmd/server/main.go` | One coordinator per process |

## Python retirement (phased)

| Phase | Action |
|-------|--------|
| **Done** | Python publisher scrapers removed; registration helpers live in `src/db/scraped_register.py`; `url_switch.json` kept for the frontend. |
| **Done** | Flask `/scrape` removed; use Go `GET /scrape` and `GET /scrape/status`. |
| **Done** | `cloudscraper` removed from `requirements.txt`. |

Production path: **Go `make go-run`** with `SCRAPE_INTERVAL` replaces **`py-daemon`**.
