# Define phony targets (targets that don't represent actual files)
.PHONY: activate venv venvclean start deploy server scrape remote stop \
				backup repopulate db_update update-py setup-py setup clean \
				proto-py proto-go migrate-up migrate-down test-front test-py test-go \
				dockerize docker chokidar help

# Enable running multiple commands in a recipe using a single shell
.ONESHELL:

# Virtual environment configuration
VENV_DIR=comics_env
ACT_VENV:=. ./$(VENV_DIR)/bin/activate

# Colors for help text
CYAN := \033[36m
RESET := \033[0m

## help          Display this help message
help:
	@echo "Available targets:"
	@awk '\
		BEGIN { \
			cmd_width = 14; \
		} \
		/^##/ { \
			line = substr($$0, 4); \
			cmd = substr(line, 1, cmd_width); \
			desc = substr(line, cmd_width + 1); \
			gsub(/^[ \t]+/, "", desc); \
			printf "$(CYAN)%-*s$(RESET) %s\n", cmd_width, cmd, desc; \
		}' $(MAKEFILE_LIST)

## venv          Create and setup Python virtual environment
$(VENV_DIR)/touchfile: requirements.txt
	test -d "$(VENV_DIR)" || python3 -m venv "$(VENV_DIR)"
	$(ACT_VENV) && pip install --upgrade --requirement requirements.txt
	touch "$(VENV_DIR)/touchfile"

venv: $(VENV_DIR)/touchfile

## venvclean     Remove the virtual environment
venvclean:
	rm -rf $(VENV_DIR)

## activate      Show virtual environment activation instructions
activate:
	@echo "Run '$(ACT_VENV)' to activate the virtual environment."

## start         Start the frontend development server
start:
	npm run start

## deploy        Deploy the frontend development server
deploy:
	npm run deploy

## server        Start the backend server
server:
	$(ACT_VENV) && python3 src/__main__.py server

## scrape        Run the web scraper
scrape:
	$(ACT_VENV) && python3 src/__main__.py

## remote        Run the web server and scraper, save logs and detach the process
remote:
	@if [ -f ./server.pid ]; then \
		echo "Server is already running. Use 'make stop' to stop it."; \
		exit 1; \
	fi
	@echo "Running detached web server and scraper, logs will be saved to ./output.log"
	$(ACT_VENV) && (python3 src scrape server > ./output.log 2>&1 & echo $$! > ./server.pid)
	@echo "PID $$(cat ./server.pid) saved to ./server.pid"

## stop          Stop the background process using the PID file
stop:
	@if [ -f ./server.pid ]; then \
		PID=$$(cat ./server.pid); \
		SPIN='|/-\\'; \
		i=0; \
		while ps -p $$PID > /dev/null; do \
			kill $$PID > /dev/null 2>&1; \
			printf "\rKilling process $$PID... %s" $$(echo $$SPIN | cut -c $$(($$i % 4 + 1))); \
			i=$$((i + 1)); \
			sleep 0.3; \
		done; \
		echo "\nProcess $$PID stopped. Cleaning up."; \
		rm ./server.pid; \
	else \
		echo "No PID file found."; \
	fi

## backup        Run the web backup
backup:
	$(ACT_VENV) && \
	python3 -c 'from src.db.backup_db import backup_database; backup_database()'

## repopulate    Run the web backup
repopulate:
	$(ACT_VENV) && \
	python3 -c 'from src.db.repopulate_db import main; main()'

## db_update     Run the web backup
db_update:
	$(ACT_VENV) && \
	python3 -c 'from src.db.db_update import main; main()'

## update-py     Update all Python dependencies to latest versions
update-py:
	$(ACT_VENV) && \
	cat requirements.txt | cut -f1 -d= | xargs pip install -U && \
	pip freeze > requirements.txt

## setup-py      Initialize Python environment and dependencies
setup-py:
	@echo "\nPython setup..."
	$(ACT_VENV) && pip install -r requirements.txt

## setup         Initialize both Go and Python environments
setup:
	@echo "Setting up the servers..."
	chmod +x .githooks/pre-commit
	(cd go_server && go mod tidy)
	$(MAKE) setup-py
	$(MAKE) proto-py
	$(MAKE) proto-go

# Protobuf configuration
# Directory containing .proto files
PROTO_DIR := proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)

# Python Protobuf output configuration
PYTHON_OUT := src/pb
PYTHON_PROTO_OUT := $(PYTHON_OUT)
PYTHON_SERVICE_OUT := $(PYTHON_OUT)

# JavaScript Protobuf output configuration
JS_OUT := src/frontend/pb

# Go Protobuf output configuration
GO_OUT := go_server/pkg/pb
GO_PROTO_OUT := $(GO_OUT)
GO_SERVICE_OUT := $(GO_OUT)

# Protobuf compiler and tools configuration
PROTOC := protoc
PYTHON_GRPC := python3 -m grpc_tools.protoc
GO_GRPC := protoc-gen-go-grpc

## proto-py      Generate Python Protobuf files from definitions
proto-py:
	@echo "\nInstalling Python Protobuf dependencies..."
	$(ACT_VENV) && pip install grpcio==1.70.0 grpcio-tools==1.70.0
	@echo "\nGenerating Python Protobuf files..."
	@mkdir -p $(PYTHON_PROTO_OUT)
	@mkdir -p $(PYTHON_SERVICE_OUT)
	$(ACT_VENV) && \
	$(PYTHON_GRPC) -I$(PROTO_DIR) \
		--python_out=$(PYTHON_OUT) \
		--pyi_out=$(PYTHON_OUT) \
		--grpc_python_out=$(PYTHON_OUT) \
		$(PROTO_FILES)
	@touch $(PYTHON_PROTO_OUT)/__init__.py
	@touch $(PYTHON_SERVICE_OUT)/__init__.py

## proto-go      Generate Go Protobuf files from definitions
proto-go:
	@echo "\nGenerating Go Protobuf files..."
	@mkdir -p $(GO_PROTO_OUT)
	@mkdir -p $(GO_SERVICE_OUT)
	$(PROTOC) -I$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go_opt=paths=source_relative \
		--validate_out="lang=go,paths=source_relative:$(GO_OUT)" \
		--go-grpc_out=$(GO_OUT) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)

## proto-js      Generate JavaScript Protobuf files from definitions
proto-js:
	@echo "\nGenerating JavaScript Protobuf files..."
	$(PROTOC) -I$(PROTO_DIR) \
		--plugin=./node_modules/.bin/protoc-gen-ts_proto \
		--ts_proto_out=$(JS_OUT) \
		--ts_proto_opt=snakeToCamel=false \
		$(PROTO_FILES)

## migrate-up    Run database migrations forward
migrate-up:
	go run go_server/cmd/migrate/main.go up

## migrate-down  Roll back database migrations
migrate-down:
	go run go_server/cmd/migrate/main.go down

## test-go       Run all Go tests
test-go:
	go test -v go_server/...

## test-front    Run all Frontend tests
test-front:
	npm run test

## test-py       Run all Python tests
test-py:
	$(ACT_VENV) && env PYTHONPATH=src python3 -m pytest test/*_test.py -v

## clean         Clean up all generated files and caches
clean:
	@echo "Cleaning generated files..."
	@rm -rf $(PYTHON_PROTO_OUT)/*_pb2*.py
	@rm -rf $(PYTHON_SERVICE_OUT)/*_pb2*.py
	@rm -rf $(GO_PROTO_OUT)/*.pb.go
	@rm -rf $(GO_SERVICE_OUT)/*.pb.go
	find . -type d -name "__pycache__" -exec rm -r {} +

## dockerize    Build Docker image
dockerize:
	docker build -t comic-tracker .

## docker       Run Docker container
docker:
	docker run -p 5001:5001 comic-tracker

## chokidar     Run Docker container with chokidar
chokidar:
	docker run -e CHOKIDAR_USEPOLLING=true -v ${PWD}/src/:/code/src/ -p 5001:5001 comic-tracker