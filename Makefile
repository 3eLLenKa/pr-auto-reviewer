run:
	go run ./cmd/main.go

test:
	$(GO) test ./... -cover

lint:
	golangci-lint run ./...

openapi-gen:
	oapi-codegen -config oapi-codegen.yml openapi.yml

up:
	docker compose up -d --build

down:
	docker compose down

docker-logs:
	docker compose logs -f

env:
	cp .env.example .env