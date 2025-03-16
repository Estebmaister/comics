prepare:
	go get -u ./...
	go mod tidy
	go mod verify -modfile=tool.go.mod
	go get -modfile=tool.go.mod tool

vuln:
# Checks for vulnerabilities in dependencies.
	go tool -modfile=tool.go.mod govulncheck
	go tool -modfile=tool.go.mod trivy fs .
	go tool -modfile=tool.go.mod trivy repository --scanners misconfig .
# Static analysis of Go code for vulnerabilities and unsafe patterns.
	go tool -modfile=tool.go.mod gosec -quiet -exclude-dir=pkg/pb ./...

lint:
# Checks for style issues and adherence to Go conventions.
	go tool -modfile=tool.go.mod golint -set_exit_status ./...
# Provides advanced static analysis to detect bugs, performance issues, and potential bugs.
	go tool -modfile=tool.go.mod staticcheck ./...
# Run combined linters
	go tool -modfile=tool.go.mod golangci-lint run

sqlc:
	go tool -modfile=tool.go.mod sqlc generate

swag:
	go tool -modfile=tool.go.mod swag fmt & \
	go tool -modfile=tool.go.mod swag i -g ./cmd/server/main.go

test:
	set -o pipefail && go test -cover -race -v ./... -json | \
	go tool -modfile=tool.go.mod tparse -all -follow

run:
	go tool -modfile=tool.go.mod air server

build:
	go build -o tmp/server ./cmd/server/
	./tmp/server

migration:
	go build -o tmp/migrate ./cmd/migrate/
	./tmp/migrate up