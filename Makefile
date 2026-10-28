.DEFAULT_GOAL := help

# Define phony targets (targets that don't represent actual files)
.PHONY: help setup setup-go setup-front \
				front-dev front-dev-https front-build front-test front-deploy \
				go-run go-scrape go-test go-vet go-build go-migrate-up go-migrate-down go-import-postgres go-dedupe mongo-up mongo-init \
				db-backup db-restore db-check db-checkpoint-wal db-freeze db-reindex \
				db-audit-discovery db-export-discovery-plan db-validate-discovery-plan db-apply-discovery-plan db-audit-discovery-live \
				check check-go check-front perf-catalog perf-backend \
				proto-go proto-js \
				dev-up dev-down dev-status dev-logs dev-restart \
				docker-build docker-run docker-dev docker-down clean doctor status

# Enable running multiple commands in a recipe using a single shell
.ONESHELL:

DB_FILE ?= db/comics.db
COMICS_JSON ?= db/comics.json
DISCOVERY_PLAN ?= db/recovery/discovery-repair-20261026.json
DISCOVERY_BACKUP ?=
DISCOVERY_GROUP ?=
DISCOVERY_ALL_APPROVED ?= 0
CONFIRM ?=
BACKUP_FILE ?=
PERF_SNAPSHOT ?= db/backups/catalog-perf-snapshot.db

# Colors for help text
CYAN := \033[36m
DIM := \033[2m
BOLD := \033[1m
RESET := \033[0m

# Per-target detail: make help TARGET=go-run
TARGET ?=

help:
	@if [ -n "$(TARGET)" ]; then \
		lines=$$(grep -E '^##\? $(TARGET)[[:space:]]' $(MAKEFILE_LIST) || true); \
		if [ -z "$$lines" ]; then \
			echo "Unknown target: $(TARGET)"; \
			echo "Run 'make help' for the full list."; \
			exit 1; \
		fi; \
		echo "$(BOLD)$(TARGET)$(RESET)"; \
		echo "$$lines" | sed -E 's/^##\? $(TARGET)[[:space:]]*/  /'; \
	else \
		echo "$(BOLD)Comics — available targets$(RESET)"; \
		echo ""; \
		for section in setup frontend go database proto quality deploy docker util; do \
			case "$$section" in \
				setup) label="Setup";; \
				frontend) label="Frontend";; \
				go) label="Go";; \
				database) label="Database";; \
				proto) label="Protobuf";; \
				quality) label="Quality";; \
				deploy) label="Deploy";; \
				docker) label="Docker";; \
				util) label="Utility";; \
			esac; \
			lines=$$(grep "^## @$$section:" $(MAKEFILE_LIST) | sed "s/^## @$$section://" | sed 's/^[ \t]*//' | awk '{desc=$$0; sub(/^[^ \t]+[ \t]+/, "", desc); printf "  %-22s %s\n", $$1, desc}'); \
			if [ -n "$$lines" ]; then \
				echo "$(CYAN)$$label$(RESET)"; \
				echo "$$lines"; \
				echo ""; \
			fi; \
		done; \
		echo "$(DIM)Env templates: local.env (optional), go_server/local.env (Go), .env (local copy)$(RESET)"; \
		echo "$(DIM)Default ports: front :3000, go-run :8081$(RESET)"; \
		echo "$(DIM)Detail: make help TARGET=<name>   Validate: make doctor$(RESET)"; \
	fi

## @util:help              Show grouped targets, or `make help TARGET=<name>` for details
##? help  Example: make help TARGET=go-run

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

## @setup:setup              Onboarding: Go, frontend, protobuf, git hooks
##? setup  Run once after cloning to install toolchain dependencies.
##? setup  Example: make setup && cp go_server/local.env go_server/.env && make doctor
setup: setup-go setup-front proto-go setup-hooks
	@echo "Setup complete. Run 'make doctor' to validate the environment."

## @setup:setup-hooks         Point git at .githooks and mark hook scripts executable
##? setup-hooks  Sets core.hooksPath=.githooks. AI hooks stay off unless COMICS_HOOKS_AI=1.
setup-hooks:
	@git config core.hooksPath .githooks
	@chmod +x .githooks/pre-commit .githooks/prepare-commit-msg .githooks/generate_docs.sh .githooks/lint_ai.sh
	@echo "Git hooks: .githooks (see .githooks/README.md)"

## @setup:setup-go            Install and tidy Go module dependencies
setup-go:
	@echo "Go setup..."
	$(MAKE) -C go_server setup-go

## @setup:setup-front          Install frontend dependencies with npm ci
setup-front:
	@echo "Frontend setup..."
	npm ci

# ---------------------------------------------------------------------------
# Full-stack dev (Go + Vite)
# ---------------------------------------------------------------------------

## @util:dev-up                 Start Go API + frontend in background (.dev/logs)
##? dev-up  Ensures go_server/.env; starts Docker Mongo when DB_ADDR is local
##? dev-up  URLs: http://localhost:3000/comics/  API https://localhost:8081
dev-up:
	@./dev/stack.sh up

## @util:dev-down               Stop dev-up stack; bounded graceful stop + WAL checkpoint
##? dev-down  Env: DEV_SHUTDOWN_SEC (default 90), DEV_CHECKPOINT_SEC (default 180, 0=skip)
dev-down:
	@./dev/stack.sh down

## @util:dev-status             Show dev-up PIDs and health hints
dev-status:
	@./dev/stack.sh status

## @util:dev-logs               Tail Go + frontend logs from dev-up
dev-logs:
	@./dev/stack.sh logs

## @util:dev-restart            dev-down then dev-up
dev-restart: dev-down
	@sleep 1
	@$(MAKE) dev-up

# ---------------------------------------------------------------------------
# Frontend
# ---------------------------------------------------------------------------

## @frontend:front-dev         Start Vite dev server (http://localhost:3000)
front-dev:
	npm run start

## @frontend:front-dev-https   Start Vite HTTPS dev server (requires ./tls/comics.crt + .key)
front-dev-https:
	npm run start:dev

## @frontend:front-build        Typecheck and production-build frontend to build/
front-build:
	npm run build

## @frontend:front-test         Run Vitest test suite once
front-test:
	npm run test

## @deploy:front-deploy         Build and publish frontend to GitHub Pages
front-deploy:
	npm run deploy

# ---------------------------------------------------------------------------
# Go backend
# ---------------------------------------------------------------------------

## @go:go-run                  Start Go REST server with air hot reload (port 8081)
go-run:
	$(MAKE) -C go_server go-run

## @go:go-scrape               Run one native Go scrape pass (no HTTP server)
go-scrape:
	$(MAKE) -C go_server go-scrape

## @go:go-test                 Run all Go tests
go-test:
	$(MAKE) -C go_server go-test

## @go:perf-catalog            Opt-in catalog benchmarks against verified DB snapshot
##? perf-catalog  Uses PERF_SNAPSHOT (default db/backups/catalog-perf-snapshot.db), never the live writer without copy
perf-catalog:
	@set -euo pipefail; \
	snapshot="$(PERF_SNAPSHOT)"; \
	if [ ! -f "$$snapshot" ]; then \
		mkdir -p db/backups; \
		cp "$(DB_FILE)" "$$snapshot"; \
	fi; \
	result=$$(sqlite3 "$$snapshot" "PRAGMA integrity_check;"); \
	test "$$result" = "ok" || (echo "perf snapshot integrity failed: $$result" && exit 1); \
	snapshot_abs=$$(cd "$$(dirname "$$snapshot")" && pwd)/$$(basename "$$snapshot"); \
	echo "Using perf snapshot $$snapshot_abs"; \
	$(MAKE) -C go_server perf-catalog COMICS_PERF_DB="$$snapshot_abs"

## @go:perf-backend            Opt-in backend perf (rebuild FTS + strict budgets when COMICS_PERF_STRICT=1)
perf-backend:
	@set -euo pipefail; \
	snapshot="$(PERF_SNAPSHOT)"; \
	if [ ! -f "$$snapshot" ]; then \
		mkdir -p db/backups; \
		cp "$(DB_FILE)" "$$snapshot"; \
	fi; \
	result=$$(sqlite3 "$$snapshot" "PRAGMA integrity_check;"); \
	test "$$result" = "ok" || (echo "perf snapshot integrity failed: $$result" && exit 1); \
	snapshot_abs=$$(cd "$$(dirname "$$snapshot")" && pwd)/$$(basename "$$snapshot"); \
	echo "Using perf snapshot $$snapshot_abs"; \
	COMICS_SQLITE_PATH="$$snapshot_abs" $(MAKE) -C go_server db-rebuild-catalog-search; \
	$(MAKE) -C go_server perf-backend COMICS_PERF_DB="$$snapshot_abs"

## @go:contract-test           Run HTTP contract tests for comics REST parity
contract-test:
	$(MAKE) -C go_server contract-test

## @go:go-vet                  Run go vet static analysis
go-vet:
	$(MAKE) -C go_server go-vet

## @go:go-build                Build Go server and Postgres importer binaries
go-build:
	$(MAKE) -C go_server go-build

## @go:go-migrate-up           Run Go SQL migrations forward
go-migrate-up:
	$(MAKE) -C go_server go-migrate-up

## @go:go-migrate-down         Roll back Go SQL migrations
go-migrate-down:
	$(MAKE) -C go_server go-migrate-down

## @go:go-import-postgres      Import db/comics.db into configured Postgres target
go-import-postgres:
	$(MAKE) -C go_server go-import-postgres

## @go:go-dedupe                 Audit, export, and validate discovery repair plans
go-dedupe:
	$(MAKE) -C go_server go-dedupe

## @database:go-db-maint          Offline DB maintenance (backup first; see go-db-maint-help)
go-db-maint:
	$(MAKE) -C go_server go-db-maint ARGS="$(ARGS)"

## @go:mongo-up                 Start local MongoDB for Go auth (docker compose)
mongo-up:
	$(MAKE) -C go_server mongo-up

## @go:mongo-init               Create local Mongo user esteb (password: localdev)
mongo-init:
	$(MAKE) -C go_server mongo-init

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------

## @database:db-audit-discovery Read-only discovery integrity summary
db-audit-discovery:
	$(MAKE) -C go_server go-dedupe ARGS="--audit --db ../$(DB_FILE)"

## @database:db-export-discovery-plan Export an unapproved repair plan
db-export-discovery-plan:
	@mkdir -p "$$(dirname "$(DISCOVERY_PLAN)")"
	$(MAKE) -C go_server go-dedupe ARGS="--export-plan ../$(DISCOVERY_PLAN) --db ../$(DB_FILE)"

## @database:db-validate-discovery-plan Validate plan against current read-only audit
db-validate-discovery-plan:
	@test -f "$(DISCOVERY_PLAN)" || (echo "Plan not found: $(DISCOVERY_PLAN)"; exit 2)
	$(MAKE) -C go_server go-dedupe ARGS="--validate-plan ../$(DISCOVERY_PLAN) --db ../$(DB_FILE)"

## @database:db-apply-discovery-plan Apply reviewed repairs offline (CONFIRM=reviewed, DISCOVERY_BACKUP, group selector)
db-apply-discovery-plan:
	@test "$(CONFIRM)" = "reviewed" || (echo "Set CONFIRM=reviewed"; exit 2)
	@test -n "$(DISCOVERY_BACKUP)" || (echo "Set DISCOVERY_BACKUP"; exit 2)
	@test -n "$(DISCOVERY_GROUP)" -o "$(DISCOVERY_ALL_APPROVED)" = "1" || (echo "Set DISCOVERY_GROUP or DISCOVERY_ALL_APPROVED=1"; exit 2)
	@selector=""; \
	if [ -n "$(DISCOVERY_GROUP)" ]; then selector="--group-id $(DISCOVERY_GROUP)"; fi; \
	if [ "$(DISCOVERY_ALL_APPROVED)" = "1" ]; then selector="--all-approved"; fi; \
	COMICS_SQLITE_PATH="../$(DB_FILE)" $(MAKE) -C go_server go-db-maint \
	  ARGS="--apply-discovery-plan ../$(DISCOVERY_PLAN) --backup ../$(DISCOVERY_BACKUP) --confirm-reviewed $$selector"

## @database:db-audit-discovery-live Opt-in live catalog audit (no writes)
db-audit-discovery-live:
	cd go_server && AUDIT_DISCOVERY=1 COMICS_SQLITE_PATH=../$(DB_FILE) \
	  go test ./cmd/dedupe -run TestAuditLiveDiscoveryIntegrity -count=1 -v

## @database:db-backup         SQLite backup file under db/backups/
db-backup:
	@mkdir -p db/backups
	@ts=$$(date +%Y%m%d_%H%M%S); \
	dest="$(BACKUP_FILE)"; \
	if [ -z "$$dest" ]; then dest="db/backups/comics_backup_$${ts}.db"; fi; \
	if [ -e "$$dest" ]; then echo "Backup already exists: $$dest"; exit 2; fi; \
	sqlite3 "$(DB_FILE)" ".backup '$${dest}'"; \
	result=$$(sqlite3 -readonly "$$dest" "PRAGMA integrity_check;"); \
	test "$$result" = "ok"; \
	echo "Backup: $${dest} (integrity=ok)"

## @database:db-restore         Restore SQLite from BACKUP after API stop (CONFIRM_OFFLINE=yes)
db-restore:
	@test "$(CONFIRM_OFFLINE)" = "yes" || (echo "Set CONFIRM_OFFLINE=yes after stopping the API"; exit 2)
	@test -n "$(BACKUP)" || (echo "Set BACKUP=db/backups/<file>.db"; exit 2)
	@test -f "$(BACKUP)" || (echo "Backup not found: $(BACKUP)"; exit 1)
	@result=$$(sqlite3 -readonly "$(BACKUP)" "PRAGMA integrity_check;"); \
		test "$$result" = "ok" || (echo "Backup integrity failed: $$result" && exit 1)
	@trap 'rm -f "$(DB_FILE).restore"' EXIT; \
		cp "$(BACKUP)" "$(DB_FILE).restore"; \
		rm -f "$(DB_FILE)-wal" "$(DB_FILE)-shm"; \
		mv "$(DB_FILE).restore" "$(DB_FILE)"; \
		echo "Restored $(DB_FILE) from $(BACKUP)"

## @database:db-check           Run PRAGMA integrity_check on comics.db
##? db-check  Env: DB_FILE (default db/comics.db)
db-check:
	@if [ ! -f "$(DB_FILE)" ]; then \
		echo "Database not found: $(DB_FILE)"; \
		exit 1; \
	fi
	@result=$$(sqlite3 "$(DB_FILE)" "PRAGMA integrity_check;"); \
	echo "$(DB_FILE): $$result"; \
	test "$$result" = "ok"

## @database:db-checkpoint-wal  Flush WAL into comics.db (stop API first; large WAL may take minutes)
##? db-checkpoint-wal  Env: DB_FILE (default db/comics.db). Same as graceful shutdown checkpoint.
db-checkpoint-wal:
	@if [ ! -f "$(DB_FILE)" ]; then \
		echo "Database not found: $(DB_FILE)"; \
		exit 1; \
	fi
	@set -euo pipefail; \
	attempt=0; \
	max=30; \
	while [ $$attempt -lt $$max ]; do \
		attempt=$$((attempt + 1)); \
		if lsof "$(DB_FILE)" >/dev/null 2>&1; then \
			echo "Waiting for processes to release $(DB_FILE) (attempt $$attempt/$$max)..."; \
			sleep 2; \
			continue; \
		fi; \
		echo "Checkpointing WAL for $(DB_FILE) (TRUNCATE, attempt $$attempt)..."; \
		result=$$(sqlite3 "$(DB_FILE)" "PRAGMA wal_checkpoint(TRUNCATE);"); \
		echo "PRAGMA wal_checkpoint: $$result"; \
		busy=$${result%%|*}; \
		if [ "$$busy" = "0" ]; then \
			ls -lh "$(DB_FILE)" "$(DB_FILE)-wal" 2>/dev/null || ls -lh "$(DB_FILE)"; \
			exit 0; \
		fi; \
		sleep 2; \
	done; \
	echo "WAL checkpoint did not complete (database busy). Stop the Go API and retry."; \
	exit 1

## @database:db-freeze           Stop API holders, checkpoint WAL, verify vs last git commit
##? db-freeze  Runs db-checkpoint-wal then prints comics.db size vs HEAD (git add db/comics.db to record)
db-freeze:
	@$(MAKE) db-checkpoint-wal
	@set -euo pipefail; \
	head_bytes=$$(git show HEAD:"$(DB_FILE)" 2>/dev/null | wc -c | tr -d ' '); \
	work_bytes=$$(wc -c < "$(DB_FILE)" | tr -d ' '); \
	echo "comics.db: HEAD=$${head_bytes} bytes, working tree=$${work_bytes} bytes"; \
	if [ "$$head_bytes" = "$$work_bytes" ]; then \
		echo "No size change vs last commit (WAL may already be merged)."; \
	else \
		echo "Database changed — commit with: git add $(DB_FILE) && git commit -m '...'"; \
	fi; \
	git status --short "$(DB_FILE)"

## @database:db-reindex          Repair SQLite index corruption (checkpoint + REINDEX + VACUUM)
##? db-reindex  Env: DB_FILE (default db/comics.db)
db-reindex:
	@if [ ! -f "$(DB_FILE)" ]; then \
		echo "Database not found: $(DB_FILE)"; \
		exit 1; \
	fi
	@sqlite3 "$(DB_FILE)" "PRAGMA wal_checkpoint(TRUNCATE); REINDEX; VACUUM; PRAGMA integrity_check;"

# ---------------------------------------------------------------------------
# Protobuf
# ---------------------------------------------------------------------------

PROTO_DIR := proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
JS_OUT := src/frontend/pb
GO_OUT := go_server/pkg/pb
GO_PROTO_OUT := $(GO_OUT)
GO_SERVICE_OUT := $(GO_OUT)
PROTOC := protoc

## @proto:proto-go              Generate Go protobuf bindings
proto-go:
	@echo "Generating Go Protobuf files..."
	@mkdir -p $(GO_PROTO_OUT) $(GO_SERVICE_OUT)
	$(PROTOC) -I$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go_opt=paths=source_relative \
		--validate_out="lang=go,paths=source_relative:$(GO_OUT)" \
		--go-grpc_out=$(GO_OUT) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)

## @proto:proto-js              Generate TypeScript protobuf bindings
proto-js:
	@echo "Generating JavaScript Protobuf files..."
	$(PROTOC) -I$(PROTO_DIR) \
		--plugin=./node_modules/.bin/protoc-gen-ts_proto \
		--ts_proto_out=$(JS_OUT) \
		--ts_proto_opt=snakeToCamel=false \
		$(PROTO_FILES)

# ---------------------------------------------------------------------------
# Quality
# ---------------------------------------------------------------------------

## @quality:check-go             Run go-test, contract-test, go-vet, go-build, and golangci-lint
check-go: go-test contract-test go-vet go-build
	$(MAKE) -C go_server lint

## @quality:check-front          Run front-test and front-build
check-front: front-test front-build

## @quality:check                Run all release-path checks (Go + frontend)
check:
	@failed=0; \
	echo "== check-go =="; \
	$(MAKE) check-go || failed=1; \
	echo ""; \
	echo "== check-front =="; \
	$(MAKE) check-front || failed=1; \
	echo ""; \
	if [ $$failed -ne 0 ]; then \
		echo "check FAILED"; \
		exit 1; \
	fi; \
	echo "check passed"

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------

## @docker:docker-build         Build Go API Docker image (tag: comics-go-api)
docker-build:
	docker build -t comics-go-api -f Dockerfile .

## @docker:docker-run           Start API + Mongo via docker compose (port 8081)
docker-run:
	docker compose up --build comics_server

## @docker:docker-dev           Foreground compose stack (API + Mongo, SQLite in db/)
docker-dev:
	docker compose up --build

## @docker:docker-down          Stop compose stack
docker-down:
	docker compose down

# ---------------------------------------------------------------------------
# Utility
# ---------------------------------------------------------------------------

## @util:doctor                Validate local environment (node_modules, env files, DB)
doctor:
	@failed=0; \
	echo "Environment check:"; \
	if [ -d node_modules ]; then echo "  [ok] node_modules/"; else echo "  [!!] missing node_modules/ — run make setup-front"; failed=1; fi; \
	if [ -f go_server/.env ] || [ -f go_server/local.env ]; then echo "  [ok] Go env template"; else echo "  [??] no go_server/.env — needed for go-run"; fi; \
	if [ -f tls/comics.crt ] && [ -f tls/comics.key ]; then echo "  [ok] TLS certs for HTTPS dev"; else echo "  [..] no TLS certs — front-dev-https / go-run HTTPS unavailable"; fi; \
	if [ -f "$(DB_FILE)" ]; then \
		result=$$(sqlite3 "$(DB_FILE)" "PRAGMA integrity_check;" 2>/dev/null || echo "error"); \
		if [ "$$result" = "ok" ]; then echo "  [ok] $(DB_FILE) integrity"; else echo "  [!!] $(DB_FILE) integrity: $$result"; failed=1; fi; \
	else echo "  [??] $(DB_FILE) not found"; fi; \
	exit $$failed

## @util:status                 Show dev stack state or default URLs
status:
	@if [ -f .dev/pids/go.pid ] || [ -f .dev/pids/front.pid ]; then \
		./dev/stack.sh status; \
	else \
		echo "Dev stack not running (use: make dev-up)"; \
		echo ""; \
		echo "  make dev-up     start API + frontend in background"; \
		echo "  make dev-down   stop background dev stack"; \
		echo "  make go-run     foreground Go API only (air + debugger)"; \
		echo "  make front-dev  foreground Vite only"; \
	fi

## @util:clean                  Remove generated protobuf files and __pycache__
clean:
	@echo "Cleaning generated files..."
	@rm -rf $(GO_PROTO_OUT)/*.pb.go
	@rm -rf $(GO_SERVICE_OUT)/*.pb.go
	find . -type d -name "__pycache__" -exec rm -r {} + 2>/dev/null || true
