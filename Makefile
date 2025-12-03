.PHONY: build run test lint docker-build docker-run clean help verify-all

# Binary name
BINARY_NAME=agis-bot
DOCKER_IMAGE=ghcr.io/wethegamers/agis-bot
DOCKER_TAG=dev
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-X github.com/wethegamers/agis-core/version.Version=$(VERSION) \
	-X github.com/wethegamers/agis-core/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/wethegamers/agis-core/version.BuildDate=$(BUILD_DATE)"

# Ensure GOPRIVATE is set for private modules
export GOPRIVATE=github.com/wethegamers/*

# Build the application
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) .
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
	@export PATH="$$HOME/go/bin:$$PATH" && golangci-lint run ./...
	@echo "✅ Lint complete"

# Install golangci-lint
lint-install:
	@echo "Installing golangci-lint v1.64.8..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
	@echo "✅ golangci-lint installed"

# Setup git hooks for pre-push lint checks
hooks:
	@echo "Setting up git hooks..."
	@mkdir -p .git/hooks
	@cp scripts/pre-push .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "✅ Git hooks installed"

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

# Build Docker image locally (requires GitHub token for private modules)
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "❌ GITHUB_TOKEN required. Set it with: export GITHUB_TOKEN=ghp_xxx"; \
		exit 1; \
	fi
	DOCKER_BUILDKIT=1 docker build \
		--secret id=github_token,env=GITHUB_TOKEN \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--platform linux/arm64 \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "✅ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Build Docker image for local arch (faster for testing)
docker-build-local:
	@echo "Building Docker image for local architecture..."
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "❌ GITHUB_TOKEN required. Set it with: export GITHUB_TOKEN=ghp_xxx"; \
		exit 1; \
	fi
	DOCKER_BUILDKIT=1 docker build \
		--secret id=github_token,env=GITHUB_TOKEN \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "✅ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Run in Docker
docker-run:
	@echo "Running $(DOCKER_IMAGE):$(DOCKER_TAG) in Docker..."
	docker run --rm --env-file .env -p 9090:9090 $(DOCKER_IMAGE):$(DOCKER_TAG)

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
	@echo "AGIS Bot Development Commands"
	@echo "=============================="
	@echo ""
	@echo "Local Development:"
	@echo "  build           - Build the application binary"
	@echo "  run             - Run the application locally"
	@echo "  test            - Run unit tests with race detector"
	@echo "  test-coverage   - Run tests and generate HTML coverage report"
	@echo "  lint            - Run golangci-lint"
	@echo "  lint-install    - Install golangci-lint"
	@echo "  fmt             - Format Go code"
	@echo "  tidy            - Tidy and verify Go modules"
	@echo "  hooks           - Install git pre-push hooks"
	@echo ""
	@echo "Docker (requires GITHUB_TOKEN for private modules):"
	@echo "  docker-build       - Build Docker image (arm64 for prod)"
	@echo "  docker-build-local - Build Docker image for local arch (faster)"
	@echo "  docker-run         - Run application in Docker"
	@echo ""
	@echo "Pre-CI Verification (run before pushing):"
	@echo "  verify-all      - Run all checks locally (lint, test, build)"
	@echo ""
	@echo "Cluster Operations:"
	@echo "  migrate         - Run database migration in cluster"
	@echo "  db-forward      - Port-forward postgres-dev pod"
	@echo "  grafana-forward - Port-forward Grafana service"
	@echo ""
	@echo "Utilities:"
	@echo "  clean           - Remove build artifacts"
	@echo "  help            - Show this help message"
	@echo ""
	@echo "Environment Variables:"
	@echo "  GITHUB_TOKEN    - Required for docker-build (access to agis-core)"
	@echo "  DOCKER_TAG      - Docker image tag (default: dev)"
	@echo "  VERSION         - Version string (default: git describe)"

# Run all verification steps locally (saves CI minutes)
verify-all: tidy lint test build
	@echo ""
	@echo "============================================"
	@echo "✅ All local verification passed!"
	@echo "   Safe to push to trigger CI pipeline."
	@echo "============================================"
