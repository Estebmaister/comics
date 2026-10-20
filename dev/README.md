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
make dev-down     # stop API, frontend, and local Mongo if dev-up started it
make dev-restart  # down then up
```

Foreground alternative: `make go-run` + `make front-dev` in two terminals.
