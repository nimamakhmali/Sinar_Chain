# Sinar Chain Makefile
# ===================

# Variables
BINARY_NAME=sinar_chain
BUILD_DIR=build
SRC_DIR=src
MAIN_FILE=main.go
VERSION=1.0.0

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Default target
all: clean build

# Build the application
build:
	@echo "🔨 Building Sinar Chain..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "✅ Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application
run:
	@echo "🚀 Running Sinar Chain..."
	$(GOCMD) run $(MAIN_FILE)

# Run with race detection
run-race:
	@echo "🏃 Running Sinar Chain with race detection..."
	$(GOCMD) run -race $(MAIN_FILE)

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "✅ Clean completed"

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	$(GOMOD) tidy
	$(GOMOD) download
	@echo "✅ Dependencies installed"

# Run tests
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...
	@echo "✅ Tests completed"

# Run tests with coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "🎨 Formatting code..."
	$(GOCMD) fmt ./...
	@echo "✅ Code formatted"

# Lint code
lint:
	@echo "🔍 Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Install development tools
install-tools:
	@echo "🛠️  Installing development tools..."
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOCMD) install golang.org/x/tools/cmd/goimports@latest
	@echo "✅ Development tools installed"

# Build for different platforms
build-linux:
	@echo "🐧 Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_FILE)

build-windows:
	@echo "🪟 Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_FILE)

build-darwin:
	@echo "🍎 Building for macOS..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_FILE)

build-all: build-linux build-windows build-darwin
	@echo "✅ Multi-platform build completed"

# Docker commands
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t sinar-chain:$(VERSION) .
	@echo "✅ Docker image built: sinar-chain:$(VERSION)"

docker-run:
	@echo "🐳 Running Sinar Chain in Docker..."
	docker run -p 8080:8080 sinar-chain:$(VERSION)

# Development commands
dev: deps fmt lint test run

# Production build
prod: clean deps fmt lint test build
	@echo "🚀 Production build completed"

# Show help
help:
	@echo "Sinar Chain - Makefile Commands"
	@echo "================================"
	@echo "build          - Build the application"
	@echo "run            - Run the application"
	@echo "run-race       - Run with race detection"
	@echo "clean          - Clean build artifacts"
	@echo "deps           - Install dependencies"
	@echo "test           - Run tests"
	@echo "test-coverage  - Run tests with coverage"
	@echo "fmt            - Format code"
	@echo "lint           - Lint code"
	@echo "install-tools  - Install development tools"
	@echo "build-linux    - Build for Linux"
	@echo "build-windows  - Build for Windows"
	@echo "build-darwin   - Build for macOS"
	@echo "build-all      - Build for all platforms"
	@echo "docker-build   - Build Docker image"
	@echo "docker-run     - Run in Docker"
	@echo "dev            - Development workflow"
	@echo "prod           - Production build"
	@echo "help           - Show this help"

# Default target
.DEFAULT_GOAL := help

# Phony targets
.PHONY: all build run run-race clean deps test test-coverage fmt lint install-tools build-linux build-windows build-darwin build-all docker-build docker-run dev prod help 