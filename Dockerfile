# Production image: Go comics API (replaces legacy Python Flask on :5001).
# Build context: repository root (see docker-compose.yaml).

FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
ENV GOTOOLCHAIN=auto
COPY go_server/go.mod go_server/go.sum ./
RUN go mod download
COPY go_server/ .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache curl ca-certificates tzdata \
	&& addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /server .
COPY go_server/templates ./templates
COPY go_server/static ./static
COPY go_server/docs ./docs

RUN chown -R appuser:appgroup /app

USER appuser

ENV HTTP_ADDRESS=0.0.0.0
ENV HTTP_PORT=8081
ENV COMICS_DB_DRIVER=sqlite
ENV COMICS_SQLITE_PATH=/data/comics.db
ENV HEALTHCHECK_PATH=/health

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
	CMD curl --fail "http://127.0.0.1:${HTTP_PORT}${HEALTHCHECK_PATH}" || exit 1

CMD ["./server"]
