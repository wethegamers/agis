.PHONY: build run test lint docker-build docker-run clean help

# Binary name
BINARY_NAME=agis-bot
DOCKER_IMAGE=agis-bot
DOCKER_TAG=dev

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) .
	@echo "✅ Build complete: bin/$(BINARY_NAME)"

# Run the application locally
run:
	@echo "Running $(BINARY_NAME) locally..."
	API_PORT=8080 go run .

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "✅ Tests complete"

# Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# Run linter
lint:
	@echo "Running golangci-lint..."
	golangci-lint run ./...
	@echo "✅ Lint complete"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .
	@echo "✅ Format complete"

# Tidy dependencies
tidy:
	@echo "Tidying Go modules..."
	go mod tidy
	go mod verify
	@echo "✅ Dependencies tidied"

# Build Docker image
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "✅ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Run in Docker
docker-run:
	@echo "Running $(DOCKER_IMAGE):$(DOCKER_TAG) in Docker..."
	docker run --rm --env-file .env -p 8080:8080 -p 8081:8081 $(DOCKER_IMAGE):$(DOCKER_TAG)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

# Run database migration (requires port-forward or cluster access)
migrate:
	@echo "Running database migration..."
	kubectl exec -i -n postgres-dev postgres-dev-0 -- psql -U agis_dev_user -d agis_dev < deployments/migrations/v1.7.0-rest-api-scheduling.sql
	@echo "✅ Migration complete"

# Port-forward Postgres for local development
db-forward:
	@echo "Port-forwarding postgres-dev..."
	kubectl port-forward -n postgres-dev postgres-dev-0 5432:5432

# Port-forward Grafana for dashboard access
grafana-forward:
	@echo "Port-forwarding Grafana..."
	kubectl port-forward -n monitoring svc/grafana 3000:80

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application binary"
	@echo "  run             - Run the application locally"
	@echo "  test            - Run unit tests with race detector"
	@echo "  test-coverage   - Run tests and generate HTML coverage report"
	@echo "  lint            - Run golangci-lint"
	@echo "  fmt             - Format Go code"
	@echo "  tidy            - Tidy and verify Go modules"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Run application in Docker"
	@echo "  clean           - Remove build artifacts"
	@echo "  migrate         - Run database migration in cluster"
	@echo "  db-forward      - Port-forward postgres-dev pod"
	@echo "  grafana-forward - Port-forward Grafana service"
	@echo "  help            - Show this help message"
