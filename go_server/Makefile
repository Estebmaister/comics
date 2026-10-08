.DEFAULT_GOAL := help

.PHONY: help setup-go setup-tools go-run go-run-plain go-scrape go-test go-vet go-build \
				go-migrate-up go-migrate-down go-import-postgres contract-test check-go \
				mongo-up mongo-down mongo-init sqlc swag lint vuln doctor status clean

.ONESHELL:

TOOL_MOD := tool.go.mod
TOOL_GO := go tool -modfile=$(TOOL_MOD)
TMP_DIR := tmp
SERVER_BIN := $(TMP_DIR)/server
IMPORT_BIN := $(TMP_DIR)/import_comics_postgres
MIGRATE_BIN := $(TMP_DIR)/migrate

CYAN := \033[36m
DIM := \033[2m
BOLD := \033[1m
RESET := \033[0m

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
		echo "$(BOLD)Go server — available targets$(RESET)"; \
		echo ""; \
		for section in setup server database codegen quality util; do \
			case "$$section" in \
				setup) label="Setup";; \
				server) label="Server";; \
				database) label="Database";; \
				codegen) label="Codegen";; \
				quality) label="Quality";; \
				util) label="Utility";; \
			esac; \
			lines=$$(grep "^## @$$section:" $(MAKEFILE_LIST) | sed "s/^## @$$section://" | sed 's/^[ \t]*//' | awk '{desc=$$0; sub(/^[^ \t]+[ \t]+/, "", desc); printf "  %-24s %s\n", $$1, desc}'); \
			if [ -n "$$lines" ]; then \
				echo "$(CYAN)$$label$(RESET)"; \
				echo "$$lines"; \
				echo ""; \
			fi; \
		done; \
		echo "$(DIM)Env template: local.env (copy to .env for local runs)$(RESET)"; \
		echo "$(DIM)Default port: https://localhost:8081$(RESET)"; \
		echo "$(DIM)Detail: make help TARGET=<name>   Validate: make doctor$(RESET)"; \
	fi

## @util:help                Show grouped targets, or `make help TARGET=<name>` for details
##? help  Example: make help TARGET=go-run

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

## @setup:setup-go            Tidy Go module dependencies
setup-go:
	go mod tidy

## @setup:setup-tools         Install and verify dev tools from tool.go.mod
##? setup-tools  Installs air, delve, sqlc, swag, linters, and scanners into module cache.
setup-tools:
	go get -u ./...
	go mod tidy
	go mod verify -modfile=$(TOOL_MOD)
	$(TOOL_GO) dlv version
	@echo "Dev tools ready. Run 'make go-run' for hot reload + debugger."

# ---------------------------------------------------------------------------
# Server
# ---------------------------------------------------------------------------

## @server:go-run             Start API with air hot reload and Delve debugger
##? go-run  Port: 8081 (HTTPS when ../tls/comics.crt exists)
##? go-run  Debugger: 127.0.0.1:2345 (attach from IDE)
##? go-run  Env: .env from local.env
##? go-run  Example: cp local.env .env && make go-run
go-run:
	$(TOOL_GO) air server

## @server:go-run-plain        Start API with air hot reload (no debugger)
go-run-plain:
	$(TOOL_GO) air -c .air.plain.toml server

## @server:go-scrape           Run one native scrape pass (no HTTP server)
go-scrape:
	go run ./cmd/scrape

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------

## @database:mongo-up          Start local MongoDB via docker compose
##? mongo-up  Requires Docker. Exposes localhost:27017
mongo-up:
	docker compose up -d mongodb

## @database:mongo-down        Stop local MongoDB container
mongo-down:
	docker compose stop mongodb

## @database:mongo-init        Create local dev user esteb (password: localdev)
##? mongo-init  Idempotent. Run after mongo-up on a fresh/no-auth Mongo instance.
mongo-init: mongo-up
	@until mongosh --quiet mongodb://localhost:27017/admin --eval "db.adminCommand('ping').ok" >/dev/null 2>&1; do sleep 1; done
	mongosh --quiet mongodb://localhost:27017/admin scripts/mongo-init-local.js

## @database:go-migrate-up    Run SQL migrations forward
go-migrate-up:
	go run ./cmd/migrate up

## @database:go-migrate-down  Roll back SQL migrations
go-migrate-down:
	go run ./cmd/migrate down

## @database:go-import-postgres Import ../src/db/comics.db into configured Postgres target
##? go-import-postgres  Configure COMICS_POSTGRES_URL in .env
go-import-postgres:
	go run ./cmd/import_comics_postgres

# ---------------------------------------------------------------------------
# Codegen
# ---------------------------------------------------------------------------

## @codegen:sqlc              Generate typed SQL from sqlc queries
sqlc:
	$(TOOL_GO) sqlc generate

## @codegen:swag              Regenerate Swagger docs from API annotations
swag:
	$(TOOL_GO) swag fmt
	$(TOOL_GO) swag init -g ./cmd/server/main.go

# ---------------------------------------------------------------------------
# Quality
# ---------------------------------------------------------------------------

## @quality:go-test           Run all Go tests
go-test:
	go test ./...

## @quality:contract-test      Run HTTP contract tests for comics REST parity
contract-test:
	go test ./api/route/... -run Contract -count=1

## @quality:go-vet            Run go vet static analysis
go-vet:
	go vet ./...

## @quality:go-build           Build server and Postgres importer binaries
go-build:
	mkdir -p $(TMP_DIR)
	go build -o $(SERVER_BIN) ./cmd/server
	go build -o $(IMPORT_BIN) ./cmd/import_comics_postgres

## @quality:check-go          Run go-test, contract-test, go-vet, and go-build
check-go: go-test contract-test go-vet go-build

## @quality:lint              Run golint, staticcheck, and golangci-lint
lint:
	$(TOOL_GO) golint -set_exit_status ./...
	$(TOOL_GO) staticcheck ./...
	$(TOOL_GO) golangci-lint run

## @quality:vuln               Run govulncheck, trivy, and gosec security scans
vuln:
	$(TOOL_GO) govulncheck ./...
	$(TOOL_GO) trivy fs .
	$(TOOL_GO) trivy repository --scanners misconfig .
	$(TOOL_GO) gosec -quiet -exclude-dir=pkg/pb ./...

# ---------------------------------------------------------------------------
# Utility
# ---------------------------------------------------------------------------

## @util:doctor               Validate Go server local environment
##? doctor  Exits non-zero when critical setup pieces are missing
doctor:
	@failed=0; \
	echo "Go server environment:"; \
	if command -v go >/dev/null 2>&1; then echo "  [ok] go $$(go version | sed 's/^go version //')"; else echo "  [!!] go not found"; failed=1; fi; \
	if [ -f .env ] || [ -f local.env ]; then echo "  [ok] env template (.env or local.env)"; else echo "  [??] no .env — copy local.env to .env"; fi; \
	if [ -f ../tls/comics.crt ] && [ -f ../tls/comics.key ]; then echo "  [ok] TLS certs for HTTPS dev"; else echo "  [..] no TLS certs — HTTP only unless platform TLS terminates"; fi; \
	if $(TOOL_GO) dlv version >/dev/null 2>&1; then \
		dlv_ver=$$($(TOOL_GO) dlv version 2>/dev/null | head -n1); \
		echo "  [ok] delve ($$dlv_ver)"; \
	else \
		echo "  [!!] delve missing — run make setup-tools"; failed=1; \
	fi; \
	if $(TOOL_GO) air -v >/dev/null 2>&1; then echo "  [ok] air hot reload"; else echo "  [!!] air missing — run make setup-tools"; failed=1; fi; \
	mongo_addr=$$(grep -E '^DB_ADDR=' .env local.env 2>/dev/null | tail -n1 | cut -d= -f2-); \
	if echo "$$mongo_addr" | grep -q 'mongodb+srv://'; then \
		echo "  [ok] mongodb atlas ($$mongo_addr)"; \
	elif nc -z localhost 27017 >/dev/null 2>&1; then \
		echo "  [ok] mongodb localhost:27017"; \
	else \
		echo "  [!!] mongodb not reachable — run make mongo-up or set DB_ADDR to Atlas"; failed=1; \
	fi; \
	exit $$failed

## @util:status               Show default endpoints and debugger attach info
status:
	@echo "Go server URLs:"; \
	echo "  API        https://localhost:8081"; \
	echo "  Swagger    https://localhost:8081/swagger/index.html"; \
	echo "  Health     https://localhost:8081/health"; \
	echo "  Debugger   127.0.0.1:2345 (when using make go-run)"; \
	echo ""; \
	if [ -f .env ]; then echo "Env: .env"; elif [ -f local.env ]; then echo "Env: local.env (copy to .env for runs)"; else echo "Env: none"; fi

## @util:clean                Remove build artifacts in tmp/
clean:
	rm -rf $(TMP_DIR)
