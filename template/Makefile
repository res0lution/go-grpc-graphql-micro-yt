APP=identity-manager
MIGRATIONS_DIR=internal/db/migrations
DATABASE_URL?=

.PHONY: run build test test-unit test-integration lint lint-install migrate-up migrate-check

run:
	go run ./cmd/identity-manager

build:
	go build ./...

test:
	go test ./...

test-unit:
	go test ./cmd/... ./internal/...

test-integration:
	DATABASE_URL="$(DATABASE_URL)" go test ./internal/repository -run Integration

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./cmd/... ./internal/...

lint-install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

migrate-check:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required"; exit 1)
	@command -v psql >/dev/null 2>&1 || (echo "psql is required for external migrations"; exit 1)

migrate-up: migrate-check
	@for file in $(MIGRATIONS_DIR)/*.sql; do \
		echo "Applying $$file"; \
		psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f "$$file"; \
	done
