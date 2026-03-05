.PHONY: dev build test lint migrate seed

# Start everything with Docker
dev:
	docker compose up --build

# Start only dependencies (DB + Redis)
dev-deps:
	docker compose up postgres redis adminer

# Run backend locally
run-backend:
	cd backend && go run ./cmd/api

# Run frontend locally
run-frontend:
	cd frontend && npm run dev

# Build backend binary
build-backend:
	cd backend && CGO_ENABLED=0 go build -o bin/api ./cmd/api

# Run Go tests
test-backend:
	cd backend && go test ./... -v

# Run frontend type check
test-frontend:
	cd frontend && npm run type-check

# Lint everything
lint:
	cd backend && golangci-lint run
	cd frontend && npm run lint

# Format Go code
fmt:
	cd backend && gofmt -w .

# Tidy Go modules
tidy:
	cd backend && go mod tidy

# Install frontend deps
install:
	cd frontend && npm install

# Generate Swagger docs
swagger:
	cd backend && swag init -g cmd/api/main.go

help:
	@echo "Available commands:"
	@echo "  make dev            - Start full stack with Docker"
	@echo "  make dev-deps       - Start only DB + Redis"
	@echo "  make run-backend    - Run Go backend locally"
	@echo "  make run-frontend   - Run Next.js frontend locally"
	@echo "  make test-backend   - Run Go tests"
	@echo "  make build-backend  - Build Go binary"
	@echo "  make tidy           - Tidy Go modules"
