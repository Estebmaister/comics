prepare:
	go get -u ./...
	go mod tidy
	go install github.com/air-verse/air@latest						## local dev server
	go install github.com/swaggo/swag/cmd/swag@latest			## API docs
	go install github.com/mfridman/tparse@latest 					## tests formatter
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest		## sql generator

sqlc:
	sqlc generate

swag:
	swag fmt & swag i -g ./cmd/server/main.go

test:
	set -o pipefail && go test -cover -race -v ./... -json | tparse -all -follow

run:
	air server

migration:
	go build -o tmp/migrate ./cmd/migrate/
	./tmp/migrate up