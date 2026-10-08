.DEFAULT_GOAL := help

# Define phony targets (targets that don't represent actual files)
.PHONY: help setup setup-venv setup-venv-clean setup-shell setup-py setup-go setup-front \
				deps-py-upgrade front-dev front-dev-https front-build front-test front-deploy \
				py-server py-scrape py-daemon py-daemon-stop \
				go-run go-scrape go-test go-vet go-build go-migrate-up go-migrate-down go-import-postgres mongo-up mongo-init \
				db-backup db-restore db-check db-repair-identity db-audit-covers \
				py-test check check-go check-front check-py \
				proto-py proto-go proto-js \
				docker-build docker-run docker-dev clean doctor status

# Enable running multiple commands in a recipe using a single shell
.ONESHELL:

# Virtual environment configuration
VENV_DIR := comics_env
ACT_VENV := . ./$(VENV_DIR)/bin/activate
PYTHONPATH := src
DB_FILE ?= src/db/comics.db
REPAIR_FLAGS := $(if $(APPLY),--apply,)

# Colors for help text
CYAN := \033[36m
DIM := \033[2m
BOLD := \033[1m
RESET := \033[0m

# Per-target detail: make help TARGET=py-server
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
		for section in setup frontend python go database proto quality deploy docker util; do \
			case "$$section" in \
				setup) label="Setup";; \
				frontend) label="Frontend";; \
				python) label="Python";; \
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
		echo "$(DIM)Env templates: local.env (Python/Render), go_server/local.env (Go), .env (local copy)$(RESET)"; \
		echo "$(DIM)Default ports: front :3000, py-server :5001, go-run :8081$(RESET)"; \
		echo "$(DIM)Detail: make help TARGET=<name>   Validate: make doctor$(RESET)"; \
	fi

## @util:help              Show grouped targets, or `make help TARGET=<name>` for details
##? help  Example: make help TARGET=py-server

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

## @setup:setup              Full onboarding: venv, Python, Go, frontend, protobuf, git hooks
##? setup  Run once after cloning to install all toolchain dependencies.
##? setup  Example: make setup && cp local.env .env && make doctor
setup: setup-venv setup-py setup-go setup-front proto-py proto-go
	@chmod +x .githooks/pre-commit
	@echo "Setup complete. Run 'make doctor' to validate the environment."

$(VENV_DIR)/touchfile: requirements.txt
	test -d "$(VENV_DIR)" || python3 -m venv "$(VENV_DIR)"
	$(ACT_VENV) && pip install --upgrade --requirement requirements.txt
	touch "$(VENV_DIR)/touchfile"

## @setup:setup-venv         Create Python virtual environment from requirements.txt
##? setup-venv  Creates comics_env/ and installs Python dependencies.
setup-venv: $(VENV_DIR)/touchfile

## @setup:setup-venv-clean    Remove the Python virtual environment
setup-venv-clean:
	rm -rf $(VENV_DIR)

## @setup:setup-shell         Print virtual environment activation command
setup-shell:
	@echo "Run '$(ACT_VENV)' to activate the virtual environment."

## @setup:setup-py            Install Python dependencies into the virtualenv
setup-py:
	@echo "Python setup..."
	$(ACT_VENV) && pip install -r requirements.txt

## @setup:setup-go            Install and tidy Go module dependencies
setup-go:
	@echo "Go setup..."
	$(MAKE) -C go_server setup-go

## @setup:setup-front          Install frontend dependencies with npm ci
setup-front:
	@echo "Frontend setup..."
	npm ci

## @setup:deps-py-upgrade      Upgrade Python deps and rewrite requirements.txt
deps-py-upgrade:
	$(ACT_VENV) && \
	cat requirements.txt | cut -f1 -d= | xargs pip install -U && \
	pip freeze > requirements.txt

# ---------------------------------------------------------------------------
# Frontend
# ---------------------------------------------------------------------------

## @frontend:front-dev         Start Vite dev server (http://localhost:3000)
##? front-dev  Port: 3000
##? front-dev  Env: VITE_API_SERVER=/api (default proxy to Go :8081)
##? front-dev  Example: make front-dev
front-dev:
	npm run start

## @frontend:front-dev-https   Start Vite HTTPS dev server (requires ./tls/comics.crt + .key)
##? front-dev-https  Port: 3000 (HTTPS)
##? front-dev-https  Requires TLS files in ./tls/
front-dev-https:
	npm run start:dev

## @frontend:front-build        Typecheck and production-build frontend to build/
front-build:
	npm run build

## @frontend:front-test         Run Vitest test suite once
front-test:
	npm run test

## @deploy:front-deploy         Build and publish frontend to GitHub Pages
##? front-deploy  Runs predeploy build, then gh-pages push to build/
front-deploy:
	npm run deploy

# ---------------------------------------------------------------------------
# Python backend
# ---------------------------------------------------------------------------

## @python:py-server           Start Python Flask/gevent API (port 5001)
##? py-server  Port: 5001 (HTTPS in dev when ./tls/comics.crt exists)
##? py-server  Env: PRODUCTION, DEBUG, DB_ENGINE, PORT, DB_FILE
##? py-server  Example: cp local.env .env && make py-server
py-server:
	$(ACT_VENV) && python3 src/__main__.py server

## @python:py-scrape            Deprecated alias for native Go scrape
py-scrape:
	@echo "$(DIM)py-scrape -> go-scrape (Python scrapers removed)$(RESET)"
	$(MAKE) go-scrape

## @python:py-daemon            Run legacy Flask server in background (./server.pid)
##? py-daemon  Logs: ./output.log  Stop with: make py-daemon-stop
##? py-daemon  For API+periodic scrape use: make go-run (SCRAPE_INTERVAL default 10m)
py-daemon:
	@if [ -f ./server.pid ]; then \
		echo "Daemon already running. Use 'make py-daemon-stop' to stop it."; \
		exit 1; \
	fi
	@echo "Starting detached py-server; logs -> ./output.log (scraping: make go-run)"
	$(ACT_VENV) && (python3 src/__main__.py server > ./output.log 2>&1 & echo $$! > ./server.pid)
	@echo "PID $$(cat ./server.pid) saved to ./server.pid"

## @python:py-daemon-stop       Stop background server+scraper using ./server.pid
py-daemon-stop:
	@if [ -f ./server.pid ]; then \
		PID=$$(cat ./server.pid); \
		SPIN='|/-\\'; \
		i=0; \
		while ps -p $$PID > /dev/null 2>&1; do \
			kill $$PID > /dev/null 2>&1; \
			printf "\rStopping process $$PID... %s" $$(echo $$SPIN | cut -c $$(($$i % 4 + 1))); \
			i=$$((i + 1)); \
			sleep 0.3; \
		done; \
		echo "\nProcess $$PID stopped."; \
		rm ./server.pid; \
	else \
		echo "No PID file found."; \
	fi

# ---------------------------------------------------------------------------
# Go backend
# ---------------------------------------------------------------------------

## @go:go-run                  Start Go REST server with air hot reload (port 8081)
##? go-run  Port: 8081 (HTTPS when ../tls/comics.crt exists)
##? go-run  Env: go_server/.env from go_server/local.env
go-run:
	$(MAKE) -C go_server go-run

## @go:go-scrape               Run one native Go scrape pass (no HTTP server)
go-scrape:
	$(MAKE) -C go_server go-scrape

## @go:go-test                 Run all Go tests
go-test:
	$(MAKE) -C go_server go-test

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

## @go:go-import-postgres      Import src/db/comics.db into configured Postgres target
##? go-import-postgres  Configure COMICS_POSTGRES_URL in go_server/.env
go-import-postgres:
	$(MAKE) -C go_server go-import-postgres

## @go:mongo-up                 Start local MongoDB for Go auth (docker compose)
mongo-up:
	$(MAKE) -C go_server mongo-up

## @go:mongo-init               Create local Mongo user esteb (password: localdev)
mongo-init:
	$(MAKE) -C go_server mongo-init

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------

## @database:db-backup         Export SQLite comics to timestamped JSON in src/db/backups/
db-backup:
	$(ACT_VENV) && env PYTHONPATH=$(PYTHONPATH) \
	python3 -c 'from db.backup_db import backup_database; backup_database()'

## @database:db-restore         Rebuild SQLite from src/db/comics.json
db-restore:
	$(ACT_VENV) && env PYTHONPATH=$(PYTHONPATH) \
	python3 -c 'from db.repopulate_db import main; main()'

## @database:db-check           Run PRAGMA integrity_check on comics.db
##? db-check  Env: DB_FILE (default src/db/comics.db)
db-check:
	@if [ ! -f "$(DB_FILE)" ]; then \
		echo "Database not found: $(DB_FILE)"; \
		exit 1; \
	fi
	@result=$$(sqlite3 "$(DB_FILE)" "PRAGMA integrity_check;"); \
	echo "$(DB_FILE): $$result"; \
	test "$$result" = "ok"

## @database:db-repair-identity Audit/repair duplicate identity_key groups (dry-run default)
##? db-repair-identity  Apply merges: make db-repair-identity APPLY=1
db-repair-identity:
	$(ACT_VENV) && env PYTHONPATH=$(PYTHONPATH) \
	python3 -P src/db/repair_identity_duplicates.py $(REPAIR_FLAGS) $(ARGS)

## @database:db-audit-covers    Probe cover URLs and optionally mark failures invisible
##? db-audit-covers  Apply updates: make db-audit-covers ARGS="--apply"
db-audit-covers:
	$(ACT_VENV) && env PYTHONPATH=$(PYTHONPATH) \
	python3 src/db/audit_cover_visibility.py $(ARGS)

# ---------------------------------------------------------------------------
# Protobuf
# ---------------------------------------------------------------------------

PROTO_DIR := proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
PYTHON_OUT := src/pb
PYTHON_PROTO_OUT := $(PYTHON_OUT)
PYTHON_SERVICE_OUT := $(PYTHON_OUT)
JS_OUT := src/frontend/pb
GO_OUT := go_server/pkg/pb
GO_PROTO_OUT := $(GO_OUT)
GO_SERVICE_OUT := $(GO_OUT)
PROTOC := protoc
PYTHON_GRPC := python3 -m grpc_tools.protoc

## @proto:proto-py              Generate Python gRPC/protobuf bindings
proto-py:
	@echo "Installing Python Protobuf dependencies..."
	$(ACT_VENV) && pip install grpcio==1.70.0 grpcio-tools==1.70.0
	@echo "Generating Python Protobuf files..."
	@mkdir -p $(PYTHON_PROTO_OUT) $(PYTHON_SERVICE_OUT)
	$(ACT_VENV) && \
	$(PYTHON_GRPC) -I$(PROTO_DIR) \
		--python_out=$(PYTHON_OUT) \
		--pyi_out=$(PYTHON_OUT) \
		--grpc_python_out=$(PYTHON_OUT) \
		$(PROTO_FILES)
	@touch $(PYTHON_PROTO_OUT)/__init__.py $(PYTHON_SERVICE_OUT)/__init__.py

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

## @quality:py-test             Run Python pytest suite (PYTHONPATH=src)
py-test:
	$(ACT_VENV) && env PYTHONPATH=$(PYTHONPATH) python3 -m pytest test/*_test.py -v

## @quality:check-go             Run go-test, contract-test, go-vet, and go-build
check-go: go-test contract-test go-vet go-build

## @quality:check-front          Run front-test and front-build
check-front: front-test front-build

## @quality:check-py             Alias for py-test
check-py: py-test

## @quality:check                Run all release-path checks (Go + frontend + Python)
##? check  Pre-push gate: check-go, check-front, check-py
check:
	@failed=0; \
	echo "== check-go =="; \
	$(MAKE) check-go || failed=1; \
	echo ""; \
	echo "== check-front =="; \
	$(MAKE) check-front || failed=1; \
	echo ""; \
	echo "== check-py =="; \
	$(MAKE) check-py || failed=1; \
	echo ""; \
	if [ $$failed -ne 0 ]; then \
		echo "check FAILED"; \
		exit 1; \
	fi; \
	echo "check passed"

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------

## @docker:docker-build         Build comic-tracker Docker image
docker-build:
	docker build -t comic-tracker .

## @docker:docker-run           Run comic-tracker container on port 5001
docker-run:
	docker run -p 5001:5001 comic-tracker

## @docker:docker-dev           Run container with src bind-mount and file polling
docker-dev:
	docker run -e CHOKIDAR_USEPOLLING=true -v ${PWD}/src/:/code/src/ -p 5001:5001 comic-tracker

# ---------------------------------------------------------------------------
# Utility
# ---------------------------------------------------------------------------

## @util:doctor                Validate local environment (venv, node_modules, env files)
##? doctor  Exits non-zero when critical setup pieces are missing
doctor:
	@failed=0; \
	echo "Environment check:"; \
	if [ -d "$(VENV_DIR)" ]; then echo "  [ok] $(VENV_DIR)/"; else echo "  [!!] missing $(VENV_DIR)/ — run make setup-venv"; failed=1; fi; \
	if [ -d node_modules ]; then echo "  [ok] node_modules/"; else echo "  [!!] missing node_modules/ — run make setup-front"; failed=1; fi; \
	if [ -f .env ] || [ -f local.env ]; then echo "  [ok] Python env template (.env or local.env)"; else echo "  [??] no .env or local.env — copy local.env to .env for py-server"; fi; \
	if [ -f go_server/.env ] || [ -f go_server/local.env ]; then echo "  [ok] Go env template"; else echo "  [??] no go_server/.env — needed for go-run/postgres workflows"; fi; \
	if [ -f tls/comics.crt ] && [ -f tls/comics.key ]; then echo "  [ok] TLS certs for HTTPS dev"; else echo "  [..] no TLS certs — front-dev-https and py-server HTTPS unavailable"; fi; \
	if [ -f "$(DB_FILE)" ]; then \
		result=$$(sqlite3 "$(DB_FILE)" "PRAGMA integrity_check;" 2>/dev/null || echo "error"); \
		if [ "$$result" = "ok" ]; then echo "  [ok] $(DB_FILE) integrity"; else echo "  [!!] $(DB_FILE) integrity: $$result"; failed=1; fi; \
	else echo "  [??] $(DB_FILE) not found"; fi; \
	exit $$failed

## @util:status                 Show daemon PID, log path, and default dev URLs
status:
	@echo "Dev URLs:"; \
	echo "  frontend   http://localhost:3000"; \
	echo "  py-server  https://localhost:5001"; \
	echo "  go-run     https://localhost:8081"; \
	echo ""; \
	if [ -f ./server.pid ]; then \
		PID=$$(cat ./server.pid); \
		if ps -p $$PID > /dev/null 2>&1; then \
			echo "py-daemon: running (PID $$PID, log ./output.log)"; \
		else \
			echo "py-daemon: stale PID file (PID $$PID not running)"; \
		fi; \
	else \
		echo "py-daemon: not running (no ./server.pid)"; \
	fi

## @util:clean                  Remove generated protobuf files and __pycache__
clean:
	@echo "Cleaning generated files..."
	@rm -rf $(PYTHON_PROTO_OUT)/*_pb2*.py
	@rm -rf $(PYTHON_SERVICE_OUT)/*_pb2*.py
	@rm -rf $(GO_PROTO_OUT)/*.pb.go
	@rm -rf $(GO_SERVICE_OUT)/*.pb.go
	find . -type d -name "__pycache__" -exec rm -r {} + 2>/dev/null || true
