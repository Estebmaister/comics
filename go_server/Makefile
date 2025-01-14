swag:
	swag fmt & swag i -g ./cmd/server/main.go

swaginstall:
	go install github.com/swaggo/swag/cmd/swag@latest

test:
	go test -cover -race -v -timeout 20s ./...

prepare:
	go get -u ./...
	go mod tidy
	go install github.com/air-verse/air@latest

run:
	air server

migration:
	go build -o tmp/migrate ./cmd/migrate/
	./tmp/migrate up