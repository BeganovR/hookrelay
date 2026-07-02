BIN      := hookrelay
CMD      := ./cmd/hookrelay
BUILD    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -ldflags "-s -w -X main.version=$(BUILD)"

.PHONY: build test test-integration lint vet docker-up docker-down migrate clean

build:
	go build $(LDFLAGS) -o bin/$(BIN) $(CMD)

test:
	go test ./... -count=1 -race -timeout 60s

test-integration:
	go test ./... -tags integration -count=1 -race -timeout 300s

lint:
	golangci-lint run ./...

vet:
	go vet ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

migrate:
	@echo "Migrations are applied automatically on startup."

clean:
	rm -rf bin/
