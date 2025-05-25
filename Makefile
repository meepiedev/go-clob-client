# Makefile for Go CLOB Client
# Based on patterns from go-order-utils-main/Makefile and py-clob-client-main/Makefile

.PHONY: all build test clean lint fmt vet examples

# Variables
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)
GONAME=go-clob-client

# Build the project
all: test build

build:
	@echo "Building..."
	@go build -o $(GOBIN)/$(GONAME) ./cmd/...

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./pkg/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./pkg/...
	@go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@go clean
	@rm -rf $(GOBIN)
	@rm -f coverage.out coverage.html

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run --timeout=5m

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Build examples
examples:
	@echo "Building examples..."
	@for example in $(shell ls examples/*.go); do \
		go build -o $(GOBIN)/$$(basename $$example .go) $$example; \
	done

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run a specific example
run-example:
	@if [ -z "$(EXAMPLE)" ]; then \
		echo "Usage: make run-example EXAMPLE=get_ok"; \
		exit 1; \
	fi
	@echo "Running example: $(EXAMPLE)"
	@go run examples/$(EXAMPLE).go

# Generate mocks (if needed)
mocks:
	@echo "Generating mocks..."
	@mockery --all --dir pkg --output pkg/mocks

# Check for security vulnerabilities
security:
	@echo "Checking for vulnerabilities..."
	@govulncheck ./...

# Run all checks before commit
pre-commit: fmt vet lint test
	@echo "All checks passed!"

# Show help
help:
	@echo "Available targets:"
	@echo "  make build          - Build the project"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make lint           - Run linter"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make examples       - Build all examples"
	@echo "  make deps           - Install dependencies"
	@echo "  make run-example    - Run a specific example (EXAMPLE=name)"
	@echo "  make mocks          - Generate mocks"
	@echo "  make security       - Check for vulnerabilities"
	@echo "  make pre-commit     - Run all checks before commit"