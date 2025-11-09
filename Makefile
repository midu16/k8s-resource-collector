.PHONY: build test clean install plugin help

# Variables
BINARY_NAME=k8s-resource-collector
PLUGIN_NAME=oc-collect-resources
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"
CMD_DIR=./cmd

# Default target
all: test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) $(CMD_DIR)

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	@echo "Build complete! Binaries in bin/"

# Build as oc plugin
plugin:
	@echo "Building as oc plugin..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(PLUGIN_NAME) $(CMD_DIR)

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run enhanced functional tests
test: build
	@echo "Running enhanced functional tests..."
	cd tests && ./simple_test_runner.sh

# Run comprehensive functional tests
test-comprehensive: build
	@echo "Running comprehensive functional tests..."
	cd tests && ./test_runner.sh

# Run Go-based functional tests
test-go:
	@echo "Running Go-based functional tests..."
	cd tests && go test -v -timeout 30s

# Run all tests (unit + functional)
test-all: test-unit test-go
	@echo "All tests completed!"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Check formatting
fmt-check:
	@echo "Checking code formatting..."
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "The following files are not formatted:"; \
		gofmt -s -l .; \
		exit 1; \
	fi

# Run vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run linters
lint: fmt-check vet
	@echo "Linting complete!"

# Tidy go modules
tidy:
	@echo "Tidying go modules..."
	go mod tidy
	go mod verify

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f bin/$(BINARY_NAME) bin/$(PLUGIN_NAME)
	rm -f bin/$(BINARY_NAME)-*
	rm -f coverage.out

# Clean test artifacts
clean-tests:
	@echo "Cleaning test artifacts..."
	rm -rf tests/test-output tests/test-reports
	cd tests && ./test_runner.sh cleanup 2>/dev/null || true

# Clean everything
clean-all: clean clean-tests
	@echo "Cleaning all artifacts..."
	go clean -cache -modcache -testcache

# Install binary to /usr/local/bin
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp bin/$(BINARY_NAME) /usr/local/bin/

# Install as oc plugin
install-plugin: plugin
	@echo "Installing $(PLUGIN_NAME) as oc plugin..."
	sudo cp bin/$(PLUGIN_NAME) /usr/local/bin/

# Run the binary with default settings
run: build
	@echo "Running $(BINARY_NAME)..."
	./bin/$(BINARY_NAME)

# Run with verbose output
run-verbose: build
	@echo "Running $(BINARY_NAME) with verbose output..."
	./bin/$(BINARY_NAME) --verbose

# Run in single file mode
run-single: build
	@echo "Running $(BINARY_NAME) in single file mode..."
	./bin/$(BINARY_NAME) --single-file --file ./output/all-resources.yaml

# Test the binary
test-binary: build
	@echo "Testing $(BINARY_NAME) binary..."
	./bin/$(BINARY_NAME) --help
	@echo "Binary test completed successfully!"

# Show help
help:
	@echo "Available targets:"
	@echo "  build              - Build the binary"
	@echo "  build-all          - Build for multiple platforms"
	@echo "  plugin             - Build as oc plugin"
	@echo "  test               - Run enhanced functional tests (auto-builds)"
	@echo "  test-unit          - Run unit tests"
	@echo "  test-go            - Run Go-based functional tests"
	@echo "  test-comprehensive - Run comprehensive functional tests (auto-builds)"
	@echo "  test-all           - Run all tests (unit + functional)"
	@echo "  test-binary        - Test the binary (requires build)"
	@echo "  fmt                - Format code"
	@echo "  fmt-check          - Check code formatting"
	@echo "  vet                - Run go vet"
	@echo "  lint               - Run all linters (fmt-check + vet)"
	@echo "  tidy               - Tidy and verify go modules"
	@echo "  clean              - Clean build artifacts"
	@echo "  clean-tests        - Clean test artifacts"
	@echo "  clean-all          - Clean everything (build + test + cache)"
	@echo "  install            - Install binary to /usr/local/bin"
	@echo "  install-plugin     - Install as oc plugin"
	@echo "  run                - Run the binary"
	@echo "  run-verbose        - Run with verbose output"
	@echo "  run-single         - Run in single file mode"
	@echo "  help               - Show this help"

# Create directories
dirs:
	@echo "Creating directories..."
	mkdir -p bin output output-custom tests/test-output tests/test-reports

# Setup development environment
setup: dirs tidy
	@echo "Setting up development environment..."
	@echo "Development environment ready!"
