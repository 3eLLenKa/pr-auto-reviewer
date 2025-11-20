FROM golang:1.24.4-alpine

RUN apk add --no-cache git

WORKDIR /app

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

ENTRYPOINT [ "goose" ]