FROM golang:1.24.0-alpine3.19 AS build-env

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN  go build -o /server cmd/server 

FROM alpine:3.19.1

RUN apk update \
    && apk upgrade

WORKDIR /app
COPY --from=build-env /server .
COPY --from=build-env /app/cmd/server /app/

EXPOSE 80
CMD [ "./server" ]