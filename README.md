# Comics MVP: Server, Scrapper and Interface
![GitHub license](https://img.shields.io/github/license/estebmaister/comics)
![GitHub last commit](https://img.shields.io/github/last-commit/estebmaister/comics)
![GitHub contributors](https://img.shields.io/github/contributors/estebmaister/comics)
![GitHub repo size](https://img.shields.io/github/repo-size/estebmaister/comics)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/estebmaister/comics)
![GitHub top language](https://img.shields.io/github/languages/top/estebmaister/comics)
![GitHub language count](https://img.shields.io/github/languages/count/estebmaister/comics)

## Quick start

```sh
make setup
cp local.env .env
make doctor

# terminal 1 — Go API (comics + auth + scrape)
make go-run

# terminal 2 — Vite frontend
make front-dev

# before pushing
make check
```

Run `make` or `make help` for the full grouped command list. Per-target detail:
`make help TARGET=py-server`.

## Make commands

The root [`Makefile`](Makefile) is the canonical developer reference. Common targets:

| Area | Targets |
|------|---------|
| Setup | `setup`, `setup-venv`, `setup-front`, `setup-go`, `doctor` |
| Frontend | `front-dev`, `front-dev-https`, `front-build`, `front-test`, `front-deploy` |
| Python | `py-server`, `py-scrape`, `py-daemon`, `py-daemon-stop` |
| Go | `go-run`, `go-test`, `go-import-postgres` |
| Database | `db-backup`, `db-restore`, `db-check`, `db-repair-identity`, `db-audit-covers` |
| Quality | `check`, `check-go`, `check-front`, `contract-test`, `py-test` |

`make check` runs Go test/vet/build (including HTTP contract tests), frontend test/build, and Python pytest.

## Scrapper and Server deployment (Go primary)

Production and local development default to the Go server on port `8081`. It serves
comics REST, JWT auth/profile, metrics, and native publisher scrapers. Python
Flask/scrape remain available for legacy workflows and contract-fixture generation.

### Virtual environment (Optional)

```sh
make setup-venv
make setup-shell   # prints activation command
```

> [!IMPORTANT]
> For a corrupted virtual env:
> ```sh
> make setup-venv-clean
> make setup-venv
> ```

### Installing dependencies

```sh
make setup          # Python + Go + frontend + protobuf
make deps-py-upgrade   # bump Python deps and rewrite requirements.txt
```

### Running scrapper

```sh
make py-scrape      # one pass
make py-daemon      # server + scraper loop in background (./output.log)
make py-daemon-stop
```

### Running tests

```sh
make py-test
make front-test
make go-test
make check          # all release-path checks
```

### Running Python server

```sh
make py-server
```

Set `PRODUCTION=true` on Render/production hosts to use gevent instead of the Flask dev server.
Set `DB_ENGINE=sqlite` explicitly to avoid env parsing warnings.

On startup, corrupt SQLite files are rebuilt automatically from `src/db/comics.json`.

### Running Go server

```sh
cp go_server/local.env go_server/.env
make go-run
```

The Go server listens on `https://localhost:8081` when `tls/comics.crt` and `tls/comics.key`
are present. See [`go_server/README.md`](go_server/README.md) for MongoDB, Postgres, and env details.

Import SQLite into Postgres:

```sh
make go-import-postgres
```

### Deployment on Heroku

```sh
git push heroku
heroku logs --tail
```

### Deployment on Render

Triggered on pushes to `main`. Typical settings:

- Start command: `python ./src server`
- Env: `PRODUCTION=true`, `DB_ENGINE=sqlite`, `PORT` from Render
- Ensure `src/db/comics.json` ships with the deploy (SQLite recovery source)

### Production Docker (SSH workflow)

On merge to `main`, CI can SSH to the host, `git pull`, and `docker compose up -d`.
See [`.github/workflows/ci-cd.yml`](.github/workflows/ci-cd.yml).

## Interface deployment (React + Vite)

### Running frontend

> [!TIP]
> Check the `tls` folder for HTTPS local development certificates.

```sh
make front-dev          # http://localhost:3000
make front-dev-https    # https://localhost:3000 (requires ./tls/comics.crt + .key)
```

Open the Vite URL printed in the terminal. The page reloads on file changes.

Frontend API env (`.env`):

- `VITE_API_SERVER` — preferred API base URL
- `VITE_PY_SERVER` — legacy fallback for the Python backend

### Running tests

```sh
make front-test
# or watch mode:
npm run test:watch
```

### Running build and deployment

```sh
make front-build
make front-deploy && git push origin   # GitHub Pages
```

Production build output goes to `build/`.
