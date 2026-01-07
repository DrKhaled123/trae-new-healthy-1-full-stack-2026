.PHONY: help build run test clean migrate setup dev watch

# Default target
help:
	@echo "Available commands:"
	@echo "  make setup     - Initial setup (deps, migrations, build)"
	@echo "  make build     - Build the application"
	@echo "  make run       - Run the application"
	@echo "  make dev       - Run in development mode with auto-reload"
	@echo "  make test      - Run all tests"
	@echo "  make migrate   - Run database migrations"
	@echo "  make clean     - Clean build artifacts"
	@echo "  make lint      - Run linter"
	@echo "  make fmt       - Format code"
	@echo "  make docker    - Build Docker image"
	@echo "  make test-api  - Test API endpoints"

# Setup everything
setup:
	@echo "🚀 Setting up..."
	go mod download
	go mod tidy
	chmod +x run_migrations.sh
	./run_migrations.sh
	go build -o bin/nutrition-platform ./main.go
	@echo "✅ Setup complete!"

# Build
build:
	@echo "🔨 Building..."
	go build -o bin/nutrition-platform ./main.go
	@echo "✅ Build complete!"

# Run
run:
	@echo "▶️  Starting server..."
	go run main.go

# Development mode with auto-reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@if command -v air > /dev/null; then \
		echo "🔄 Starting in dev mode with auto-reload..."; \
		air; \
	else \
		echo "⚠️  Air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "▶️  Running normally..."; \
		go run main.go; \
	fi

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# Migrate database
migrate:
	@echo "🗄️  Running migrations..."
	chmod +x run_migrations.sh
	./run_migrations.sh

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean
	@echo "✅ Clean complete!"

# Format code
fmt:
	@echo "📝 Formatting code..."
	go fmt ./...
	@echo "✅ Formatting complete!"

# Lint code
lint:
	@echo "🔍 Linting..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		go vet ./...; \
	fi

# Build Docker image
docker:
	@echo "🐳 Building Docker image..."
	docker build -t nutrition-platform:latest .
	@echo "✅ Docker image built!"

# Kill processes on port 8080
kill-port:
	@echo "🛑 Killing processes on port 8080..."
	@chmod +x scripts/kill-port.sh
	@./scripts/kill-port.sh

# Start server with port cleanup
start-clean: kill-port
	@echo "🚀 Starting server..."
	@go run main.go

# Build and start with port cleanup
run-clean: kill-port build
	@echo "🚀 Running server..."
	@./bin/nutrition-platform

# Test API endpoints (requires server to be running)
test-api:
	@echo "🧪 Testing API endpoints..."
	@chmod +x scripts/test-api.sh
	@./scripts/test-api.sh

# Test public routes (requires server to be running)
test-public-routes:
	@echo "🧪 Testing public routes..."
	@chmod +x scripts/test-public-routes.sh
	@./scripts/test-public-routes.sh

# Generate API documentation
docs:
	@echo "📚 Generating API docs..."
	@if command -v swag > /dev/null; then \
		swag init; \
	else \
		echo "⚠️  swag not installed. Install with: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi