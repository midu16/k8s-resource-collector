#!/bin/bash

# Enhanced Functional Testing for k8s-resource-collector
# This script provides comprehensive testing of the tool's functionality

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
BINARY_PATH="../bin/k8s-resource-collector"
TEST_OUTPUT_DIR="./test-output"
TEST_KUBECONFIG="./test-kubeconfig"
MOCK_CLUSTER_DATA="./mock-cluster-data"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Function to print test results
print_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name - $message"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Function to check if binary exists
check_binary() {
    if [ ! -f "$BINARY_PATH" ]; then
        echo -e "${RED}Error: Binary not found at $BINARY_PATH${NC}"
        echo "Please run 'make build' first"
        exit 1
    fi
}

# Function to create mock kubeconfig
create_mock_kubeconfig() {
    cat > "$TEST_KUBECONFIG" << 'KUBECONFIG_EOF'
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://mock-cluster.example.com
  name: mock-cluster
contexts:
- context:
    cluster: mock-cluster
    user: mock-user
  name: mock-context
current-context: mock-context
users:
- name: mock-user
  user:
    token: mock-token
KUBECONFIG_EOF
}

# Function to create mock cluster data
create_mock_cluster_data() {
    mkdir -p "$MOCK_CLUSTER_DATA"
    
    # Create mock API resources response
    cat > "$MOCK_CLUSTER_DATA/api-resources.txt" << 'API_RESOURCES_EOF'
pods
services
configmaps
deployments
replicasets
API_RESOURCES_EOF

    # Create mock pod data
    cat > "$MOCK_CLUSTER_DATA/pods.yaml" << 'PODS_EOF'
apiVersion: v1
kind: List
items:
- apiVersion: v1
  kind: Pod
  metadata:
    name: test-pod-1
    namespace: default
  spec:
    containers:
    - name: test-container
      image: nginx:latest
- apiVersion: v1
  kind: Pod
  metadata:
    name: test-pod-2
    namespace: kube-system
  spec:
    containers:
    - name: system-container
      image: pause:latest
PODS_EOF

    # Create mock service data
    cat > "$MOCK_CLUSTER_DATA/services.yaml" << 'SERVICES_EOF'
apiVersion: v1
kind: List
items:
- apiVersion: v1
  kind: Service
  metadata:
    name: test-service
    namespace: default
  spec:
    ports:
    - port: 80
      targetPort: 8080
SERVICES_EOF

    # Create mock configmap data
    cat > "$MOCK_CLUSTER_DATA/configmaps.yaml" << 'CONFIGMAPS_EOF'
apiVersion: v1
kind: List
items:
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: test-configmap
    namespace: default
  data:
    key1: value1
    key2: value2
CONFIGMAPS_EOF
}

# Function to create mock oc/kubectl commands
create_mock_commands() {
    # Create a mock oc command that returns our test data
    cat > "./mock-oc" << 'MOCK_OC_EOF'
#!/bin/bash

case "$1" in
    "api-resources")
        if [ "$2" = "--verbs=list,get" ] && [ "$3" = "-o" ] && [ "$4" = "name" ]; then
            cat "../mock-cluster-data/api-resources.txt"
            exit 0
        fi
        ;;
    "get")
        resource="$2"
        if [ "$3" = "--all-namespaces" ] && [ "$4" = "-o" ] && [ "$5" = "yaml" ]; then
            case "$resource" in
                "pods")
                    cat "../mock-cluster-data/pods.yaml"
                    exit 0
                    ;;
                "services")
                    cat "../mock-cluster-data/services.yaml"
                    exit 0
                    ;;
                "configmaps")
                    cat "../mock-cluster-data/configmaps.yaml"
                    exit 0
                    ;;
                "deployments")
                    echo "apiVersion: v1
kind: List
items: []"
                    exit 0
                    ;;
                "replicasets")
                    echo "apiVersion: v1
kind: List
items: []"
                    exit 0
                    ;;
            esac
        fi
        ;;
esac

echo "Error: Unknown command or arguments: $*" >&2
exit 1
MOCK_OC_EOF

    chmod +x "./mock-oc"
}

# Function to test help command
test_help_command() {
    echo -e "${BLUE}Testing help command...${NC}"
    
    if output=$("$BINARY_PATH" --help 2>&1); then
        if echo "$output" | grep -q "Usage of"; then
            print_result "Help Command" "PASS" "Help output displayed correctly"
        else
            print_result "Help Command" "FAIL" "Help output format incorrect"
        fi
    else
        print_result "Help Command" "FAIL" "Help command failed"
    fi
}

# Function to test invalid arguments
test_invalid_arguments() {
    echo -e "${BLUE}Testing invalid arguments...${NC}"
    
    # Test unknown flag
    if output=$("$BINARY_PATH" --unknown-flag 2>&1); then
        print_result "Invalid Arguments" "FAIL" "Should fail with unknown flag"
    else
        print_result "Invalid Arguments" "PASS" "Correctly rejects unknown flag"
    fi
}

# Function to test kubeconfig validation
test_kubeconfig_validation() {
    echo -e "${BLUE}Testing kubeconfig validation...${NC}"
    
    # Test with non-existent kubeconfig
    if output=$("$BINARY_PATH" --kubeconfig "/nonexistent/path" --verbose 2>&1); then
        print_result "Kubeconfig Validation" "FAIL" "Should fail with non-existent kubeconfig"
    else
        if echo "$output" | grep -q "kubeconfig file not found"; then
            print_result "Kubeconfig Validation" "PASS" "Correctly validates kubeconfig path"
        else
            print_result "Kubeconfig Validation" "FAIL" "Wrong error message for invalid kubeconfig"
        fi
    fi
}

# Function to test output directory creation
test_output_directory_creation() {
    echo -e "${BLUE}Testing output directory creation...${NC}"
    
    local test_dir="$TEST_OUTPUT_DIR/dir-creation-test"
    
    # Clean up any existing directory
    rm -rf "$test_dir"
    
    # Test directory creation (this will fail due to no cluster access, but should create directory)
    if output=$("$BINARY_PATH" --output "$test_dir" --verbose 2>&1); then
        if [ -d "$test_dir" ]; then
            print_result "Output Directory Creation" "PASS" "Directory created successfully"
        else
            print_result "Output Directory Creation" "FAIL" "Directory not created"
        fi
    else
        # Even if the command fails, the directory should be created
        if [ -d "$test_dir" ]; then
            print_result "Output Directory Creation" "PASS" "Directory created even on failure"
        else
            print_result "Output Directory Creation" "FAIL" "Directory not created on failure"
        fi
    fi
    
    # Clean up
    rm -rf "$test_dir"
}

# Function to test clean mode
test_clean_mode() {
    echo -e "${BLUE}Testing clean mode...${NC}"
    
    local test_dir="$TEST_OUTPUT_DIR/clean-test"
    
    # Create test directory with some files
    mkdir -p "$test_dir"
    echo "test content" > "$test_dir/test-file.txt"
    
    # Test clean mode (will fail due to no cluster access, but should clean directory)
    if output=$("$BINARY_PATH" --output "$test_dir" --clean --verbose 2>&1); then
        if [ ! -f "$test_dir/test-file.txt" ]; then
            print_result "Clean Mode" "PASS" "Directory cleaned successfully"
        else
            print_result "Clean Mode" "FAIL" "Directory not cleaned"
        fi
    else
        # Check if clean happened even on failure
        if [ ! -f "$test_dir/test-file.txt" ]; then
            print_result "Clean Mode" "PASS" "Directory cleaned even on failure"
        else
            print_result "Clean Mode" "FAIL" "Directory not cleaned on failure"
        fi
    fi
    
    # Clean up
    rm -rf "$test_dir"
}

# Function to test single file mode
test_single_file_mode() {
    echo -e "${BLUE}Testing single file mode...${NC}"
    
    local test_file="$TEST_OUTPUT_DIR/single-file-test.yaml"
    local test_dir=$(dirname "$test_file")
    
    # Create test directory
    mkdir -p "$test_dir"
    
    # Test single file mode (will fail due to no cluster access, but should create file)
    if output=$("$BINARY_PATH" --single-file --output-file "$test_file" --verbose 2>&1); then
        if [ -f "$test_file" ]; then
            print_result "Single File Mode" "PASS" "Single file created successfully"
        else
            print_result "Single File Mode" "FAIL" "Single file not created"
        fi
    else
        # Check if file was created even on failure
        if [ -f "$test_file" ]; then
            print_result "Single File Mode" "PASS" "Single file created even on failure"
        else
            print_result "Single File Mode" "FAIL" "Single file not created on failure"
        fi
    fi
    
    # Clean up
    rm -rf "$test_dir"
}

# Function to test verbose mode
test_verbose_mode() {
    echo -e "${BLUE}Testing verbose mode...${NC}"
    
    local test_dir="$TEST_OUTPUT_DIR/verbose-test"
    
    # Test verbose mode (will fail due to no cluster access, but should show verbose output)
    if output=$("$BINARY_PATH" --output "$test_dir" --verbose 2>&1); then
        print_result "Verbose Mode" "FAIL" "Should fail without cluster access"
    else
        if echo "$output" | grep -q "Starting resource collection"; then
            print_result "Verbose Mode" "PASS" "Verbose output displayed correctly"
        else
            print_result "Verbose Mode" "FAIL" "Verbose output not displayed"
        fi
    fi
    
    # Clean up
    rm -rf "$test_dir"
}

# Function to test with mock commands (if available)
test_with_mock_commands() {
    echo -e "${BLUE}Testing with mock commands...${NC}"
    
    # Check if we can create mock commands
    if command -v oc >/dev/null 2>&1 || command -v kubectl >/dev/null 2>&1; then
        print_result "Mock Commands Test" "SKIP" "Real oc/kubectl available, skipping mock test"
        return
    fi
    
    # Create mock commands and test
    create_mock_commands
    create_mock_cluster_data
    
    # Temporarily add mock to PATH
    export PATH="$(pwd):$PATH"
    
    local test_dir="$TEST_OUTPUT_DIR/mock-test"
    
    # Test with mock commands
    if output=$("$BINARY_PATH" --output "$test_dir" --verbose 2>&1); then
        if [ -d "$test_dir" ] && [ -f "$test_dir/pods.yaml" ]; then
            print_result "Mock Commands Test" "PASS" "Mock commands worked correctly"
        else
            print_result "Mock Commands Test" "FAIL" "Mock commands did not work"
        fi
    else
        print_result "Mock Commands Test" "FAIL" "Mock commands failed: $output"
    fi
    
    # Clean up
    rm -rf "$test_dir"
    rm -f "./mock-oc"
    rm -rf "$MOCK_CLUSTER_DATA"
}

# Function to test file format validation
test_file_format_validation() {
    echo -e "${BLUE}Testing file format validation...${NC}"
    
    local test_dir="$TEST_OUTPUT_DIR/format-test"
    
    # Create a test file to check if the tool creates proper YAML files
    mkdir -p "$test_dir"
    
    # Test that the tool attempts to create YAML files (will fail due to no cluster access)
    if output=$("$BINARY_PATH" --output "$test_dir" --verbose 2>&1); then
        print_result "File Format Validation" "FAIL" "Should fail without cluster access"
    else
        # Check if the tool mentions YAML in the output
        if echo "$output" | grep -qi "yaml"; then
            print_result "File Format Validation" "PASS" "Tool mentions YAML format"
        else
            print_result "File Format Validation" "FAIL" "Tool does not mention YAML format"
        fi
    fi
    
    # Clean up
    rm -rf "$test_dir"
}

# Function to run all tests
run_all_tests() {
    echo -e "${YELLOW}Starting Enhanced Functional Testing for k8s-resource-collector${NC}"
    echo "=================================================="
    
    # Check prerequisites
    check_binary
    
    # Create test environment
    mkdir -p "$TEST_OUTPUT_DIR"
    create_mock_kubeconfig
    
    # Run tests
    test_help_command
    test_invalid_arguments
    test_kubeconfig_validation
    test_output_directory_creation
    test_clean_mode
    test_single_file_mode
    test_verbose_mode
    test_file_format_validation
    test_with_mock_commands
    
    # Print summary
    echo ""
    echo "=================================================="
    echo -e "${YELLOW}Test Summary:${NC}"
    echo -e "Tests Run: $TESTS_RUN"
    echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}All tests passed! 🎉${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed. Please review the output above.${NC}"
        exit 1
    fi
}

# Function to clean up test environment
cleanup() {
    echo -e "${BLUE}Cleaning up test environment...${NC}"
    rm -rf "$TEST_OUTPUT_DIR"
    rm -f "$TEST_KUBECONFIG"
    rm -f "./mock-oc"
    rm -rf "$MOCK_CLUSTER_DATA"
}

# Main execution
case "${1:-}" in
    "cleanup")
        cleanup
        ;;
    "help")
        echo "Usage: $0 [cleanup|help]"
        echo "  cleanup: Clean up test environment"
        echo "  help: Show this help message"
        echo "  (no args): Run all tests"
        ;;
    *)
        run_all_tests
        ;;
esac
