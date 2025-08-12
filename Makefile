# MySQL Schema Sync - Makefile

# Application information
APP_NAME := mysql-schema-sync
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -X main.GoVersion=$(GO_VERSION)"

# Cross-compilation targets
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
BUILD_DIR := build

.PHONY: help build build-all build-linux build-darwin build-windows test test-unit test-integration test-all clean docker-test docker-clean release

# Default target
help:
	@echo "Available targets:"
	@echo "  build            - Build the application for current platform"
	@echo "  build-all        - Build for all supported platforms"
	@echo "  build-linux      - Build for Linux platforms"
	@echo "  build-darwin     - Build for macOS platforms"
	@echo "  build-windows    - Build for Windows platforms"
	@echo "  test             - Run unit tests"
	@echo "  test-unit        - Run unit tests only"
	@echo "  test-integration - Run integration tests (requires MySQL)"
	@echo "  test-all         - Run all tests"
	@echo "  docker-test      - Run tests in Docker containers"
	@echo "  docker-clean     - Clean up Docker test containers"
	@echo "  clean            - Clean build artifacts"
	@echo "  coverage         - Generate test coverage report"
	@echo "  benchmark        - Run benchmark tests"
	@echo "  release          - Create release packages"
	@echo "  install          - Install the application locally"
	@echo "  version          - Show version information"

# Build the application for current platform
build:
	@echo "Building $(APP_NAME) v$(VERSION) for current platform..."
	go build $(LDFLAGS) -o $(APP_NAME) .

# Build for all supported platforms
build-all: clean-build
	@echo "Building $(APP_NAME) v$(VERSION) for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(APP_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then output_name=$$output_name.exe; fi; \
		echo "Building for $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $(BUILD_DIR)/$$output_name .; \
		if [ $$? -ne 0 ]; then \
			echo "Failed to build for $$os/$$arch"; \
			exit 1; \
		fi; \
	done
	@echo "Build complete. Binaries available in $(BUILD_DIR)/"

# Build for Linux platforms
build-linux: clean-build
	@echo "Building $(APP_NAME) v$(VERSION) for Linux platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 .

# Build for macOS platforms
build-darwin: clean-build
	@echo "Building $(APP_NAME) v$(VERSION) for macOS platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .

# Build for Windows platforms
build-windows: clean-build
	@echo "Building $(APP_NAME) v$(VERSION) for Windows platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-arm64.exe .

# Run unit tests only
test-unit:
	go test -v -short ./...

# Run integration tests (requires MySQL to be available)
test-integration:
	go test -v -tags=integration ./internal

# Run all tests
test-all: test-unit test-integration

# Default test target (unit tests)
test: test-unit

# Run tests with coverage
coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmark tests
benchmark:
	go test -v -bench=. -benchmem ./...

# Run integration benchmarks (requires MySQL)
benchmark-integration:
	go test -v -bench=. -benchmem -tags=integration ./internal

# Docker-based testing
docker-test:
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker-compose -f docker-compose.test.yml down

# Clean up Docker containers and images
docker-clean:
	docker-compose -f docker-compose.test.yml down -v --remove-orphans
	docker system prune -f

# Clean build artifacts
clean:
	rm -f mysql-schema-sync
	rm -f coverage.out coverage.html
	go clean -testcache

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Run tests with race detection
test-race:
	go test -v -race -short ./...

# Quick development test (unit tests only, no verbose output)
test-quick:
	go test -short ./...

# Test specific package
test-pkg:
	@if [ -z "$(PKG)" ]; then echo "Usage: make test-pkg PKG=./internal/database"; exit 1; fi
	go test -v $(PKG)

# Run tests and generate coverage for specific package
test-pkg-coverage:
	@if [ -z "$(PKG)" ]; then echo "Usage: make test-pkg-coverage PKG=./internal/database"; exit 1; fi
	go test -v -coverprofile=coverage.out $(PKG)
	go tool cover -html=coverage.out -o coverage.html

# Development workflow - format, test, build
dev: fmt test-quick build

# CI workflow - comprehensive testing
ci: fmt test-race test-all coverage

# Install development tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install the application locally
install: build
	@echo "Installing $(APP_NAME) to $(GOPATH)/bin or $(GOBIN)..."
	go install $(LDFLAGS) .

# Show version information
version:
	@echo "Application: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Go Version: $(GO_VERSION)"

# Create release packages
release: build-all
	@echo "Creating release packages..."
	@mkdir -p $(BUILD_DIR)/packages
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		binary_name=$(APP_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then binary_name=$$binary_name.exe; fi; \
		package_name=$(APP_NAME)-$(VERSION)-$$os-$$arch; \
		echo "Creating package for $$os/$$arch..."; \
		mkdir -p $(BUILD_DIR)/packages/$$package_name; \
		cp $(BUILD_DIR)/$$binary_name $(BUILD_DIR)/packages/$$package_name/; \
		cp README.md $(BUILD_DIR)/packages/$$package_name/; \
		cp LICENSE $(BUILD_DIR)/packages/$$package_name/; \
		cp CHANGELOG.md $(BUILD_DIR)/packages/$$package_name/; \
		cp -r examples $(BUILD_DIR)/packages/$$package_name/; \
		if [ $$os = "windows" ]; then \
			cd $(BUILD_DIR)/packages && zip -r $$package_name.zip $$package_name/; \
		else \
			cd $(BUILD_DIR)/packages && tar -czf $$package_name.tar.gz $$package_name/; \
		fi; \
		rm -rf $(BUILD_DIR)/packages/$$package_name; \
	done
	@echo "Release packages created in $(BUILD_DIR)/packages/"

# Clean build artifacts
clean-build:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

# Clean all artifacts
clean: clean-build
	@echo "Cleaning all artifacts..."
	rm -f $(APP_NAME)
	rm -f coverage.out coverage.html
	go clean -testcache

# Docker build for distribution
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

# Docker release build (multi-platform)
docker-build-release:
	@echo "Building multi-platform Docker images..."
	docker buildx build --platform linux/amd64,linux/arm64 -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest --push .

# Verify builds work
verify-builds: build-all
	@echo "Verifying builds..."
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		binary_name=$(APP_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then binary_name=$$binary_name.exe; fi; \
		echo "Checking $$binary_name..."; \
		if [ ! -f $(BUILD_DIR)/$$binary_name ]; then \
			echo "ERROR: $$binary_name not found"; \
			exit 1; \
		fi; \
		file $(BUILD_DIR)/$$binary_name; \
	done
	@echo "All builds verified successfully"

# Generate checksums for release
checksums: build-all
	@echo "Generating checksums..."
	@cd $(BUILD_DIR) && sha256sum * > checksums.txt
	@echo "Checksums generated in $(BUILD_DIR)/checksums.txt"