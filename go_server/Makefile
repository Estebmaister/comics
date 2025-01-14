swag:
	swag fmt & swag i -g ./cmd/server/main.go

swaginstall:
	go install github.com/swaggo/swag/cmd/swag@latest

test:
	set -o pipefail && go test -cover -race -v ./... -json | tparse -all -follow

prepare:
	go get -u ./...
	go mod tidy
	go install github.com/air-verse/air@latest
	go install github.com/mfridman/tparse@latest

run:
	air server

migration:
	go build -o tmp/migrate ./cmd/migrate/
	./tmp/migrate up