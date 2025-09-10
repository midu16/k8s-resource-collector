package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/yaml"
)

var (
	kubeconfig string
	outputDir  string
	outputFile string
	verbose    bool
	singleFile bool
	clean      bool
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	flag.StringVar(&outputDir, "output", "./output", "Output directory for collected resources")
	flag.StringVar(&outputFile, "file", "", "Output file for single file mode")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&singleFile, "single-file", false, "Collect all resources to a single YAML file")
	flag.BoolVar(&clean, "clean", false, "Clean output directory before collection")
	flag.Parse()

	if err := runCollector(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCollector() error {
	// Determine output mode
	if outputFile != "" {
		singleFile = true
	} else if singleFile {
		outputFile = "./output/all-resources.yaml"
	}

	// Parse kubeconfig
	config, err := parseKubeConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Create clients
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	if singleFile {
		// Single file mode
		if outputFile == "" {
			outputFile = "./output/all-resources.yaml"
		}

		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Clean file if requested
		if clean {
			if err := os.Remove(outputFile); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to clean output file: %w", err)
			}
		}

		return collectAllResourcesToSingleFile(discoveryClient, dynamicClient, outputFile)
	} else {
		// Directory mode
		// Ensure output directory exists
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Clean directory if requested
		if clean {
			if err := cleanDirectory(outputDir); err != nil {
				return fmt.Errorf("failed to clean output directory: %w", err)
			}
		}

		return collectResources(discoveryClient, dynamicClient, outputDir)
	}
}

func parseKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	var configPath string

	// Priority: flag > environment variable > default location
	if kubeconfigPath != "" {
		configPath = kubeconfigPath
	} else if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
		configPath = envKubeconfig
	} else {
		configPath = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig file not found at %s", configPath)
	}

	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}

func collectResources(discovery discovery.DiscoveryInterface, dynamic dynamic.Interface, outputDir string) error {
	startTime := time.Now()
	
	if verbose {
		fmt.Printf("Starting resource collection to directory: %s\n", outputDir)
	}

	// Get all API resources
	resources, err := discovery.ServerPreferredResources()
	if err != nil {
		return fmt.Errorf("failed to discover API resources: %w", err)
	}

	collectedCount := 0
	errorCount := 0

	for _, resourceList := range resources {
		for _, resource := range resourceList.APIResources {
			// Skip subresources
			if strings.Contains(resource.Name, "/") {
				continue
			}

			// Only collect resources that support list and get verbs
			if !contains(resource.Verbs, "list") || !contains(resource.Verbs, "get") {
				continue
			}

			if verbose {
				fmt.Printf("Collecting resource: %s\n", resource.Name)
			}

			err := collectResource(dynamic, resource, resourceList.GroupVersion, outputDir)
			if err != nil {
				if verbose {
					fmt.Printf("  %s: ERROR - %v\n", resource.Name, err)
				}
				errorCount++
			} else {
				collectedCount++
			}
		}
	}

	// Print summary
	duration := time.Since(startTime)
	fmt.Printf("\n=== Collection Summary ===\n")
	fmt.Printf("Successfully collected: %d resources\n", collectedCount)
	fmt.Printf("Errors encountered: %d resources\n", errorCount)
	fmt.Printf("Output directory: %s\n", outputDir)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("========================\n")

	return nil
}

func collectResource(dynamic dynamic.Interface, resource metav1.APIResource, groupVersion string, outputDir string) error {
	// Parse group version
	gv, err := schema.ParseGroupVersion(groupVersion)
	if err != nil {
		return fmt.Errorf("failed to parse group version: %w", err)
	}

	// Create GVR
	gvr := schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: resource.Name,
	}

	// Get all instances of this resource across all namespaces
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	unstructuredList, err := dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get resource instances for %s: %w", resource.Name, err)
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(unstructuredList)
	if err != nil {
		return fmt.Errorf("failed to marshal %s to YAML: %w", err)
	}

	// Create filename and path
	filename := formatFilename(resource.Name, groupVersion)
	filePath := filepath.Join(outputDir, filename)

	// Create header
	header := formatHeader(resource.Name, groupVersion)
	finalYaml := header + string(yamlData)

	// Write to file
	err = os.WriteFile(filePath, []byte(finalYaml), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	if verbose {
		fmt.Printf("  %s: SUCCESS - Saved to %s\n", resource.Name, filePath)
	}

	return nil
}

func collectAllResourcesToSingleFile(discovery discovery.DiscoveryInterface, dynamic dynamic.Interface, outputFile string) error {
	startTime := time.Now()
	
	if verbose {
		fmt.Printf("Starting resource collection to single file: %s\n", outputFile)
	}

	// Get all API resources
	resources, err := discovery.ServerPreferredResources()
	if err != nil {
		return fmt.Errorf("failed to discover API resources: %w", err)
	}

	var allResourcesYaml strings.Builder
	collectedCount := 0
	errorCount := 0

	for _, resourceList := range resources {
		for _, resource := range resourceList.APIResources {
			// Skip subresources
			if strings.Contains(resource.Name, "/") {
				continue
			}

			// Only collect resources that support list and get verbs
			if !contains(resource.Verbs, "list") || !contains(resource.Verbs, "get") {
				continue
			}

			if verbose {
				fmt.Printf("Collecting resource: %s\n", resource.Name)
			}

			err := collectResourceToBuffer(dynamic, resource, resourceList.GroupVersion, &allResourcesYaml)
			if err != nil {
				if verbose {
					fmt.Printf("  %s: ERROR - %v\n", resource.Name, err)
				}
				errorCount++
			} else {
				collectedCount++
			}
		}
	}

	// Write all resources to file
	err = os.WriteFile(outputFile, []byte(allResourcesYaml.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputFile, err)
	}

	// Print summary
	duration := time.Since(startTime)
	fmt.Printf("\n=== Collection Summary ===\n")
	fmt.Printf("Successfully collected: %d resources\n", collectedCount)
	fmt.Printf("Errors encountered: %d resources\n", errorCount)
	fmt.Printf("Output file: %s\n", outputFile)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("========================\n")

	return nil
}

func collectResourceToBuffer(dynamic dynamic.Interface, resource metav1.APIResource, groupVersion string, buffer *strings.Builder) error {
	// Parse group version
	gv, err := schema.ParseGroupVersion(groupVersion)
	if err != nil {
		return fmt.Errorf("failed to parse group version: %w", err)
	}

	// Create GVR
	gvr := schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: resource.Name,
	}

	// Get all instances of this resource across all namespaces
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	unstructuredList, err := dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get resource instances for %s: %w", resource.Name, err)
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(unstructuredList)
	if err != nil {
		return fmt.Errorf("failed to marshal %s to YAML: %w", err)
	}

	// Add resource comment
	buffer.WriteString(fmt.Sprintf("--- # Resource: %s\n", resource.Name))
	buffer.WriteString(string(yamlData))
	buffer.WriteString("\n")

	return nil
}

func formatFilename(resourceName string, groupVersion string) string {
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
	
	sanitizedName := replacer.Replace(resourceName)
	
	if groupVersion != "" {
		// Add group version to filename
		sanitizedGroupVersion := replacer.Replace(groupVersion)
		return fmt.Sprintf("%s-%s.yaml", sanitizedGroupVersion, sanitizedName)
	}
	
	return fmt.Sprintf("%s.yaml", sanitizedName)
}

func formatHeader(resourceName string, groupVersion string) string {
	var header strings.Builder
	
	header.WriteString("# Generated by k8s-resource-collector\n")
	header.WriteString(fmt.Sprintf("# Generated at: %s\n", time.Now().Format(time.RFC3339)))
	header.WriteString(fmt.Sprintf("# Resource: %s\n", resourceName))
	if groupVersion != "" {
		header.WriteString(fmt.Sprintf("# Group Version: %s\n", groupVersion))
	}
	header.WriteString("# ---\n\n")
	
	return header.String()
}

func cleanDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to clean
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", entryPath, err)
		}
		if verbose {
			fmt.Printf("Removed: %s\n", entryPath)
		}
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
