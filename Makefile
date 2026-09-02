.PHONY: up down restart log migrate-up migrate-down test run-gateway run-vault

# Docker operations
up:
	docker-compose up -d

down:
	docker-compose down

restart:
	docker-compose down && docker-compose up -d

log:
	docker-compose logs -f

# Database Migrations (requires golang-migrate)
# Install: brew install golang-migrate
DB_URL="postgres://pp_user:pp_secret@localhost:5432/payment_processor?sslmode=disable"

migrate-up:
	migrate -path migrations -database $(DB_URL) -verbose up

migrate-down:
	migrate -path migrations -database $(DB_URL) -verbose down

# Run Services Locally
run-gateway:
	go run services/gateway/cmd/main.go

run-vault:
	go run services/vault/cmd/main.go

test:
	go test -v ./...

run-ledger:
	go run services/ledger/cmd/main.go

run-webhook:
	go run services/webhook/cmd/main.go
