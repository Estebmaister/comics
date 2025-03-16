# Alpine-based Go image (Smallest)
FROM golang:1.24-alpine3.21 AS builder

# Installing git + SSL ca certificates
# Git is required for fetching the dependencies
# Ca-certificates is required to call HTTPS endpoints
RUN apk update && apk add --no-cache git make ca-certificates tzdata && update-ca-certificates

WORKDIR /app

# Copying go.mod & go.sum in previous layer for caching
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN echo $PWD && ls -la

# RUN  go build -o /server cmd/server 
# go build command with the -ldflags="-w -s" option to produce a 
# smaller binary file by stripping debug information and symbol tables 
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o ./server ./cmd/server

FROM alpine:3.21

RUN apk update && apk upgrade

# Import certs from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd

# Create a non-root user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory and copy the server binary and the static files
WORKDIR /app
COPY --from=builder app/server .
COPY --from=builder app/templates ./templates
COPY --from=builder app/static ./static
COPY --from=builder app/docs ./docs

# Adjust ownership and permissions
RUN chown -R appuser:appgroup /app

# Switch to the non-root user
USER appuser

HEALTHCHECK --interval=30s --timeout=10s --retries=3 CMD curl --fail http://localhost:8081/ || exit 1
EXPOSE 8081

CMD [ "./server" ]