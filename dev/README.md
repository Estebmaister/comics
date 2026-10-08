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
