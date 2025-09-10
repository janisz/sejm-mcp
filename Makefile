# Sejm MCP Server Makefile

.PHONY: build docker-build docker-test test test-unit test-integration test-smoke test-short test-coverage check-apis clean generate-types download-specs help

# Default target
all: build

# Build the server binary (static)
build:
	@echo "Building Sejm MCP Server (static binary)..."
	CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o sejm-mcp ./cmd/sejm-mcp

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t sejm-mcp:latest .

# Test Docker image
docker-test: docker-build
	@echo "Testing Docker image..."
	docker run --rm sejm-mcp:latest --help

# Run all tests
test: test-unit test-integration

# Run unit tests only (uses mocked data - no network required)
test-unit:
	@echo "Running unit tests..."
	go test -v ./internal/... -short

# Run integration tests (requires network access)
test-integration:
	@echo "Running integration tests..."
	@echo "Note: Integration tests require network access to Polish Parliament APIs"
	go test -v ./test/integration/... -timeout 60s

# Run smoke tests only (minimal connectivity check)
test-smoke:
	@echo "Running smoke tests (API connectivity check)..."
	@echo "Note: Smoke tests require network access to verify API connectivity"
	go test -v ./internal/server -run="TestAPIConnectivitySmoke" -timeout 30s

# Run tests in short mode (skip integration tests)
test-short:
	@echo "Running tests in short mode (unit tests only)..."
	go test -v -short ./...

# Run all tests with coverage
test-coverage:
	@echo "Running all tests with coverage..."
	go test -v -cover ./...

# Check API connectivity
check-apis:
	@echo "Checking Polish Parliament API connectivity..."
	@echo "Testing Sejm API..."
	@curl -s -o /dev/null -w "Sejm API: %{http_code}\n" https://api.sejm.gov.pl/sejm/term10/MP || echo "Sejm API: Failed"
	@echo "Testing ELI API..."
	@curl -s -o /dev/null -w "ELI API: %{http_code}\n" https://api.sejm.gov.pl/eli/acts || echo "ELI API: Failed"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f sejm-mcp
	rm -f *-openapi.json
	rm -f *-openapi-converted.json

# Download OpenAPI specifications
download-specs:
	@echo "Downloading OpenAPI specifications..."
	curl -s https://api.sejm.gov.pl/sejm/openapi/ -o sejm-openapi.json
	curl -s https://api.sejm.gov.pl/eli/openapi/ -o eli-openapi.json

# Convert YAML specs to JSON
convert-specs: download-specs
	@echo "Converting specifications to JSON..."
	yq eval -j sejm-openapi.json > sejm-openapi-converted.json
	yq eval -j eli-openapi.json > eli-openapi-converted.json

# Generate Go types from OpenAPI specifications
generate-types: convert-specs
	@echo "Generating Go types from OpenAPI specifications..."
	oapi-codegen -config sejm-codegen.yaml sejm-openapi-converted.json
	oapi-codegen -config eli-codegen.yaml eli-openapi-converted.json
	@echo "Type generation complete!"

# Install development dependencies
install-deps:
	@echo "Installing development dependencies..."
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "Dependencies installed!"

# Lint the code
lint:
	@echo "Running linter..."
	golangci-lint run

# Format the code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Tidy go modules
tidy:
	@echo "Tidying Go modules..."
	go mod tidy

# Run the server
run: build
	@echo "Starting Sejm MCP Server..."
	./sejm-mcp

# Development setup
dev-setup: install-deps generate-types
	@echo "Development environment setup complete!"

# Full rebuild (clean, generate, build)
rebuild: clean generate-types build

# Check for required tools
check-tools:
	@echo "Checking for required tools..."
	@which go > /dev/null || (echo "Go is not installed" && exit 1)
	@which yq > /dev/null || (echo "yq is not installed" && exit 1)
	@which oapi-codegen > /dev/null || (echo "oapi-codegen is not installed, run 'make install-deps'" && exit 1)
	@echo "All required tools are available!"

# Help target
help:
	@echo "Sejm MCP Server - Available targets:"
	@echo ""
	@echo "  build          - Build the server binary (static)"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-test    - Build and test Docker image"
	@echo "  test           - Run all tests (unit + integration)"
	@echo "  test-unit      - Run unit tests only (uses mocked data)"
	@echo "  test-integration - Run integration tests only (requires network)"
	@echo "  test-smoke     - Run API connectivity smoke tests (requires network)"
	@echo "  test-short     - Run tests in short mode (unit tests only)"
	@echo "  test-coverage  - Run all tests with coverage report"
	@echo "  check-apis     - Check Polish Parliament API connectivity"
	@echo "  clean          - Clean build artifacts and downloaded files"
	@echo "  download-specs - Download OpenAPI specifications"
	@echo "  convert-specs  - Convert YAML specs to JSON"
	@echo "  generate-types - Generate Go types from OpenAPI specs"
	@echo "  install-deps   - Install development dependencies"
	@echo "  lint           - Run code linter"
	@echo "  fmt            - Format Go code"
	@echo "  tidy           - Tidy Go modules"
	@echo "  run            - Build and run the server"
	@echo "  dev-setup      - Set up development environment"
	@echo "  rebuild        - Clean, generate types, and build"
	@echo "  check-tools    - Check for required development tools"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make dev-setup    # First-time setup"
	@echo "  make rebuild      # Full rebuild after API changes"
	@echo "  make test-unit    # Run fast unit tests (no network required)"
	@echo "  make test-smoke   # Quick API connectivity check"
	@echo "  make test         # Run all tests before committing"