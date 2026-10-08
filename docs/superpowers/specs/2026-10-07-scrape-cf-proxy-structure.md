# Scrape: Cloudflare hold, dev proxy, code layout

## Nelomanga / Cloudflare (on hold)

**Goal:** Paginate `https://www.nelomanga.net/manga-list/latest-manga?page=N` in Go without Python.

**What works today**

| URL | Go Chrome TLS (`tls-client`) | Notes |
|-----|------------------------------|--------|
| `https://www.nelomanga.net/` | 200, ~56 `div.itemupdate.first` | In `url_switch.json` |
| `/manga-list/latest-manga?page=N` | 403 or challenge HTML | Blocked |

**Approaches tried (no durable fix for listing pages)**

1. Plain `net/http` + Chrome `User-Agent` + `Referer: https://www.nelomanga.net/`
2. Python **`cloudscraper`** (`browser='chrome'`) — same as legacy `scrapper.py` (403 on manga-list)
3. **Headless Chromium (rod + stealth)** — still challenge HTML on manga-list
4. **Go `bogdanfinn/tls-client`** with `profiles.Chrome_120`, cookie jar, home warmup before listing — home OK, manga-list 403 / “Just a moment…”

**Current behavior (Go)**

- `ChromeTLSFetcher` (`go_server/internal/scrape/chrome_fetch.go`) is the default `NewPageFetcher()`.
- Challenge HTML is a **soft fail** (warn + empty body); scrape run continues.
- **Manganato URLs in `url_switch.json` are home-only** until listing bypass exists. Route to re-enable:  
  `https://www.nelomanga.net/manga-list/latest-manga?page=1` …

**Deferred options (not implemented)**

- Browser automation with human-in-the-loop (out of scope for “simple Go server”)
- External solver (FlareSolverr, etc.)
- Official/API feed if the site exposes one

---

## Demonic Scans pagination

**Configured:** `index.php` + `lastupdates.php?list=2` … `list=50` (Go + Python `url_switch.json`).

**Verified (2026-10-07):** Each list returns **~40** `div.updates-element.border-box` rows; titles differ across lists (e.g. list=2 vs 21 vs 50). Live audit previously reported 40 extracts per list for 2–20.

**Not fully mapped:** Site does not expose a clear “max list” in HTML nav; high indices (e.g. 200+) still return 40 rows — treat list index as opaque pages, extend config when audits show new coverage.

**Go extractor:** `scrapeDemonicPage` in `publishers.go` (`updates-element-info`, `a.chplinks`).

---

## Local dev: why `/api` proxy exists

| Piece | Local default |
|-------|----------------|
| Frontend | `http://localhost:3000/comics/` (Vite) |
| Go API | `https://localhost:8081` (local TLS certs in `tls/`) |
| `VITE_API_SERVER` | `/api` → Vite proxies to Go, strips `/api` prefix |

**Problems the proxy solves**

1. **Self-signed TLS** — Browser `fetch('https://localhost:8081/...')` fails until the cert is trusted; same-origin `/api` avoids that.
2. **Single origin** — Cookies and auth headers behave like one app (important as auth matures).
3. **Mixed setup** — HTTP Vite + HTTPS Go without teaching every dev to install the cert.

**Go already allows direct calls**

- `CORS_ALLOWED_ORIGINS` includes `http://localhost:3000` (and GitHub Pages).
- `Config.ts` fallback: `https://localhost:8081` when `VITE_API_SERVER` is unset.

**Ways to drop the proxy (simpler mental model)**

| Option | Frontend env | Backend | Tradeoff |
|--------|----------------|---------|----------|
| **A. Keep proxy (current)** | `VITE_API_SERVER=/api` | HTTPS :8081 | Extra Vite config; best DX with self-signed TLS |
| **B. Direct HTTPS** | unset or `https://localhost:8081` | HTTPS :8081 | Trust `tls/comics.crt` once per browser |
| **C. Direct HTTP in dev** | `http://localhost:8081` | HTTP only in dev (no TLS) | Simplest wire-up; TLS only in prod behind platform terminator |

**Production (GitHub Pages)** — No Vite proxy. Set `VITE_API_SERVER` to the public API URL at build time. CORS must list `https://estebmaister.github.io` (already default).

**Security note:** Proxy is a **dev convenience**, not an auth boundary. Production security = HTTPS, CORS allowlist, JWT/cookies, no `secure: false` to arbitrary hosts.

---

## Code structure (pre–full Python removal)

**Target layout (Go owns runtime; Python shrinks to fixtures/one-off tools)**

```
go_server/
  cmd/server/           # HTTP entry
  internal/
    scrape/             # fetch + publishers + register + runner (canonical)
    identity/           # identity_key / dedupe
    repo/comics/        # SQLite/Postgres
    usecase/            # REST use cases
  api/route/            # Gin handlers + contract tests

src/scrape/             # LEGACY: mirror until deleted; no new features
src/frontend/           # UI only; API via Config.ts

proto/                  # gRPC (partial; defer consolidation)
```

**Migration hygiene**

1. One **`url_switch.json`** source of truth — today duplicated under `go_server/internal/scrape/` (embed) and `src/scrape/`; plan: generate embed from root `config/url_switch.json` or sync in `make check`.
2. **Publisher module split** (optional): `scrape/publishers/asura.go` etc. instead of single `publishers.go` as file grows.
3. **Fetch layer:** `fetch.go` (stdlib tests), `chrome_fetch.go` (production TLS client).
4. Delete Python scrape path only when Go audit (`AUDIT_SCRAPE=1`) matches per-publisher counts and `/scrape` is sole production path.

**Suggested phases**

| Phase | Work |
|-------|------|
| P0 | Docs + Demonic lists 21–50; Manganato home-only |
| P1 | Unify `url_switch.json`; Makefile `check` fails on drift |
| P2 | Split `publishers.go`; drop `src/scrape` from `py-daemon` default |
| P3 | Nelomanga listing when CF strategy exists; remove Python scrape package |
