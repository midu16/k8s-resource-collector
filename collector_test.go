package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var kubeconfig string

type ResourceCollector struct {
	outputDir string
	verbose   bool
}

func getKubeConfig() (*rest.Config, error) {
	var kubeconfigPath string

	// Priority: flag > environment variable > default location
	if kubeconfig != "" {
		kubeconfigPath = kubeconfig
	} else if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
		kubeconfigPath = envKubeconfig
	} else {
		kubeconfigPath = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	// Check if file exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, err
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func sanitizeFilename(name string) string {
	// Replace characters that are not safe for filenames
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
		" ", "-",
	)
	return replacer.Replace(name)
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pods", "pods"},
		{"services", "services"},
		{"configmaps", "configmaps"},
		{"custom-resource/definitions", "custom-resource-definitions"},
		{"test:resource", "test-resource"},
		{"resource*with?bad<chars>", "resource-with-bad-chars-"},
	}

	for _, test := range tests {
		result := sanitizeFilename(test.input)
		if result != test.expected {
			t.Errorf("sanitizeFilename(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestContains(t *testing.T) {
	tests := []struct {
		slice    []string
		item     string
		expected bool
	}{
		{[]string{"get", "list", "watch"}, "list", true},
		{[]string{"get", "list", "watch"}, "create", false},
		{[]string{}, "get", false},
		{[]string{"get"}, "get", true},
	}

	for _, test := range tests {
		result := contains(test.slice, test.item)
		if result != test.expected {
			t.Errorf("contains(%v, %s) = %t, expected %t", test.slice, test.item, result, test.expected)
		}
	}
}

func TestGetKubeConfig(t *testing.T) {
	// Test with non-existent kubeconfig
	originalKubeconfig := kubeconfig
	kubeconfig = "/non/existent/path"
	
	_, err := getKubeConfig()
	if err == nil {
		t.Error("Expected error for non-existent kubeconfig, got nil")
	}
	
	kubeconfig = originalKubeconfig
}

func TestCreateOutputDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testOutputDir := filepath.Join(tempDir, "test-output")
	
	// Test directory creation
	err := os.MkdirAll(testOutputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	
	// Verify directory exists
	if _, err := os.Stat(testOutputDir); os.IsNotExist(err) {
		t.Error("Output directory was not created")
	}
	
	// Test with existing directory (should not error)
	err = os.MkdirAll(testOutputDir, 0755)
	if err != nil {
		t.Errorf("Failed to create existing directory: %v", err)
	}
}

func TestResourceCollectorInitialization(t *testing.T) {
	// This test would require a mock Kubernetes client
	// For now, we'll test the basic structure
	
	tempDir := t.TempDir()
	
	collector := &ResourceCollector{
		outputDir: tempDir,
		verbose:   true,
	}
	
	if collector.outputDir != tempDir {
		t.Errorf("Expected outputDir %s, got %s", tempDir, collector.outputDir)
	}
	
	if !collector.verbose {
		t.Error("Expected verbose to be true")
	}
}

// Integration test that would require a running cluster
// This is commented out as it requires actual cluster access
/*
func TestCollectResourcesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// This test would require:
	// 1. A running Kubernetes cluster
	// 2. Proper kubeconfig setup
	// 3. Mock or test cluster setup
	
	tempDir := t.TempDir()
	outputDir = tempDir
	
	// Set up test kubeconfig if available
	if os.Getenv("KUBECONFIG") == "" {
		t.Skip("Skipping integration test: no KUBECONFIG set")
	}
	
	err := runCollector(nil, nil)
	if err != nil {
		t.Errorf("CollectResources failed: %v", err)
	}
	
	// Verify some files were created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}
	
	if len(files) == 0 {
		t.Error("No files were created in output directory")
	}
	
	// Check that files are YAML files
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".yaml") {
			t.Errorf("Expected YAML file, got: %s", file.Name())
		}
	}
}
*/