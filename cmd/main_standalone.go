package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	kubeconfig string
	outputDir  string
	outputFile string
	verbose    bool
	singleFile bool
	clean      bool
	importFile string
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	flag.StringVar(&outputDir, "output", "./output", "Output directory for collected resources")
	flag.StringVar(&outputFile, "file", "", "Output file for single file mode")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&singleFile, "single-file", false, "Collect all resources to a single YAML file")
	flag.BoolVar(&clean, "clean", false, "Clean output directory before collection")
	flag.StringVar(&importFile, "import", "", "Import all-resources.yaml file and split ClusterResources into individual files")
	flag.Parse()

	if err := runCollector(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCollector() error {
	// Handle import mode first
	if importFile != "" {
		return importAllResourcesFile(importFile)
	}

	// Determine output mode
	if outputFile != "" {
		singleFile = true
	} else if singleFile {
		outputFile = "./output/all-resources.yaml"
	}

	// Check if oc or kubectl is available
	ocCmd, err := exec.LookPath("oc")
	if err != nil {
		kubectlCmd, err := exec.LookPath("kubectl")
		if err != nil {
			return fmt.Errorf("neither 'oc' nor 'kubectl' command found in PATH")
		}
		ocCmd = kubectlCmd
	}

	if verbose {
		fmt.Printf("Using command: %s\n", ocCmd)
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

		return collectAllResourcesToSingleFile(ocCmd, outputFile)
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

		return collectResources(ocCmd, outputDir)
	}
}

func collectResources(ocCmd string, outputDir string) error {
	startTime := time.Now()
	
	if verbose {
		fmt.Printf("Starting resource collection to directory: %s\n", outputDir)
	}

	// Get all API resources
	resources, err := getAPIResources(ocCmd)
	if err != nil {
		return fmt.Errorf("failed to get API resources: %w", err)
	}

	collectedCount := 0
	errorCount := 0

	for _, resource := range resources {
		if verbose {
			fmt.Printf("Collecting resource: %s\n", resource)
		}

		err := collectResource(ocCmd, resource, outputDir)
		if err != nil {
			if verbose {
				fmt.Printf("  %s: ERROR - %v\n", resource, err)
			}
			errorCount++
		} else {
			collectedCount++
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

func collectResource(ocCmd string, resource string, outputDir string) error {
	// Create filename
	filename := formatFilename(resource)
	filePath := filepath.Join(outputDir, filename)

	// Create header
	header := formatHeader(resource)
	
	// Get resource data
	cmd := exec.Command(ocCmd, "get", resource, "--all-namespaces", "-o", "yaml")
	if kubeconfig != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	}
	
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get resource %s: %w", resource, err)
	}

	// Write to file
	finalYaml := header + string(output)
	err = os.WriteFile(filePath, []byte(finalYaml), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	if verbose {
		fmt.Printf("  %s: SUCCESS - Saved to %s\n", resource, filePath)
	}

	return nil
}

func collectAllResourcesToSingleFile(ocCmd string, outputFile string) error {
	startTime := time.Now()
	
	if verbose {
		fmt.Printf("Starting resource collection to single file: %s\n", outputFile)
	}

	// Get all API resources
	resources, err := getAPIResources(ocCmd)
	if err != nil {
		return fmt.Errorf("failed to get API resources: %w", err)
	}

	var allResourcesYaml strings.Builder
	collectedCount := 0
	errorCount := 0

	for _, resource := range resources {
		if verbose {
			fmt.Printf("Collecting resource: %s\n", resource)
		}

		err := collectResourceToBuffer(ocCmd, resource, &allResourcesYaml)
		if err != nil {
			if verbose {
				fmt.Printf("  %s: ERROR - %v\n", resource, err)
			}
			errorCount++
		} else {
			collectedCount++
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

func collectResourceToBuffer(ocCmd string, resource string, buffer *strings.Builder) error {
	// Add resource comment
	buffer.WriteString(fmt.Sprintf("--- # Resource: %s\n", resource))
	
	// Get resource data
	cmd := exec.Command(ocCmd, "get", resource, "--all-namespaces", "-o", "yaml")
	if kubeconfig != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	}
	
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get resource %s: %w", resource, err)
	}

	buffer.WriteString(string(output))
	buffer.WriteString("\n")

	return nil
}

func getAPIResources(ocCmd string) ([]string, error) {
	// Get all API resources that support list and get verbs
	cmd := exec.Command(ocCmd, "api-resources", "--verbs=list,get", "-o", "name")
	if kubeconfig != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	}
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get API resources: %w", err)
	}

	// Parse output and sort
	resources := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	// Remove empty lines and sort
	var cleanResources []string
	for _, resource := range resources {
		resource = strings.TrimSpace(resource)
		if resource != "" {
			cleanResources = append(cleanResources, resource)
		}
	}

	return cleanResources, nil
}

func formatFilename(resourceName string) string {
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
	return fmt.Sprintf("%s.yaml", sanitizedName)
}

func formatHeader(resourceName string) string {
	var header strings.Builder
	
	header.WriteString("# Generated by k8s-resource-collector\n")
	header.WriteString(fmt.Sprintf("# Generated at: %s\n", time.Now().Format(time.RFC3339)))
	header.WriteString(fmt.Sprintf("# Resource: %s\n", resourceName))
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

func importAllResourcesFile(inputFile string) error {
	startTime := time.Now()
	
	if verbose {
		fmt.Printf("Starting import of all-resources.yaml file: %s\n", inputFile)
	}

	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}

	// Create output-import directory
	outputImportDir := "./output-import"
	if err := os.MkdirAll(outputImportDir, 0755); err != nil {
		return fmt.Errorf("failed to create output-import directory: %w", err)
	}

	// Clean directory if requested
	if clean {
		if err := cleanDirectory(outputImportDir); err != nil {
			return fmt.Errorf("failed to clean output-import directory: %w", err)
		}
	}

	// Parse the all-resources.yaml file
	clusterResources, err := parseAllResourcesFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to parse all-resources.yaml file: %w", err)
	}

	// Split ClusterResources into individual files
	processedCount := 0
	errorCount := 0

	for resourceName, resourceContent := range clusterResources {
		if verbose {
			fmt.Printf("Processing ClusterResource: %s\n", resourceName)
		}

		err := writeClusterResourceToFile(resourceName, resourceContent, outputImportDir)
		if err != nil {
			if verbose {
				fmt.Printf("  %s: ERROR - %v\n", resourceName, err)
			}
			errorCount++
		} else {
			processedCount++
		}
	}

	// Print summary
	duration := time.Since(startTime)
	fmt.Printf("\n=== Import Summary ===\n")
	fmt.Printf("Successfully processed: %d ClusterResources\n", processedCount)
	fmt.Printf("Errors encountered: %d ClusterResources\n", errorCount)
	fmt.Printf("Output directory: %s\n", outputImportDir)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("====================\n")

	return nil
}

func parseAllResourcesFile(inputFile string) (map[string]string, error) {
	file, err := os.Open(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	clusterResources := make(map[string]string)
	var currentResource strings.Builder
	var currentResourceName string
	var inResource bool

	// Regex to match resource headers like "--- # Resource: pods"
	resourceHeaderRegex := regexp.MustCompile(`^---\s*#\s*Resource:\s*(.+)$`)

	// Create scanner with increased buffer size to handle large lines
	scanner := bufio.NewScanner(file)
	
	// Increase buffer size to handle very long lines (up to 1MB per line)
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Check if this is a resource header
		if matches := resourceHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Save previous resource if exists
			if inResource && currentResourceName != "" {
				clusterResources[currentResourceName] = currentResource.String()
			}
			
			// Start new resource
			currentResourceName = matches[1]
			currentResource.Reset()
			inResource = true
			
			if verbose {
				fmt.Printf("Found ClusterResource: %s\n", currentResourceName)
			}
		} else if inResource {
			// Add line to current resource
			currentResource.WriteString(line)
			currentResource.WriteString("\n")
		}
	}

	// Save the last resource
	if inResource && currentResourceName != "" {
		clusterResources[currentResourceName] = currentResource.String()
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return clusterResources, nil
}

func writeClusterResourceToFile(resourceName string, resourceContent string, outputDir string) error {
	// Create filename
	filename := formatFilename(resourceName)
	filePath := filepath.Join(outputDir, filename)

	// Create header
	header := formatHeader(resourceName)
	
	// Write to file
	finalYaml := header + resourceContent
	err := os.WriteFile(filePath, []byte(finalYaml), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	if verbose {
		fmt.Printf("  %s: SUCCESS - Saved to %s\n", resourceName, filePath)
	}

	return nil
}
