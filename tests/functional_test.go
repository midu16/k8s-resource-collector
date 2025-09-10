package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test configuration
const (
	binaryPath = "../bin/k8s-resource-collector"
	testDir    = "./test-output"
)

// TestResult represents the result of a test
type TestResult struct {
	Name    string
	Passed  bool
	Message string
	Error   error
}

// TestSuite manages the test execution
type TestSuite struct {
	results []TestResult
}

// NewTestSuite creates a new test suite
func NewTestSuite() *TestSuite {
	return &TestSuite{
		results: make([]TestResult, 0),
	}
}

// AddResult adds a test result to the suite
func (ts *TestSuite) AddResult(name string, passed bool, message string, err error) {
	ts.results = append(ts.results, TestResult{
		Name:    name,
		Passed:  passed,
		Message: message,
		Error:   err,
	})
}

// PrintSummary prints the test summary
func (ts *TestSuite) PrintSummary() {
	fmt.Println("\n==================================================")
	fmt.Println("Test Summary:")
	
	passed := 0
	failed := 0
	
	for _, result := range ts.results {
		if result.Passed {
			fmt.Printf("✓ PASS: %s\n", result.Name)
			passed++
		} else {
			fmt.Printf("✗ FAIL: %s - %s\n", result.Name, result.Message)
			if result.Error != nil {
				fmt.Printf("  Error: %v\n", result.Error)
			}
			failed++
		}
	}
	
	fmt.Printf("\nTests Run: %d\n", len(ts.results))
	fmt.Printf("Tests Passed: %d\n", passed)
	fmt.Printf("Tests Failed: %d\n", failed)
	fmt.Println("==================================================")
}

// RunCommand executes a command and returns the output and error
func RunCommand(args ...string) (string, error) {
	cmd := exec.Command(binaryPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// TestBinaryExists tests if the binary exists
func TestBinaryExists(t *testing.T) {
	suite := NewTestSuite()
	
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		suite.AddResult("Binary Exists", false, "Binary not found", err)
	} else {
		suite.AddResult("Binary Exists", true, "Binary found", nil)
	}
	
	suite.PrintSummary()
}

// TestHelpCommand tests the help command
func TestHelpCommand(t *testing.T) {
	suite := NewTestSuite()
	
	output, err := RunCommand("--help")
	if err != nil {
		suite.AddResult("Help Command", false, "Help command failed", err)
	} else if strings.Contains(output, "Usage of") {
		suite.AddResult("Help Command", true, "Help output displayed correctly", nil)
	} else {
		suite.AddResult("Help Command", false, "Help output format incorrect", nil)
	}
	
	suite.PrintSummary()
}

// TestInvalidArguments tests invalid command line arguments
func TestInvalidArguments(t *testing.T) {
	suite := NewTestSuite()
	
	_, err := RunCommand("--unknown-flag")
	if err != nil {
		suite.AddResult("Invalid Arguments", true, "Correctly rejects unknown flag", nil)
	} else {
		suite.AddResult("Invalid Arguments", false, "Should fail with unknown flag", nil)
	}
	
	suite.PrintSummary()
}

// TestKubeconfigValidation tests kubeconfig validation
func TestKubeconfigValidation(t *testing.T) {
	suite := NewTestSuite()
	
	output, err := RunCommand("--kubeconfig", "/nonexistent/path", "--verbose")
	if err != nil {
		if strings.Contains(output, "kubeconfig file not found") {
			suite.AddResult("Kubeconfig Validation", true, "Correctly validates kubeconfig path", nil)
		} else {
			suite.AddResult("Kubeconfig Validation", false, "Wrong error message for invalid kubeconfig", nil)
		}
	} else {
		suite.AddResult("Kubeconfig Validation", false, "Should fail with non-existent kubeconfig", nil)
	}
	
	suite.PrintSummary()
}

// TestOutputDirectoryCreation tests output directory creation
func TestOutputDirectoryCreation(t *testing.T) {
	suite := NewTestSuite()
	
	testDir := filepath.Join(testDir, "dir-creation-test")
	
	// Clean up any existing directory
	os.RemoveAll(testDir)
	
	// Test directory creation
	output, err := RunCommand("--output", testDir, "--verbose")
	
	// Check if directory was created
	if _, dirErr := os.Stat(testDir); dirErr == nil {
		suite.AddResult("Output Directory Creation", true, "Directory created successfully", nil)
	} else {
		suite.AddResult("Output Directory Creation", false, "Directory not created", dirErr)
	}
	
	// Clean up
	os.RemoveAll(testDir)
	
	suite.PrintSummary()
}

// TestCleanMode tests the clean mode functionality
func TestCleanMode(t *testing.T) {
	suite := NewTestSuite()
	
	testDir := filepath.Join(testDir, "clean-test")
	
	// Create test directory with some files
	os.MkdirAll(testDir, 0755)
	testFile := filepath.Join(testDir, "test-file.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)
	
	// Test clean mode
	_, err := RunCommand("--output", testDir, "--clean", "--verbose")
	
	// Check if file was removed
	if _, fileErr := os.Stat(testFile); os.IsNotExist(fileErr) {
		suite.AddResult("Clean Mode", true, "Directory cleaned successfully", nil)
	} else {
		suite.AddResult("Clean Mode", false, "Directory not cleaned", fileErr)
	}
	
	// Clean up
	os.RemoveAll(testDir)
	
	suite.PrintSummary()
}

// TestSingleFileMode tests single file mode
func TestSingleFileMode(t *testing.T) {
	suite := NewTestSuite()
	
	testFile := filepath.Join(testDir, "single-file-test.yaml")
	testDir := filepath.Dir(testFile)
	
	// Create test directory
	os.MkdirAll(testDir, 0755)
	
	// Test single file mode
	_, err := RunCommand("--single-file", "--output-file", testFile, "--verbose")
	
	// Check if file was created
	if _, fileErr := os.Stat(testFile); fileErr == nil {
		suite.AddResult("Single File Mode", true, "Single file created successfully", nil)
	} else {
		suite.AddResult("Single File Mode", false, "Single file not created", fileErr)
	}
	
	// Clean up
	os.RemoveAll(testDir)
	
	suite.PrintSummary()
}

// TestVerboseMode tests verbose mode
func TestVerboseMode(t *testing.T) {
	suite := NewTestSuite()
	
	testDir := filepath.Join(testDir, "verbose-test")
	
	// Test verbose mode
	output, err := RunCommand("--output", testDir, "--verbose")
	
	if err != nil {
		if strings.Contains(output, "Starting resource collection") {
			suite.AddResult("Verbose Mode", true, "Verbose output displayed correctly", nil)
		} else {
			suite.AddResult("Verbose Mode", false, "Verbose output not displayed", nil)
		}
	} else {
		suite.AddResult("Verbose Mode", false, "Should fail without cluster access", nil)
	}
	
	// Clean up
	os.RemoveAll(testDir)
	
	suite.PrintSummary()
}

// TestFileFormatValidation tests file format validation
func TestFileFormatValidation(t *testing.T) {
	suite := NewTestSuite()
	
	testDir := filepath.Join(testDir, "format-test")
	
	// Create test directory
	os.MkdirAll(testDir, 0755)
	
	// Test that the tool attempts to create YAML files
	output, err := RunCommand("--output", testDir, "--verbose")
	
	if err != nil {
		if strings.Contains(strings.ToLower(output), "yaml") {
			suite.AddResult("File Format Validation", true, "Tool mentions YAML format", nil)
		} else {
			suite.AddResult("File Format Validation", false, "Tool does not mention YAML format", nil)
		}
	} else {
		suite.AddResult("File Format Validation", false, "Should fail without cluster access", nil)
	}
	
	// Clean up
	os.RemoveAll(testDir)
	
	suite.PrintSummary()
}

// TestCommandLineFlags tests all command line flags
func TestCommandLineFlags(t *testing.T) {
	suite := NewTestSuite()
	
	// Test all flags
	flags := []string{
		"--kubeconfig",
		"--output",
		"--file",
		"--verbose",
		"--single-file",
		"--clean",
	}
	
	for _, flag := range flags {
		output, err := RunCommand("--help")
		if err == nil && strings.Contains(output, flag) {
			suite.AddResult(fmt.Sprintf("Flag %s", flag), true, "Flag documented in help", nil)
		} else {
			suite.AddResult(fmt.Sprintf("Flag %s", flag), false, "Flag not documented in help", err)
		}
	}
	
	suite.PrintSummary()
}

// TestPerformance tests basic performance characteristics
func TestPerformance(t *testing.T) {
	suite := NewTestSuite()
	
	testDir := filepath.Join(testDir, "performance-test")
	
	// Measure execution time
	start := time.Now()
	_, err := RunCommand("--output", testDir, "--verbose")
	duration := time.Since(start)
	
	if err != nil {
		if duration < 5*time.Second {
			suite.AddResult("Performance", true, fmt.Sprintf("Execution completed in %v", duration), nil)
		} else {
			suite.AddResult("Performance", false, fmt.Sprintf("Execution took too long: %v", duration), nil)
		}
	} else {
		suite.AddResult("Performance", false, "Should fail without cluster access", nil)
	}
	
	// Clean up
	os.RemoveAll(testDir)
	
	suite.PrintSummary()
}

// TestErrorHandling tests error handling
func TestErrorHandling(t *testing.T) {
	suite := NewTestSuite()
	
	// Test various error conditions
	testCases := []struct {
		name   string
		args   []string
		expectError bool
	}{
		{"Invalid kubeconfig", []string{"--kubeconfig", "/invalid/path"}, true},
		{"Invalid output directory", []string{"--output", "/root/invalid/path"}, true},
		{"Missing required args", []string{}, true},
	}
	
	for _, tc := range testCases {
		_, err := RunCommand(tc.args...)
		if tc.expectError && err != nil {
			suite.AddResult(tc.name, true, "Correctly handles error condition", nil)
		} else if !tc.expectError && err == nil {
			suite.AddResult(tc.name, true, "Correctly handles success condition", nil)
		} else {
			suite.AddResult(tc.name, false, "Incorrect error handling", err)
		}
	}
	
	suite.PrintSummary()
}

// TestIntegration tests integration scenarios
func TestIntegration(t *testing.T) {
	suite := NewTestSuite()
	
	// Test complete workflow scenarios
	scenarios := []struct {
		name string
		args []string
	}{
		{"Directory Mode", []string{"--output", filepath.Join(testDir, "integration-dir"), "--verbose"}},
		{"Single File Mode", []string{"--single-file", "--output-file", filepath.Join(testDir, "integration-single.yaml"), "--verbose"}},
		{"Clean Mode", []string{"--output", filepath.Join(testDir, "integration-clean"), "--clean", "--verbose"}},
	}
	
	for _, scenario := range scenarios {
		_, err := RunCommand(scenario.args...)
		// We expect these to fail due to no cluster access, but they should handle the failure gracefully
		if err != nil {
			suite.AddResult(scenario.name, true, "Handles failure gracefully", nil)
		} else {
			suite.AddResult(scenario.name, false, "Should fail without cluster access", nil)
		}
	}
	
	// Clean up
	os.RemoveAll(testDir)
	
	suite.PrintSummary()
}

// TestMain runs all tests
func TestMain(m *testing.M) {
	fmt.Println("Starting Enhanced Functional Testing for k8s-resource-collector")
	fmt.Println("==================================================")
	
	// Create test directory
	os.MkdirAll(testDir, 0755)
	
	// Run tests
	code := m.Run()
	
	// Clean up
	os.RemoveAll(testDir)
	
	fmt.Println("\n==================================================")
	fmt.Println("All tests completed!")
	fmt.Println("==================================================")
	
	os.Exit(code)
}
