#!/bin/bash

# Enhanced Functional Testing Runner
# This script orchestrates all functional tests for k8s-resource-collector

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BINARY_PATH="$PROJECT_ROOT/bin/k8s-resource-collector"
TEST_OUTPUT_DIR="$SCRIPT_DIR/test-output"
TEST_REPORTS_DIR="$SCRIPT_DIR/test-reports"

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to print header
print_header() {
    echo -e "${YELLOW}"
    echo "=================================================="
    echo "Enhanced Functional Testing for k8s-resource-collector"
    echo "=================================================="
    echo -e "${NC}"
}

# Function to print section header
print_section() {
    echo -e "${BLUE}"
    echo "--- $1 ---"
    echo -e "${NC}"
}

# Function to print result
print_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name - $message"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Function to check prerequisites
check_prerequisites() {
    print_section "Checking Prerequisites"
    
    # Check if binary exists
    if [ ! -f "$BINARY_PATH" ]; then
        echo -e "${RED}Error: Binary not found at $BINARY_PATH${NC}"
        echo "Please run 'make build' first"
        exit 1
    fi
    
    # Check binary size
    local binary_size=$(stat -c%s "$BINARY_PATH" 2>/dev/null || stat -f%z "$BINARY_PATH" 2>/dev/null)
    if [ "$binary_size" -lt 1000000 ]; then
        print_result "Binary Size" "FAIL" "Binary too small ($binary_size bytes)"
    else
        print_result "Binary Size" "PASS" "Binary size OK ($binary_size bytes)"
    fi
    
    # Check if binary is executable
    if [ -x "$BINARY_PATH" ]; then
        print_result "Binary Executable" "PASS" "Binary is executable"
    else
        print_result "Binary Executable" "FAIL" "Binary is not executable"
    fi
    
    # Check Go version
    if command -v go >/dev/null 2>&1; then
        local go_version=$(go version | cut -d' ' -f3)
        print_result "Go Version" "PASS" "Go $go_version available"
    else
        print_result "Go Version" "FAIL" "Go not available"
    fi
}

# Function to run shell-based tests
run_shell_tests() {
    print_section "Running Shell-based Tests"
    
    if [ -f "$SCRIPT_DIR/test_runner.sh" ]; then
        chmod +x "$SCRIPT_DIR/test_runner.sh"
        cd "$SCRIPT_DIR"
        
        if ./test_runner.sh; then
            print_result "Shell Tests" "PASS" "All shell tests passed"
        else
            print_result "Shell Tests" "FAIL" "Some shell tests failed"
        fi
    else
        print_result "Shell Tests" "FAIL" "test_runner.sh not found"
    fi
}

# Function to run Go-based tests
run_go_tests() {
    print_section "Running Go-based Tests"
    
    if [ -f "$SCRIPT_DIR/functional_test.go" ]; then
        cd "$SCRIPT_DIR"
        
        # Create test reports directory
        mkdir -p "$TEST_REPORTS_DIR"
        
        # Run Go tests with various output formats
        if go test -v -timeout 30s -json > "$TEST_REPORTS_DIR/go_tests.json" 2>&1; then
            print_result "Go Tests" "PASS" "All Go tests passed"
        else
            print_result "Go Tests" "FAIL" "Some Go tests failed"
        fi
    else
        print_result "Go Tests" "FAIL" "functional_test.go not found"
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_section "Running Integration Tests"
    
    # Test complete workflows
    local test_cases=(
        "Directory Mode:--output:$TEST_OUTPUT_DIR/integration-dir:--verbose"
        "Single File Mode:--single-file:--output-file:$TEST_OUTPUT_DIR/integration-single.yaml:--verbose"
        "Clean Mode:--output:$TEST_OUTPUT_DIR/integration-clean:--clean:--verbose"
    )
    
    for test_case in "${test_cases[@]}"; do
        IFS=':' read -r name args <<< "$test_case"
        
        # Create test directory
        mkdir -p "$TEST_OUTPUT_DIR"
        
        # Run test
        if output=$("$BINARY_PATH" $args 2>&1); then
            print_result "$name" "FAIL" "Should fail without cluster access"
        else
            # Check if it handled the failure gracefully
            if echo "$output" | grep -q "Starting resource collection\|kubeconfig file not found\|neither 'oc' nor 'kubectl' command found"; then
                print_result "$name" "PASS" "Handles failure gracefully"
            else
                print_result "$name" "FAIL" "Unexpected error: $output"
            fi
        fi
    done
}

# Function to run performance tests
run_performance_tests() {
    print_section "Running Performance Tests"
    
    local test_dir="$TEST_OUTPUT_DIR/performance-test"
    mkdir -p "$test_dir"
    
    # Test execution time
    local start_time=$(date +%s.%N)
    if output=$("$BINARY_PATH" --output "$test_dir" --verbose 2>&1); then
        print_result "Performance Test" "FAIL" "Should fail without cluster access"
    else
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "0")
        
        if (( $(echo "$duration < 5.0" | bc -l 2>/dev/null || echo "1") )); then
            print_result "Performance Test" "PASS" "Execution completed in ${duration}s"
        else
            print_result "Performance Test" "FAIL" "Execution took too long: ${duration}s"
        fi
    fi
    
    # Clean up
    rm -rf "$test_dir"
}

# Function to run error handling tests
run_error_handling_tests() {
    print_section "Running Error Handling Tests"
    
    # Test various error conditions
    local error_tests=(
        "Invalid kubeconfig:--kubeconfig:/nonexistent/path"
        "Invalid output path:--output:/root/invalid/path"
        "Unknown flag:--unknown-flag"
    )
    
    for error_test in "${error_tests[@]}"; do
        IFS=':' read -r name args <<< "$error_test"
        
        if output=$("$BINARY_PATH" $args 2>&1); then
            print_result "$name" "FAIL" "Should fail with invalid input"
        else
            print_result "$name" "PASS" "Correctly handles error condition"
        fi
    done
}

# Function to generate test report
generate_test_report() {
    print_section "Generating Test Report"
    
    mkdir -p "$TEST_REPORTS_DIR"
    
    # Create summary report
    cat > "$TEST_REPORTS_DIR/test_summary.md" << EOF
# Test Summary Report

**Date:** $(date)
**Binary:** $BINARY_PATH
**Binary Size:** $(stat -c%s "$BINARY_PATH" 2>/dev/null || stat -f%z "$BINARY_PATH" 2>/dev/null) bytes

## Test Results

- **Total Tests:** $TOTAL_TESTS
- **Passed:** $PASSED_TESTS
- **Failed:** $FAILED_TESTS
- **Success Rate:** $(( (PASSED_TESTS * 100) / TOTAL_TESTS ))%

## Test Categories

### Prerequisites
- Binary exists and is executable
- Go version available
- Binary size within expected range

### Functionality Tests
- Help command output
- Command line argument validation
- Kubeconfig validation
- Output directory creation
- Clean mode functionality
- Single file mode functionality
- Verbose mode functionality

### Error Handling
- Invalid kubeconfig path
- Invalid output path
- Unknown command line flags
- Missing command tools (oc/kubectl)

### Performance
- Execution time within acceptable limits
- Memory usage reasonable

### Integration
- Complete workflow scenarios
- End-to-end functionality

## Recommendations

