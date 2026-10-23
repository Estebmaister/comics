# Render: migrate Python web service to Go

The Python stack (`pip install`, `python ./src server`) was removed from this repo. Use the **root `Dockerfile`** (recommended) or a **native Go** build.

## Recommended: Docker runtime

In **comic-tracker-server → Settings → Build & Deploy**:

| Setting | Value |
|--------|--------|
| **Runtime** | Docker |
| **Dockerfile path** | `Dockerfile` (repository root) |
| **Docker build context** | `.` (repo root) |
| **Build command** | *(leave empty — Docker builds the image)* |
| **Start command** | *(leave empty — image `CMD ["./server"]`)* |
| **Root directory** | *(empty)* |

Clear any legacy **Python** build/start commands (`pip install -r requirements.txt`, `python ./src server`).

### Persistent disk (SQLite catalog)

Without a disk, the comics SQLite DB is lost on every deploy.

1. **Settings → Disks → Add disk**
2. **Mount path:** `/data`
3. **Size:** 1 GB (or more if your catalog grows)

The Docker image sets `COMICS_SQLITE_PATH=/data/comics.db` by default.

**First deploy on a new disk:** the DB file is empty. Either:

- Open **Shell** on the service and copy a backup, e.g. upload `comics.db` to `/data/comics.db`, or
- Let `SCRAPE_INTERVAL` repopulate over time (slower; user progress only if you restore a file).

### Health check

**Settings → Health & Alerts**

- **Health check path:** `/health`

### Environment variables

Set in **Environment** (reuse secrets from the old Python service where names match):

| Variable | Example / notes |
|----------|-----------------|
| `ENVIRONMENT` | `production` |
| `HOST_URL` | `comic-tracker-server.onrender.com` (your `*.onrender.com` host, no `https://`) |
| `DB_ADDR` | MongoDB URI host (e.g. `mongodb+srv://cluster.mongodb.net/`) |
| `DB_USER` | Mongo user |
| `DB_PASS` | Mongo password |
| `DB_NAME` | `comics` |
| `ACCESS_TOKEN_SECRET` | strong secret |
| `REFRESH_TOKEN_SECRET` | strong secret |
| `CORS_ALLOWED_ORIGINS` | `https://estebmaister.github.io` (+ any dev origins you need) |
| `SCRAPE_INTERVAL` | `10m` (optional background scrape) |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | if Google login is enabled |

**Remove or leave blank** (platform TLS terminates HTTPS; do not enable app TLS on Render):

- `HTTP_TLS_CERT_FILE`
- `HTTP_TLS_KEY_FILE`
- `PY_BACKEND_URL`

**Do not set** `HTTP_PORT` unless debugging — Render sets `PORT`; the Go server maps it automatically when `HTTP_PORT` is unset.

Optional overrides:

| Variable | Default in Docker image |
|----------|-------------------------|
| `COMICS_SQLITE_PATH` | `/data/comics.db` |
| `COMICS_URL_SWITCH` | `/app/config/url_switch.json` (bundled in image) |
| `HTTP_ADDRESS` | `0.0.0.0` |

### Deploy

Save settings → **Manual Deploy → Deploy latest commit** (or wait for auto-deploy on `main`).

Verify:

```bash
curl -sS "https://YOUR-SERVICE.onrender.com/health"
```

Point the GitHub Pages frontend at the same host (`VITE_API_SERVER` in your Pages build / env).

---

## Alternative: native Go (no Docker)

Use this only if you prefer Render’s Go buildpack over Docker.

| Setting | Value |
|--------|--------|
| **Runtime** | Go |
| **Root directory** | *(empty — repo root)* |
| **Build command** | `GOTOOLCHAIN=auto go build -C go_server -ldflags="-w -s" -o ../bin/comics-server ./cmd/server` |
| **Start command** | `./bin/comics-server` |

Add a disk mounted at `/var/data` and set:

- `COMICS_SQLITE_PATH=/var/data/comics.db`
- `COMICS_URL_SWITCH=config/url_switch.json`

Same Mongo/JWT/`HOST_URL` variables as above.

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `requirements.txt` not found | Runtime is still Python — switch to **Docker** or **Go** and clear pip build command |
| Connection refused / deploy “live” but 502 | App not listening on Render’s `PORT` — unset `HTTP_PORT` in env; redeploy with latest Go bootstrap |
| Empty catalog after deploy | Add disk at `/data` and restore `comics.db`, or run scrape |
| Auth works, comics 401 | JWT secrets or `HOST_URL` mismatch vs frontend |
| CORS errors from GitHub Pages | Add `https://estebmaister.github.io` to `CORS_ALLOWED_ORIGINS` |
