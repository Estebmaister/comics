# Define phony targets (targets that don't represent actual files)
.PHONY: activate venv venvclean start server scrape update-py \
        setup proto-py proto-go clean migrate-up migrate-down test help

# Enable running multiple commands in a recipe using a single shell
.ONESHELL:

# Virtual environment configuration
VENV_DIR=comics_env
ACTIVATE_VENV:=source $(VENV_DIR)/bin/activate

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
	$(ACTIVATE_VENV) && pip install --upgrade --requirement requirements.txt
	touch "$(VENV_DIR)/touchfile"

venv: $(VENV_DIR)/touchfile

## venvclean     Remove the virtual environment
venvclean:
	rm -rf $(VENV_DIR)

## activate      Show virtual environment activation instructions
activate:
	@echo "Run '$(ACTIVATE_VENV)' to activate the virtual environment."

## start         Start the frontend development server
start:
	npm run start

## deploy        Deploy the frontend development server
deploy:
	npm run deploy

## server        Start the backend server
server:
	$(ACTIVATE_VENV) && python3 src/__main__.py server

## scrape        Run the web scraper
scrape:
	$(ACTIVATE_VENV) && python3 src/__main__.py

## backup        Run the web backup
backup:
	$(ACTIVATE_VENV) && \
	python3 -c 'from src.db.backup_db import backup_database; backup_database()'

## repopulate    Run the web backup
repopulate:
	$(ACTIVATE_VENV) && \
	python3 -c 'from src.db.repopulate_db import main; main()'

## db_update     Run the web backup
db_update:
	$(ACTIVATE_VENV) && \
	python3 -c 'from src.db.db_update import main; main()'

## py-test       Run python tests
test-py:
	$(ACTIVATE_VENV) && env PYTHONPATH=src python3 -m pytest test/*_test.py -v

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

## update-py     Update all Python dependencies to latest versions
update-py:
	$(ACTIVATE_VENV) && \
	cat requirements.txt | cut -f1 -d= | xargs pip install -U && \
	pip freeze > requirements.txt

## setup-py      Initialize Python environment and dependencies
setup-py:
	@echo "\nPython setup..."
	$(ACTIVATE_VENV) && pip install -r requirements.txt

## setup         Initialize both Go and Python environments
setup:
	@echo "Setting up the servers..."
	(cd go_server && go mod tidy)
	$(MAKE) setup-py
	$(MAKE) proto-py
	$(MAKE) proto-go

## proto-py      Generate Python Protobuf files from definitions
proto-py:
	@echo "\nGenerating Python Protobuf files..."
	@mkdir -p $(PYTHON_PROTO_OUT)
	@mkdir -p $(PYTHON_SERVICE_OUT)
	$(ACTIVATE_VENV) && \
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

## test          Run all Go tests
test:
	go test -v go_server/...

## clean         Clean up all generated files and caches
clean:
	@echo "Cleaning generated files..."
	@rm -rf $(PYTHON_PROTO_OUT)/*_pb2*.py
	@rm -rf $(PYTHON_SERVICE_OUT)/*_pb2*.py
	@rm -rf $(GO_PROTO_OUT)/*.pb.go
	@rm -rf $(GO_SERVICE_OUT)/*.pb.go
	find . -type d -name "__pycache__" -exec rm -r {} +
