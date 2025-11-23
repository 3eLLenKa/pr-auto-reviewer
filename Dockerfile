FROM golang:1.24.4 AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY config/local.yaml config/local.yaml
COPY .env ./
RUN go mod download
COPY . .
RUN go build -o /app/pr-reviewer ./cmd/app

FROM debian:bookworm-slim
WORKDIR /app

COPY --from=builder /app/pr-reviewer ./pr-reviewer
COPY --from=builder /app/config/local.yaml ./config/
COPY --from=builder /app/.env ./

EXPOSE 8080
CMD ["./pr-reviewer"]
