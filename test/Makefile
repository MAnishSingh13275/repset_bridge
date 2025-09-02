# Comprehensive Test Suite Makefile

.PHONY: all integration load security e2e comprehensive clean help

# Default target
all: comprehensive

# Individual test categories
integration:
	@echo "Running Integration Tests..."
	@go test ./integration/... -v -race -timeout=10m

load:
	@echo "Running Load Tests..."
	@go test ./load/... -v -race -timeout=15m

security:
	@echo "Running Security Tests..."
	@go test ./security/... -v -race -timeout=10m

e2e:
	@echo "Running End-to-End Tests..."
	@go test ./e2e/... -v -race -timeout=30m

# Run all comprehensive tests
comprehensive:
	@echo "Running Comprehensive Test Suite..."
	@go test . -v -race -timeout=60m -run=TestComprehensiveSuite

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@go test ./... -v -race -cover -coverprofile=coverage.out -timeout=60m
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run performance benchmarks
benchmark:
	@echo "Running performance benchmarks..."
	@go test . -bench=. -benchmem -timeout=30m

# Validate requirements coverage
requirements:
	@echo "Validating requirements coverage..."
	@go test . -v -run=TestRequirementValidation

# Run tests in parallel
parallel:
	@echo "Running tests in parallel..."
	@go test ./integration/... ./load/... ./security/... ./e2e/... -v -race -parallel=4 -timeout=45m

# Quick smoke test
smoke:
	@echo "Running smoke tests..."
	@go test ./integration/... -v -race -run=TestCompleteHardwareToCloudFlow -timeout=5m

# Clean up test artifacts
clean:
	@echo "Cleaning up test artifacts..."
	@rm -rf coverage.out coverage.html
	@find . -name "*.db" -type f -delete
	@find . -name "*_test_*" -type d -exec rm -rf {} + 2>/dev/null || true
	@find /tmp -name "bridge_*_test*" -type d -exec rm -rf {} + 2>/dev/null || true

# Setup test environment
setup:
	@echo "Setting up test environment..."
	@go mod tidy
	@go mod download

# Continuous integration target
ci: setup comprehensive requirements coverage
	@echo "CI pipeline completed successfully"

# Development target - fast feedback
dev: smoke integration
	@echo "Development tests completed"

# Help target
help:
	@echo "Available targets:"
	@echo "  all          - Run comprehensive test suite (default)"
	@echo "  integration  - Run integration tests only"
	@echo "  load         - Run load tests only"
	@echo "  security     - Run security tests only"
	@echo "  e2e          - Run end-to-end tests only"
	@echo "  comprehensive- Run all test categories with summary"
	@echo "  coverage     - Run tests with coverage report"
	@echo "  benchmark    - Run performance benchmarks"
	@echo "  requirements - Validate requirements coverage"
	@echo "  parallel     - Run tests in parallel"
	@echo "  smoke        - Run quick smoke tests"
	@echo "  clean        - Clean up test artifacts"
	@echo "  setup        - Setup test environment"
	@echo "  ci           - Full CI pipeline"
	@echo "  dev          - Fast development tests"
	@echo "  help         - Show this help message"

# Test environment variables
export CGO_ENABLED=1
export GO111MODULE=on