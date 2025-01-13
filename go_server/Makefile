swag:
	swag fmt & swag i -g ./cmd/server/main.go

swaginstall:
	go get -u github.com/swaggo/swag/cmd/swag@latest

test:
	go test -cover -race -v -timeout 20s ./...

prepare:
	go mod tidy
	go mod vendor
	go get -u github.com/air-verse/air@latest

run:
	air server

migration:
	go build -o tmp/migrate ./cmd/migrate/
	./tmp/migrate up