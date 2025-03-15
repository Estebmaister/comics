prepare:
	go get -u ./...
	go mod tidy
	go tool trivy repository .

linters:
	go install golang.org/x/lint/golint@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
# go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
# Checks for style issues and adherence to Go conventions.
	golint -set_exit_status ./...
# Provides advanced static analysis to detect bugs, performance issues, and potential bugs.
	staticcheck ./...
# Static analysis of Go code for vulnerabilities and unsafe patterns.
	gosec ./...
# golangci-lint run # Run all linters

sqlc:
	sqlc generate

swag:
	go tool swag fmt & go tool swag i -g ./cmd/server/main.go

test:
	set -o pipefail && go test -cover -race -v ./... -json | go tool tparse -all -follow

run:
	go tool air server

build:
	go build -o tmp/server ./cmd/server/
	./tmp/server

migration:
	go build -o tmp/migrate ./cmd/migrate/
	./tmp/migrate up