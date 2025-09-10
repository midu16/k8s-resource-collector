#!/bin/bash

# Simple Enhanced Functional Testing Runner
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BINARY_PATH="$PROJECT_ROOT/bin/k8s-resource-collector"

echo -e "${YELLOW}Enhanced Functional Testing for k8s-resource-collector${NC}"
echo "=================================================="

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo -e "${RED}Error: Binary not found at $BINARY_PATH${NC}"
    echo "Please run 'make build' first"
    exit 1
fi

echo -e "${GREEN}✓ Binary found: $BINARY_PATH${NC}"

# Test 1: Help command
echo -e "${BLUE}Testing help command...${NC}"
if output=$("$BINARY_PATH" --help 2>&1); then
    if echo "$output" | grep -q "Usage of"; then
        echo -e "${GREEN}✓ Help command works${NC}"
    else
        echo -e "${RED}✗ Help command format incorrect${NC}"
    fi
else
    echo -e "${RED}✗ Help command failed${NC}"
fi

# Test 2: Invalid arguments
echo -e "${BLUE}Testing invalid arguments...${NC}"
if "$BINARY_PATH" --unknown-flag 2>/dev/null; then
    echo -e "${RED}✗ Should reject unknown flag${NC}"
else
    echo -e "${GREEN}✓ Correctly rejects unknown flag${NC}"
fi

# Test 3: Kubeconfig validation
echo -e "${BLUE}Testing kubeconfig validation...${NC}"
if output=$("$BINARY_PATH" --kubeconfig "/nonexistent/path" --verbose 2>&1); then
    echo -e "${RED}✗ Should fail with non-existent kubeconfig${NC}"
else
    if echo "$output" | grep -q "kubeconfig file not found"; then
        echo -e "${GREEN}✓ Correctly validates kubeconfig path${NC}"
    else
        echo -e "${RED}✗ Wrong error message for invalid kubeconfig${NC}"
    fi
fi

# Test 4: Output directory creation
echo -e "${BLUE}Testing output directory creation...${NC}"
test_dir="./test-output/dir-test"
rm -rf "$test_dir"
if output=$("$BINARY_PATH" --output "$test_dir" --verbose 2>&1); then
    echo -e "${RED}✗ Should fail without cluster access${NC}"
else
    if [ -d "$test_dir" ]; then
        echo -e "${GREEN}✓ Directory created successfully${NC}"
    else
        echo -e "${RED}✗ Directory not created${NC}"
    fi
fi
rm -rf "$test_dir"

# Test 5: Single file mode
echo -e "${BLUE}Testing single file mode...${NC}"
test_file="./test-output/single-test.yaml"
test_dir=$(dirname "$test_file")
mkdir -p "$test_dir"
if output=$("$BINARY_PATH" --single-file --output-file "$test_file" --verbose 2>&1); then
    echo -e "${RED}✗ Should fail without cluster access${NC}"
else
    if [ -f "$test_file" ]; then
        echo -e "${GREEN}✓ Single file created successfully${NC}"
    else
        echo -e "${RED}✗ Single file not created${NC}"
    fi
fi
rm -rf "$test_dir"

# Test 6: Clean mode
echo -e "${BLUE}Testing clean mode...${NC}"
test_dir="./test-output/clean-test"
mkdir -p "$test_dir"
echo "test content" > "$test_dir/test-file.txt"
if output=$("$BINARY_PATH" --output "$test_dir" --clean --verbose 2>&1); then
    echo -e "${RED}✗ Should fail without cluster access${NC}"
else
    if [ ! -f "$test_dir/test-file.txt" ]; then
        echo -e "${GREEN}✓ Directory cleaned successfully${NC}"
    else
        echo -e "${RED}✗ Directory not cleaned${NC}"
    fi
fi
rm -rf "$test_dir"

# Test 7: Verbose mode
echo -e "${BLUE}Testing verbose mode...${NC}"
test_dir="./test-output/verbose-test"
if output=$("$BINARY_PATH" --output "$test_dir" --verbose 2>&1); then
    echo -e "${RED}✗ Should fail without cluster access${NC}"
else
    if echo "$output" | grep -q "Starting resource collection"; then
        echo -e "${GREEN}✓ Verbose output displayed correctly${NC}"
    else
        echo -e "${RED}✗ Verbose output not displayed${NC}"
    fi
fi
rm -rf "$test_dir"

echo ""
echo -e "${YELLOW}=================================================="
echo "Enhanced functional testing completed!"
echo "=================================================="
echo -e "${NC}"
