# Local development helpers

Production (GitHub Pages + hosted Go API) does **not** use this folder.

## Vite API proxy (`proxy/`)

The React app runs on `http://localhost:3000` while Go listens on
`https://localhost:8081` with repo TLS certs. Browsers block or annoy direct
`fetch` to self-signed HTTPS, so Vite proxies same-origin `/api` → Go.

- Set `VITE_API_SERVER=/api` in `local.env` (see `proxy/env.example`).
- Optional override: `VITE_GO_API_TARGET=https://localhost:8081`

To skip the proxy: trust `tls/comics.crt`, unset `VITE_API_SERVER`, and rely on
`Config.ts` fallback `https://localhost:8081` (Go CORS must allow `:3000`).

## Full stack (`stack.sh`)

```bash
make dev-up       # Go API (air) + Vite in background → .dev/logs/
make dev-status   # PIDs and /health check
make dev-logs     # tail -f API + frontend logs
make dev-down     # stop API, frontend, optional Mongo; WAL checkpoint (bounded)
make dev-restart  # down then up
```

### Termux / Android

The Go module targets **Go 1.24.2** (Termux `pkg install golang`). From the repo:

```bash
cd ~/storage/dev/comics   # or your clone
export GOTOOLCHAIN=local
make dev-up               # native `go build` + Vite
```

Set `DB_PASS` in `go_server/.env` for MongoDB Atlas.

**Optional — UI on phone, API on Mac/PC (Tailscale):**

```bash
export DEV_FRONTEND_ONLY=1
export VITE_GO_API_TARGET=https://<mac-tailscale-host>:8081
make dev-up
```

Cross-compiled `linux/arm64` binaries from macOS often fail on Termux (TLS segment alignment); prefer native build.

Env (optional):

- `DEV_SHUTDOWN_SEC=90` — max wait for Go API exit (increase if WAL shutdown is slow).
- `DEV_CHECKPOINT_SEC=180` — max wait for `PRAGMA wal_checkpoint`; set `0` to skip on `dev-down`.
- `DEV_CHECKPOINT_SEC=600` — use on large WAL files; run `make db-checkpoint-wal` later if it times out.

Foreground alternative: `make go-run` + `make front-dev` in two terminals.
