# Enhanced Functional Testing for k8s-resource-collector

This directory contains comprehensive functional tests for the k8s-resource-collector tool. The testing suite provides multiple levels of testing to ensure the tool works correctly in various scenarios.

## Test Structure

```
tests/
├── README.md                    # This documentation
├── test_runner.sh              # Comprehensive shell-based test runner
├── simple_test_runner.sh        # Simple shell-based test runner
├── functional_test.go           # Go-based functional tests
├── test_config.yaml            # Test configuration file
├── test-output/                # Test output directory (created during tests)
└── test-reports/               # Test reports directory (created during tests)
```

## Test Categories

### 1. Basic Functionality Tests
- **Help Command**: Verifies the help output is displayed correctly
- **Invalid Arguments**: Tests that unknown flags are properly rejected
- **Command Line Validation**: Ensures proper argument parsing

### 2. Configuration Tests
- **Kubeconfig Validation**: Tests kubeconfig file path validation
- **Output Directory Creation**: Verifies output directories are created
- **Environment Variable Handling**: Tests KUBECONFIG environment variable

### 3. Mode Tests
- **Directory Mode**: Tests individual file creation for each resource type
- **Single File Mode**: Tests creation of a single file with all resources
- **Clean Mode**: Tests directory cleaning functionality

### 4. Flag Tests
- **Verbose Mode**: Tests verbose output functionality
- **All Flags Documented**: Verifies all flags are documented in help

### 5. Error Handling Tests
- **Invalid Kubeconfig Path**: Tests error handling for non-existent kubeconfig
- **Invalid Output Path**: Tests error handling for invalid output paths
- **Missing Command Tools**: Tests error handling when oc/kubectl not available

### 6. Performance Tests
- **Execution Time**: Verifies execution completes within reasonable time
- **Memory Usage**: Tests memory usage is reasonable

### 7. Integration Tests
- **Complete Workflows**: Tests end-to-end functionality
- **Cross-Platform Compatibility**: Tests on different platforms

## Running Tests

### Quick Tests (Recommended)
```bash
# Run enhanced functional tests
make test

# Or run directly
cd tests && ./simple_test_runner.sh
```

### Comprehensive Tests
```bash
# Run comprehensive functional tests
make test-comprehensive

# Or run directly
cd tests && ./test_runner.sh
```

### Go-based Tests
```bash
# Run Go-based functional tests
make test-go

# Or run directly
cd tests && go test -v -timeout 30s
```

### All Tests
```bash
# Run all test types
make test test-comprehensive test-go
```

## Test Configuration

The `test_config.yaml` file contains detailed test configuration including:

- Test suite metadata
- Binary expectations
- Test categories and scenarios
- Mock data definitions
- Performance thresholds
- Reporting settings

## Test Reports

Test reports are generated in the `test-reports/` directory:

- `test_summary.md`: Human-readable test summary
- `go_tests.json`: JSON format test results
- `junit.xml`: JUnit XML format (if available)

## Mock Data

The tests include mock data for:

- API resources (pods, services, configmaps, etc.)
- Sample resource definitions
- Mock cluster responses

## Test Environment

### Prerequisites
- Go 1.21+
- Built binary (`make build`)
- `oc` or `kubectl` command available (for some tests)

### Test Isolation
- Each test runs in isolation
- Test directories are created and cleaned up automatically
- No interference between test runs

### Error Handling
- Tests expect failures when no cluster access is available
- Tests verify graceful error handling
- Tests check for appropriate error messages

## Test Results Interpretation

### Success Criteria
- ✅ **PASS**: Test passed successfully
- ❌ **FAIL**: Test failed (needs investigation)
- ⚠️ **SKIP**: Test skipped (e.g., when real tools available)

### Common Test Scenarios

1. **Without Cluster Access**: Most tests expect to fail gracefully when no cluster is available
2. **With Invalid Input**: Tests verify proper error handling for invalid inputs
3. **With Valid Input**: Tests verify correct behavior with valid inputs

## Troubleshooting

### Test Failures

1. **Binary Not Found**: Run `make build` first
2. **Permission Issues**: Ensure test scripts are executable (`chmod +x tests/*.sh`)
3. **Directory Issues**: Check that test directories can be created

### Debugging Tests

1. **Verbose Output**: Add `-v` flag to see detailed test output
2. **Individual Tests**: Run specific test functions directly
3. **Test Artifacts**: Check `test-output/` directory for generated files

### Test Development

1. **Adding New Tests**: Follow existing patterns in test files
2. **Test Data**: Add mock data to `test_config.yaml`
3. **Test Reports**: Update report templates as needed

## Continuous Integration

The tests are designed to work in CI/CD environments:

- No external dependencies required
- Deterministic results
- Proper exit codes
- Comprehensive reporting

## Best Practices

1. **Test Isolation**: Each test should be independent
2. **Cleanup**: Always clean up test artifacts
3. **Error Handling**: Test both success and failure scenarios
4. **Documentation**: Document test scenarios and expected behavior
5. **Maintenance**: Keep tests updated with code changes

## Contributing

When adding new functionality:

1. Add corresponding tests
2. Update test configuration
3. Update documentation
4. Ensure all tests pass

## Support

For test-related issues:

1. Check this documentation
2. Review test output carefully
3. Check test configuration
4. Verify prerequisites are met
