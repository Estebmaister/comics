.PHONY: setup proto-python proto-go clean migrate-up migrate-down test
.ONESHELL:
VENV_DIR=comics_env
ACTIVATE_VENV:=. $(VENV_DIR)/bin/activate

$(VENV_DIR)/touchfile: requirements.txt
	test -d "$(VENV_DIR)" || python3 -m venv "$(VENV_DIR)"
	$(ACTIVATE_VENV)
	pip3 install --upgrade --requirement requirements.txt
	touch "$(VENV_DIR)/touchfile"

venv: $(VENV_DIR)/touchfile

venvclean:
	rm -rf $(VENV_DIR)

# Protobuf directories
PROTO_DIR := proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)

# Python output directories
PYTHON_OUT := src/pb
PYTHON_PROTO_OUT := $(PYTHON_OUT)
PYTHON_SERVICE_OUT := $(PYTHON_OUT)

# Go output directories
GO_OUT := go_server/pb
GO_PROTO_OUT := $(GO_OUT)
GO_SERVICE_OUT := $(GO_OUT)

# Tools and commands
PROTOC := protoc
PYTHON_GRPC := python -m grpc_tools.protoc
GO_GRPC := protoc-gen-go-grpc

update-py:
	cat requirements.txt | cut -f1 -d= | xargs pip3 install -U
	pip3 freeze > requirements.txt

setup-py:
	@echo "Python setup..."
	python3 -m venv comics_env
	$(ACTIVATE_VENV)
	pip3 install -r requirements.txt

setup:
	@echo "Setting up the servers..."
	(cd go_server && go mod tidy)
	setup-py proto-py proto-go

# Python Protobuf generation
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

# Go Protobuf generation
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

migrate-up:
	go run go_server/cmd/migrate/main.go up

migrate-down:
	go run go_server/cmd/migrate/main.go down

test:
	go test -v go_server/...

clean:
	@echo "Cleaning generated files..."
	@rm -rf $(PYTHON_PROTO_OUT)/*_pb2*.py
	@rm -rf $(PYTHON_SERVICE_OUT)/*_pb2*.py
	@rm -rf $(GO_PROTO_OUT)/*.pb.go
	@rm -rf $(GO_SERVICE_OUT)/*.pb.go
	find . -type d -name "__pycache__" -exec rm -r {} +
