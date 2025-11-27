.PHONY: build test lint clean install dev fmt coverage help

# Build settings
BINARY_NAME=panka
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DIR=bin
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Go settings
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/$(BUILD_DIR)
GOFILES=$(wildcard *.go)

# Tools
GOLANGCI_LINT_VERSION=v1.55.2

## help: Show this help message
help:
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/panka
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## install: Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) ./cmd/panka
	@echo "✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

## test: Run unit tests
test:
	@echo "Running unit tests..."
	@go test -v -race -timeout 30s ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

## test-integration: Run integration tests (requires LocalStack)
test-integration:
	@echo "Running integration tests..."
	@go test -v -race -tags=integration -timeout 5m ./test/integration/...

## test-all: Run all tests (unit + integration)
test-all: test test-integration

## lint: Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION))
	@golangci-lint run --timeout 5m

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w . 2>/dev/null || true
	@echo "✓ Code formatted"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.txt coverage.html
	@go clean
	@echo "✓ Clean complete"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies downloaded"

## dev: Set up development environment
dev: deps
	@echo "Setting up development environment..."
	@which golangci-lint > /dev/null || make install-tools
	@echo "✓ Development environment ready"

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)
	@echo "✓ Tools installed"

## run: Build and run the binary
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

## watch: Watch for changes and rebuild (requires entr)
watch:
	@which entr > /dev/null || (echo "entr not found. Install with: brew install entr" && exit 1)
	@echo "Watching for changes..."
	@find . -name "*.go" | entr -r make run

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t panka:$(VERSION) .
	@echo "✓ Docker image built: panka:$(VERSION)"

## localstack-start: Start LocalStack for testing
localstack-start:
	@echo "Starting LocalStack..."
	@docker-compose -f test/docker-compose.localstack.yml up -d
	@echo "✓ LocalStack started"

## localstack-stop: Stop LocalStack
localstack-stop:
	@echo "Stopping LocalStack..."
	@docker-compose -f test/docker-compose.localstack.yml down
	@echo "✓ LocalStack stopped"

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

## security: Run security scan
security:
	@echo "Running security scan..."
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	@gosec ./...

## pre-commit: Run all checks before commit
pre-commit: fmt vet lint test
	@echo "✓ All pre-commit checks passed!"

## version: Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

.DEFAULT_GOAL := help

