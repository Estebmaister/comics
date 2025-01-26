FROM golang:1.23-alpine

WORKDIR /server

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum .air.toml ./
RUN go mod download

CMD ["air", "-c", ".air.toml"]