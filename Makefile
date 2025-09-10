.PHONY: build test clean install plugin help

# Variables
BINARY_NAME=k8s-resource-collector
PLUGIN_NAME=oc-collect-resources
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Default target
all: test build

# Build the binary (standalone version)
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/main_standalone.go

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/main_standalone.go
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/main_standalone.go
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/main_standalone.go
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/main_standalone.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/main_standalone.go

# Build as oc plugin
plugin:
	@echo "Building as oc plugin..."
	go build $(LDFLAGS) -o bin/$(PLUGIN_NAME) ./cmd/main_standalone.go

# Run enhanced functional tests
test:
	@echo "Running enhanced functional tests..."
	cd tests && ./simple_test_runner.sh

# Run comprehensive functional tests
test-comprehensive:
	@echo "Running comprehensive functional tests..."
	cd tests && ./test_runner.sh

# Run Go-based functional tests
test-go:
	@echo "Running Go-based functional tests..."
	cd tests && go test -v -timeout 30s

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

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f bin/$(BINARY_NAME) bin/$(PLUGIN_NAME)
	rm -f bin/$(BINARY_NAME)-*

# Clean test artifacts
clean-tests:
	@echo "Cleaning test artifacts..."
	rm -rf tests/test-output tests/test-reports
	cd tests && ./test_runner.sh cleanup 2>/dev/null || true

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
	./bin/$(BINARY_NAME) --single-file --output-file ./output/all-resources.yaml

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
	@echo "  test               - Run enhanced functional tests"
	@echo "  test-comprehensive - Run comprehensive functional tests"
	@echo "  test-go            - Run Go-based functional tests"
	@echo "  test-binary        - Test the binary"
	@echo "  fmt                - Format code"
	@echo "  fmt-check          - Check code formatting"
	@echo "  vet                - Run go vet"
	@echo "  clean              - Clean build artifacts"
	@echo "  clean-tests        - Clean test artifacts"
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
setup: dirs
	@echo "Setting up development environment..."
	@echo "Development environment ready!"
