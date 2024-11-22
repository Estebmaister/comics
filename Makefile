# Define phony targets (targets that don't represent actual files)
.PHONY: setup proto-python proto-go clean migrate-up migrate-down test help

# Enable running multiple commands in a recipe using a single shell
.ONESHELL:

# Virtual environment configuration
VENV_DIR=comics_env
ACTIVATE_VENV:=. $(VENV_DIR)/bin/activate

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
	$(ACTIVATE_VENV)
	pip3 install --upgrade --requirement requirements.txt
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

## server        Start the backend server
server:
	npm run server

## scrape        Run the web scraper
scrape:
	npm run scrape

# Protobuf configuration
# Directory containing .proto files
PROTO_DIR := proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)

# Python Protobuf output configuration
PYTHON_OUT := src/pb
PYTHON_PROTO_OUT := $(PYTHON_OUT)
PYTHON_SERVICE_OUT := $(PYTHON_OUT)

# Go Protobuf output configuration
GO_OUT := go_server/pb
GO_PROTO_OUT := $(GO_OUT)
GO_SERVICE_OUT := $(GO_OUT)

# Protobuf compiler and tools configuration
PROTOC := protoc
PYTHON_GRPC := python -m grpc_tools.protoc
GO_GRPC := protoc-gen-go-grpc

## update-py     Update all Python dependencies to latest versions
update-py:
	cat requirements.txt | cut -f1 -d= | xargs pip3 install -U
	pip3 freeze > requirements.txt

## setup-py      Initialize Python environment and dependencies
setup-py:
	@echo "Python setup..."
	python3 -m venv comics_env
	$(ACTIVATE_VENV)
	pip3 install -r requirements.txt

## setup         Initialize both Go and Python environments
setup:
	@echo "Setting up the servers..."
	(cd go_server && go mod tidy)
	setup-py proto-py proto-go

## proto-py      Generate Python Protobuf files from definitions
proto-py:
	@echo "Generating Python Protobuf files..."
	$(ACTIVATE_VENV)
	@mkdir -p $(PYTHON_PROTO_OUT)
	@mkdir -p $(PYTHON_SERVICE_OUT)
	$(PYTHON_GRPC) -I$(PROTO_DIR) \
		--python_out=$(PYTHON_OUT) \
		--pyi_out=$(PYTHON_OUT) \
		--grpc_python_out=$(PYTHON_OUT) \
		$(PROTO_FILES)
	@touch $(PYTHON_PROTO_OUT)/__init__.py
	@touch $(PYTHON_SERVICE_OUT)/__init__.py

## proto-go      Generate Go Protobuf files from definitions
proto-go:
	@echo "Generating Go Protobuf files..."
	@mkdir -p $(GO_PROTO_OUT)
	@mkdir -p $(GO_SERVICE_OUT)
	$(PROTOC) -I$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go_opt=paths=source_relative \
		--validate_out="lang=go,paths=source_relative:$(GO_OUT)" \
		--go-grpc_out=$(GO_OUT) \
		--go-grpc_opt=paths=source_relative \
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
