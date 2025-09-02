# Makefile for Gym Door Access Bridge
# Provides convenient commands for building, testing, and deploying

# Configuration
PROJECT_NAME := gym-door-bridge
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT_HASH := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# Directories
BUILD_DIR := build
DIST_DIR := dist
DOCS_DIR := docs

# Build flags
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commitHash=$(COMMIT_HASH)
BUILD_FLAGS := -trimpath -ldflags="$(LDFLAGS)"

# Go configuration
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Docker configuration
DOCKER_IMAGE := $(PROJECT_NAME)
DOCKER_TAG := $(VERSION)

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: help build build-all clean test test-coverage lint docker docker-build docker-run install uninstall release docs serve-docs

# Default target
all: build

# Help target
help: ## Show this help message
	@echo "$(GREEN)Gym Door Access Bridge - Build System$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build for current platform
	@echo "$(GREEN)[BUILD]$(NC) Building $(PROJECT_NAME) v$(VERSION) for current platform"
	$(GOBUILD) $(BUILD_FLAGS) -o $(PROJECT_NAME) ./cmd

build-all: ## Build for all platforms
	@echo "$(GREEN)[BUILD]$(NC) Building $(PROJECT_NAME) v$(VERSION) for all platforms"
	@chmod +x scripts/build.sh
	@./scripts/build.sh

build-windows: ## Build for Windows (amd64 and 386)
	@echo "$(GREEN)[BUILD]$(NC) Building for Windows"
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME)-windows-amd64.exe ./cmd
	GOOS=windows GOARCH=386 CGO_ENABLED=1 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME)-windows-386.exe ./cmd

build-darwin: ## Build for macOS (amd64 and arm64)
	@echo "$(GREEN)[BUILD]$(NC) Building for macOS"
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME)-darwin-amd64 ./cmd
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME)-darwin-arm64 ./cmd

build-linux: ## Build for Linux (amd64 and arm64)
	@echo "$(GREEN)[BUILD]$(NC) Building for Linux"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME)-linux-amd64 ./cmd
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME)-linux-arm64 ./cmd

# Clean targets
clean: ## Clean build artifacts
	@echo "$(GREEN)[CLEAN]$(NC) Removing build artifacts"
	$(GOCLEAN)
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	rm -f $(PROJECT_NAME)

clean-all: clean ## Clean all artifacts including Docker images
	@echo "$(GREEN)[CLEAN]$(NC) Removing Docker images"
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true
	docker rmi $(DOCKER_IMAGE):latest 2>/dev/null || true

# Test targets
test: ## Run tests
	@echo "$(GREEN)[TEST]$(NC) Running tests"
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(GREEN)[TEST]$(NC) Running tests with coverage"
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)[TEST]$(NC) Coverage report generated: coverage.html"

test-integration: ## Run integration tests
	@echo "$(GREEN)[TEST]$(NC) Running integration tests"
	$(GOTEST) -v -tags=integration ./test/integration/...

test-e2e: ## Run end-to-end tests
	@echo "$(GREEN)[TEST]$(NC) Running end-to-end tests"
	$(GOTEST) -v -tags=e2e ./test/e2e/...

test-all: test test-integration test-e2e ## Run all tests

# Lint targets
lint: ## Run linter
	@echo "$(GREEN)[LINT]$(NC) Running linter"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)[WARN]$(NC) golangci-lint not found, skipping"; \
	fi

fmt: ## Format code
	@echo "$(GREEN)[FMT]$(NC) Formatting code"
	$(GOCMD) fmt ./...

vet: ## Run go vet
	@echo "$(GREEN)[VET]$(NC) Running go vet"
	$(GOCMD) vet ./...

# Docker targets
docker: docker-build ## Build Docker image

docker-build: ## Build Docker image
	@echo "$(GREEN)[DOCKER]$(NC) Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)"
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		.

docker-run: ## Run Docker container
	@echo "$(GREEN)[DOCKER]$(NC) Running Docker container"
	docker run --rm -it \
		-p 8080:8080 \
		-v $(PWD)/config.yaml.example:/app/config/config.yaml:ro \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-push: ## Push Docker image to registry
	@echo "$(GREEN)[DOCKER]$(NC) Pushing Docker image"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest

# Installation targets
install: build ## Install binary to system
	@echo "$(GREEN)[INSTALL]$(NC) Installing $(PROJECT_NAME)"
	@if [ "$(shell uname)" = "Darwin" ]; then \
		sudo cp $(PROJECT_NAME) /usr/local/bin/; \
		sudo chmod +x /usr/local/bin/$(PROJECT_NAME); \
	elif [ "$(shell uname)" = "Linux" ]; then \
		sudo cp $(PROJECT_NAME) /usr/local/bin/; \
		sudo chmod +x /usr/local/bin/$(PROJECT_NAME); \
	else \
		echo "$(YELLOW)[WARN]$(NC) Manual installation required on this platform"; \
	fi

uninstall: ## Uninstall binary from system
	@echo "$(GREEN)[UNINSTALL]$(NC) Uninstalling $(PROJECT_NAME)"
	@if [ "$(shell uname)" = "Darwin" ] || [ "$(shell uname)" = "Linux" ]; then \
		sudo rm -f /usr/local/bin/$(PROJECT_NAME); \
	else \
		echo "$(YELLOW)[WARN]$(NC) Manual uninstallation required on this platform"; \
	fi

# Release targets
release: clean build-all ## Build release artifacts
	@echo "$(GREEN)[RELEASE]$(NC) Creating release artifacts"
	@chmod +x scripts/generate-manifest.sh
	@./scripts/generate-manifest.sh
	@echo "$(GREEN)[RELEASE]$(NC) Release $(VERSION) ready in $(DIST_DIR)/"

release-docker: docker-build docker-push ## Build and push Docker release

# Documentation targets
docs: ## Generate documentation
	@echo "$(GREEN)[DOCS]$(NC) Documentation available in $(DOCS_DIR)/"
	@ls -la $(DOCS_DIR)/

serve-docs: ## Serve documentation locally (requires Python)
	@echo "$(GREEN)[DOCS]$(NC) Serving documentation at http://localhost:8000"
	@if command -v python3 >/dev/null 2>&1; then \
		cd $(DOCS_DIR) && python3 -m http.server 8000; \
	elif command -v python >/dev/null 2>&1; then \
		cd $(DOCS_DIR) && python -m SimpleHTTPServer 8000; \
	else \
		echo "$(RED)[ERROR]$(NC) Python not found"; \
	fi

# Development targets
dev: ## Run in development mode
	@echo "$(GREEN)[DEV]$(NC) Running in development mode"
	$(GOCMD) run ./cmd --config config.yaml.example --log-level debug

dev-watch: ## Run with file watching (requires entr)
	@echo "$(GREEN)[DEV]$(NC) Running with file watching"
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" | entr -r $(GOCMD) run ./cmd --config config.yaml.example --log-level debug; \
	else \
		echo "$(RED)[ERROR]$(NC) entr not found. Install with: brew install entr (macOS) or apt-get install entr (Linux)"; \
	fi

# Dependency targets
deps: ## Download dependencies
	@echo "$(GREEN)[DEPS]$(NC) Downloading dependencies"
	$(GOMOD) download

deps-update: ## Update dependencies
	@echo "$(GREEN)[DEPS]$(NC) Updating dependencies"
	$(GOMOD) tidy
	$(GOGET) -u ./...

deps-vendor: ## Vendor dependencies
	@echo "$(GREEN)[DEPS]$(NC) Vendoring dependencies"
	$(GOMOD) vendor

# Utility targets
version: ## Show version information
	@echo "$(GREEN)[VERSION]$(NC) $(PROJECT_NAME) v$(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Commit Hash: $(COMMIT_HASH)"

info: ## Show build information
	@echo "$(GREEN)[INFO]$(NC) Build Information"
	@echo "  Project: $(PROJECT_NAME)"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Commit Hash: $(COMMIT_HASH)"
	@echo "  Go Version: $(shell $(GOCMD) version)"
	@echo "  Platform: $(shell uname -s)/$(shell uname -m)"

check-tools: ## Check required tools
	@echo "$(GREEN)[CHECK]$(NC) Checking required tools"
	@echo -n "  Go: "; command -v go >/dev/null 2>&1 && echo "✓" || echo "✗"
	@echo -n "  Git: "; command -v git >/dev/null 2>&1 && echo "✓" || echo "✗"
	@echo -n "  Docker: "; command -v docker >/dev/null 2>&1 && echo "✓" || echo "✗"
	@echo -n "  Make: "; command -v make >/dev/null 2>&1 && echo "✓" || echo "✗"
	@echo -n "  SQLite3: "; command -v sqlite3 >/dev/null 2>&1 && echo "✓" || echo "✗"

# CI/CD targets
ci: lint test ## Run CI pipeline
	@echo "$(GREEN)[CI]$(NC) CI pipeline completed"

cd: build-all release ## Run CD pipeline
	@echo "$(GREEN)[CD]$(NC) CD pipeline completed"

# Security targets
security-scan: ## Run security scan (requires gosec)
	@echo "$(GREEN)[SECURITY]$(NC) Running security scan"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "$(YELLOW)[WARN]$(NC) gosec not found, install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Benchmark targets
bench: ## Run benchmarks
	@echo "$(GREEN)[BENCH]$(NC) Running benchmarks"
	$(GOTEST) -bench=. -benchmem ./...

# Profile targets
profile-cpu: ## Generate CPU profile
	@echo "$(GREEN)[PROFILE]$(NC) Generating CPU profile"
	$(GOCMD) run ./cmd --profile-cpu cpu.prof --config config.yaml.example &
	@sleep 30
	@pkill $(PROJECT_NAME) || true
	@echo "$(GREEN)[PROFILE]$(NC) CPU profile saved to cpu.prof"

profile-memory: ## Generate memory profile
	@echo "$(GREEN)[PROFILE]$(NC) Generating memory profile"
	$(GOCMD) run ./cmd --profile-memory memory.prof --config config.yaml.example &
	@sleep 30
	@pkill $(PROJECT_NAME) || true
	@echo "$(GREEN)[PROFILE]$(NC) Memory profile saved to memory.prof"