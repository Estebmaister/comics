# Python → Go Migration — Parity Matrix

Reference for REST endpoints, scrapers, and frontend usage during backend migration.

## REST endpoint parity

| Method | Path | Frontend | Python Flask | Go Gin | Notes |
|--------|------|----------|--------------|--------|-------|
| GET | `/health` | No | Yes | Yes | Liveness |
| GET | `/health/db` | No | Yes | Yes | DB ping |
| GET | `/scrape` | Yes | Yes (native) | Proxy → native | ScrapeButton |
| GET | `/comics` | Yes | Yes | Yes | List + filters/sort |
| POST | `/comics` | Yes | Yes | Yes | Create; dedupe required |
| GET | `/comics/:id` | No | Yes | Yes | |
| PUT | `/comics/:id` | Yes | Yes | Yes | Track, rating, viewed_chap, edit |
| DELETE | `/comics/:id` | Yes | Yes | Yes | |
| PATCH | `/comics/:id/cover-visibility` | Yes | Yes | Yes | ComicCover fallback |
| GET | `/comics/search/:title` | Yes | Yes | Yes | NavBar search |
| PATCH/PUT | `/comics/:id/:merging_id` | Yes | Yes | Yes | MergeComic |

### List/search query params (frontend-used)

| Param | Frontend source | Python | Go |
|-------|-----------------|--------|-----|
| `from` | MainPage pagination | Yes | Yes |
| `limit` | MainPage pagination | Yes | Yes |
| `only_tracked` | NavBar | Yes | Yes |
| `only_unchecked` | NavBar | Yes | Yes |
| `rating_min` / `rating_max` | SortFilterModal | Yes | Yes |
| `sort_by` | SortFilterModal | Yes | Yes |
| `sort_dir` | SortFilterModal | Yes | Yes |
| `full` | No | Yes | Yes |

### Response headers

| Header | Python | Go |
|--------|--------|-----|
| `total-comics` | Yes | Yes |
| `total-pages` | Yes | Yes |
| `current-page` | Yes | Yes |

## Auth endpoints (Go only; Phase 2 frontend)

| Method | Path | Go | Frontend target |
|--------|------|-----|-----------------|
| GET/POST | `/signup` | Yes | `/signup` |
| GET/POST | `/login` | Yes | `/login` |
| POST | `/refresh-token` | Yes | Auth helper |
| GET/PUT | `/protected/profile` | Yes | `/profile` |
| GET | `/auth/google` | Yes | Optional OAuth |

## Scraper publishers (active → port to Go)

| Publisher | Python module | Go module | URL config |
|-----------|---------------|-----------|------------|
| ManhuaPlus | `manhuaplus.py` | `scrape/manhuaplus.go` | `url_switch.json` |
| Asura | `asura.py` | `scrape/asura.go` | astro-island JSON |
| FlameScans | `flame.py` | `scrape/flame.go` | CSS bsx grid |
| RealmScans | `realm.py` | `scrape/realm.go` | |
| DemonicScans | `demonic.py` | `scrape/demonic.go` | |
| Manganato | `manganato.py` | `scrape/manganato.go` | |

## Known parity gaps (Phase 1 targets)

| Area | Python | Go (before fix) |
|------|--------|---------------|
| `identity_key` | `series:` / `novel:` prefix | `{comType}:{title}` |
| Create dedupe | `canonical_comic_by_titles` | None |
| SQLite `identity_key` on INSERT | Yes | Missing column in INSERT |
| `comics.json` sync | After scrape/writes | None (use db-backup export) |

## Deferred

- Per-user `user_comic_states` (Phase 5)
- gRPC comics server consolidation
- `/protected/tasks` stub
- Python gRPC auth stub
