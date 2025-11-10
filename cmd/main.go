package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/yaml"
)

var (
	kubeconfig  string
	kubeconfig1 string
	kubeconfig2 string
	mustGather  string
	mustGather1 string
	mustGather2 string
	outputDir   string
	outputFile  string
	verbose     bool
	singleFile  bool
	clean       bool
	compareMode bool
)

// DeprecationRule defines when a resource API is deprecated
type DeprecationRule struct {
	GroupVersion        string // e.g., "v1", "apps/v1"
	Resource            string // e.g., "endpoints", "componentstatuses"
	DeprecatedFrom      string // e.g., "1.19", "1.33", "4.14"
	ReplacementGV       string // e.g., "discovery.k8s.io/v1"
	ReplacementResource string // e.g., "endpointslices"
	IsOpenShift         bool   // true if this is an OpenShift-specific deprecation
}

// ClusterVersion holds version information
type ClusterVersion struct {
	Major          int
	Minor          int
	IsOpenShift    bool
	OpenShiftMajor int
	OpenShiftMinor int
}

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	flag.StringVar(&kubeconfig1, "kubeconfig1", "", "Path to first kubeconfig for cluster comparison")
	flag.StringVar(&kubeconfig2, "kubeconfig2", "", "Path to second kubeconfig for cluster comparison")
	flag.StringVar(&mustGather, "must-gather", "", "Path to must-gather directory for offline processing")
	flag.StringVar(&mustGather1, "must-gather1", "", "Path to first must-gather directory for comparison")
	flag.StringVar(&mustGather2, "must-gather2", "", "Path to second must-gather directory for comparison")
	flag.StringVar(&outputDir, "output", "./output", "Output directory for collected resources")
	flag.StringVar(&outputFile, "file", "", "Output file for single file mode")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&singleFile, "single-file", false, "Collect all resources to a single YAML file")
	flag.BoolVar(&clean, "clean", false, "Clean output directory before collection")
	flag.BoolVar(&compareMode, "compare", false, "Enable comparison mode (requires kubeconfig1 and kubeconfig2)")
	flag.Parse()

	if err := runCollector(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCollector() error {
	// Validate mutually exclusive flags
	if mustGather != "" && kubeconfig != "" {
		return fmt.Errorf("--must-gather and --kubeconfig are mutually exclusive; use one or the other")
	}

	if mustGather != "" && (kubeconfig1 != "" || kubeconfig2 != "") {
		return fmt.Errorf("--must-gather cannot be used with --kubeconfig1 or --kubeconfig2")
	}

	if (mustGather1 != "" || mustGather2 != "") && (kubeconfig != "" || kubeconfig1 != "" || kubeconfig2 != "") {
		return fmt.Errorf("--must-gather1/2 cannot be used with --kubeconfig flags; use one mode or the other")
	}

	if mustGather != "" && (mustGather1 != "" || mustGather2 != "") {
		return fmt.Errorf("--must-gather cannot be used with --must-gather1 or --must-gather2; use either single or comparison mode")
	}

	// Check if must-gather comparison mode is enabled
	if mustGather1 != "" && mustGather2 != "" {
		return runMustGatherComparisonMode()
	}

	if mustGather1 != "" || mustGather2 != "" {
		return fmt.Errorf("must-gather comparison mode requires both --must-gather1 and --must-gather2")
	}

	// Check if must-gather mode is enabled
	if mustGather != "" {
		return runMustGatherMode()
	}

	// Check if comparison mode is enabled
	if compareMode || (kubeconfig1 != "" && kubeconfig2 != "") {
		return runComparisonMode()
	}

	// Determine output mode
	if outputFile != "" {
		singleFile = true
	} else if singleFile {
		outputFile = "./output/all-resources.yaml"
	}

	// Use kubeconfig1 if provided (fallback when kubeconfig is not used), otherwise fall back to kubeconfig
	configPath := kubeconfig
	if configPath == "" && kubeconfig1 != "" {
		configPath = kubeconfig1
	}

	// Parse kubeconfig
	config, err := parseKubeConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Create clients
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

// detectClusterVersion detects the Kubernetes and OpenShift versions
func detectClusterVersion(discovery discovery.DiscoveryInterface) (*ClusterVersion, error) {
	serverVersion, err := discovery.ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}

	cv := &ClusterVersion{}

	// Parse Kubernetes version
	major, err := strconv.Atoi(strings.TrimSuffix(serverVersion.Major, "+"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse major version: %w", err)
	}
	cv.Major = major

	minor, err := strconv.Atoi(strings.TrimSuffix(serverVersion.Minor, "+"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse minor version: %w", err)
	}
	cv.Minor = minor

	// Check if this is OpenShift by looking for OpenShift-specific API groups
	apiGroups, err := discovery.ServerGroups()
	if err == nil {
		for _, group := range apiGroups.Groups {
			if strings.Contains(group.Name, "openshift.io") {
				cv.IsOpenShift = true
				break
			}
		}
	}

	// Try to detect OpenShift version from the platform
	if cv.IsOpenShift {
		// OpenShift version is typically v1.X.Y where X maps to OpenShift 4.X
		// For example, Kubernetes 1.27 = OpenShift 4.14
		cv.OpenShiftMajor = 4
		// Rough mapping (this can vary, but gives us a good approximation)
		if cv.Minor >= 27 {
			cv.OpenShiftMinor = 14 + (cv.Minor - 27)
		}
	}

	if verbose {
		fmt.Printf("Detected Kubernetes version: %d.%d\n", cv.Major, cv.Minor)
		if cv.IsOpenShift {
			fmt.Printf("Detected OpenShift cluster (estimated version: %d.%d)\n",
				cv.OpenShiftMajor, cv.OpenShiftMinor)
		}
	}

	return cv, nil
}

// getDeprecationRules returns a list of known deprecation rules
func getDeprecationRules() []DeprecationRule {
	return []DeprecationRule{
		{
			GroupVersion:        "v1",
			Resource:            "componentstatuses",
			DeprecatedFrom:      "1.19",
			ReplacementGV:       "", // No replacement - component status is deprecated without replacement
			ReplacementResource: "",
			IsOpenShift:         false,
		},
		{
			GroupVersion:        "v1",
			Resource:            "endpoints",
			DeprecatedFrom:      "1.33",
			ReplacementGV:       "discovery.k8s.io/v1",
			ReplacementResource: "endpointslices",
			IsOpenShift:         false,
		},
		{
			GroupVersion:        "apps.openshift.io/v1",
			Resource:            "deploymentconfigs",
			DeprecatedFrom:      "4.14",
			ReplacementGV:       "", // DeploymentConfigs should be migrated to standard Deployments
			ReplacementResource: "",
			IsOpenShift:         true,
		},
	}
}

// isDeprecated checks if a resource is deprecated based on cluster version
// Returns: (isDeprecated, replacementGV, replacementResource, message)
func isDeprecated(resource metav1.APIResource, groupVersion string, clusterVersion *ClusterVersion) (bool, string, string, string) {
	rules := getDeprecationRules()

	for _, rule := range rules {
		// Check if this rule applies to this resource
		if rule.GroupVersion != groupVersion || rule.Resource != resource.Name {
			continue
		}

		// Check if we should apply OpenShift-specific rules
		if rule.IsOpenShift && !clusterVersion.IsOpenShift {
			continue
		}

		// Parse the deprecation version
		parts := strings.Split(rule.DeprecatedFrom, ".")
		if len(parts) < 2 {
			continue
		}

		depMajor, err1 := strconv.Atoi(parts[0])
		depMinor, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			continue
		}

		// Compare versions
		var isDeprecated bool
		if rule.IsOpenShift {
			// Compare against OpenShift version
			if clusterVersion.OpenShiftMajor > depMajor ||
				(clusterVersion.OpenShiftMajor == depMajor && clusterVersion.OpenShiftMinor >= depMinor) {
				isDeprecated = true
			}
		} else {
			// Compare against Kubernetes version
			if clusterVersion.Major > depMajor ||
				(clusterVersion.Major == depMajor && clusterVersion.Minor >= depMinor) {
				isDeprecated = true
			}
		}

		if isDeprecated {
			var msg string
			if rule.ReplacementGV != "" && rule.ReplacementResource != "" {
				msg = fmt.Sprintf("Using %s/%s instead of deprecated %s/%s",
					rule.ReplacementGV, rule.ReplacementResource, groupVersion, resource.Name)
			} else {
				msg = fmt.Sprintf("Skipping deprecated %s/%s (no replacement available)",
					groupVersion, resource.Name)
			}
			return true, rule.ReplacementGV, rule.ReplacementResource, msg
		}
	}

	return false, "", "", ""
}

// shouldSkipResource determines if a resource should be skipped
// Returns: (shouldSkip, message)
func shouldSkipResource(resource metav1.APIResource, groupVersion string, clusterVersion *ClusterVersion) (bool, string) {
	deprecated, _, _, msg := isDeprecated(resource, groupVersion, clusterVersion)
	if deprecated {
		return true, msg
	}
	return false, ""
}

func collectResources(discovery discovery.DiscoveryInterface, dynamic dynamic.Interface, outputDir string) error {
	startTime := time.Now()

	if verbose {
		fmt.Printf("Starting resource collection to directory: %s\n", outputDir)
	}

	// Detect cluster version
	clusterVersion, err := detectClusterVersion(discovery)
	if err != nil {
		fmt.Printf("Warning: failed to detect cluster version: %v\n", err)
		fmt.Println("Continuing without deprecation checks...")
		clusterVersion = nil
	}

	// Get all API resources
	resources, err := discovery.ServerPreferredResources()
	if err != nil {
		return fmt.Errorf("failed to discover API resources: %w", err)
	}

	collectedCount := 0
	errorCount := 0
	skippedCount := 0

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

			// Check if resource is deprecated and should be skipped
			if clusterVersion != nil {
				if skip, msg := shouldSkipResource(resource, resourceList.GroupVersion, clusterVersion); skip {
					if verbose {
						fmt.Printf("%s\n", msg)
					}
					skippedCount++
					continue
				}
			}

			if verbose {
				fmt.Printf("Collecting resource: %s (%s)\n", resource.Name, resourceList.GroupVersion)
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
	if skippedCount > 0 {
		fmt.Printf("Skipped deprecated: %d resources\n", skippedCount)
	}
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
		return fmt.Errorf("failed to marshal %s to YAML: %w", resource.Name, err)
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

	// Detect cluster version
	clusterVersion, err := detectClusterVersion(discovery)
	if err != nil {
		fmt.Printf("Warning: failed to detect cluster version: %v\n", err)
		fmt.Println("Continuing without deprecation checks...")
		clusterVersion = nil
	}

	// Get all API resources
	resources, err := discovery.ServerPreferredResources()
	if err != nil {
		return fmt.Errorf("failed to discover API resources: %w", err)
	}

	var allResourcesYaml strings.Builder
	collectedCount := 0
	errorCount := 0
	skippedCount := 0

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

			// Check if resource is deprecated and should be skipped
			if clusterVersion != nil {
				if skip, msg := shouldSkipResource(resource, resourceList.GroupVersion, clusterVersion); skip {
					if verbose {
						fmt.Printf("%s\n", msg)
					}
					skippedCount++
					continue
				}
			}

			if verbose {
				fmt.Printf("Collecting resource: %s (%s)\n", resource.Name, resourceList.GroupVersion)
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
	if skippedCount > 0 {
		fmt.Printf("Skipped deprecated: %d resources\n", skippedCount)
	}
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
		return fmt.Errorf("failed to marshal %s to YAML: %w", resource.Name, err)
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

// runComparisonMode collects resources from two clusters and generates a diff
func runComparisonMode() error {
	if kubeconfig1 == "" || kubeconfig2 == "" {
		return fmt.Errorf("comparison mode requires both --kubeconfig1 and --kubeconfig2 to be specified")
	}

	fmt.Println("=== Multi-Cluster Comparison Mode ===")
	fmt.Printf("Cluster 1: %s\n", kubeconfig1)
	fmt.Printf("Cluster 2: %s\n", kubeconfig2)
	fmt.Println()

	// Get cluster names
	clusterName1, err := getClusterName(kubeconfig1)
	if err != nil {
		return fmt.Errorf("failed to get cluster name from kubeconfig1: %w", err)
	}

	clusterName2, err := getClusterName(kubeconfig2)
	if err != nil {
		return fmt.Errorf("failed to get cluster name from kubeconfig2: %w", err)
	}

	// Create comparison output directory
	compareDir := filepath.Join(outputDir, "comparison")
	if err := os.MkdirAll(compareDir, 0755); err != nil {
		return fmt.Errorf("failed to create comparison directory: %w", err)
	}

	// Collect from cluster 1
	fmt.Printf("\n[1/3] Collecting from cluster 1: %s\n", clusterName1)
	outputFile1 := filepath.Join(compareDir, fmt.Sprintf("%s-resources.yaml", sanitizeClusterName(clusterName1)))
	if err := collectFromCluster(kubeconfig1, outputFile1); err != nil {
		return fmt.Errorf("failed to collect from cluster 1: %w", err)
	}
	fmt.Printf("✓ Saved to: %s\n", outputFile1)

	// Collect from cluster 2
	fmt.Printf("\n[2/3] Collecting from cluster 2: %s\n", clusterName2)
	outputFile2 := filepath.Join(compareDir, fmt.Sprintf("%s-resources.yaml", sanitizeClusterName(clusterName2)))
	if err := collectFromCluster(kubeconfig2, outputFile2); err != nil {
		return fmt.Errorf("failed to collect from cluster 2: %w", err)
	}
	fmt.Printf("✓ Saved to: %s\n", outputFile2)

	// Generate diff
	fmt.Printf("\n[3/3] Generating difference report...\n")
	diffFile := filepath.Join(compareDir, fmt.Sprintf("diff-%s-vs-%s.txt",
		sanitizeClusterName(clusterName1),
		sanitizeClusterName(clusterName2)))

	if err := generateDiff(outputFile1, outputFile2, diffFile, clusterName1, clusterName2); err != nil {
		return fmt.Errorf("failed to generate diff: %w", err)
	}
	fmt.Printf("✓ Diff saved to: %s\n", diffFile)

	fmt.Println("\n=== Comparison Complete ===")
	fmt.Printf("Cluster 1 (%s): %s\n", clusterName1, outputFile1)
	fmt.Printf("Cluster 2 (%s): %s\n", clusterName2, outputFile2)
	fmt.Printf("Difference:     %s\n", diffFile)

	return nil
}

// getClusterName extracts the cluster name from kubeconfig
func getClusterName(kubeconfigPath string) (string, error) {
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", err
	}

	// Get current context
	currentContext := config.CurrentContext
	if currentContext == "" {
		return "", fmt.Errorf("no current context set in kubeconfig")
	}

	// Get context details
	context, exists := config.Contexts[currentContext]
	if !exists {
		return "", fmt.Errorf("context %s not found in kubeconfig", currentContext)
	}

	// Return cluster name
	if context.Cluster != "" {
		return context.Cluster, nil
	}

	return currentContext, nil
}

// sanitizeClusterName sanitizes cluster name for use in filenames
func sanitizeClusterName(name string) string {
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
		".", "-",
	)
	return replacer.Replace(name)
}

// collectFromCluster collects resources from a specific cluster
func collectFromCluster(kubeconfigPath string, outputFile string) error {
	config, err := parseKubeConfig(kubeconfigPath)
	if err != nil {
		return err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	return collectAllResourcesToSingleFile(discoveryClient, dynamicClient, outputFile)
}

// generateDiff generates a diff between two resource files
func generateDiff(file1, file2, outputFile, cluster1Name, cluster2Name string) error {
	// Read both files
	content1, err := os.ReadFile(file1)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", file1, err)
	}

	content2, err := os.ReadFile(file2)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", file2, err)
	}

	// Parse YAML resources from both files
	resources1 := parseResources(string(content1))
	resources2 := parseResources(string(content2))

	// Generate diff report
	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("=== Cluster Comparison Report ===\n"))
	diff.WriteString(fmt.Sprintf("Generated at: %s\n", time.Now().Format(time.RFC3339)))
	diff.WriteString(fmt.Sprintf("Cluster 1: %s (%d resources)\n", cluster1Name, len(resources1)))
	diff.WriteString(fmt.Sprintf("Cluster 2: %s (%d resources)\n\n", cluster2Name, len(resources2)))

	// Find resources only in cluster 1
	onlyInCluster1 := findUniqueResources(resources1, resources2)
	if len(onlyInCluster1) > 0 {
		diff.WriteString(fmt.Sprintf("\n=== Resources only in %s ===\n", cluster1Name))
		for _, resource := range onlyInCluster1 {
			diff.WriteString(fmt.Sprintf("- %s\n", resource))
		}
	}

	// Find resources only in cluster 2
	onlyInCluster2 := findUniqueResources(resources2, resources1)
	if len(onlyInCluster2) > 0 {
		diff.WriteString(fmt.Sprintf("\n=== Resources only in %s ===\n", cluster2Name))
		for _, resource := range onlyInCluster2 {
			diff.WriteString(fmt.Sprintf("- %s\n", resource))
		}
	}

	// Find common resources
	commonResources := findCommonResources(resources1, resources2)
	if len(commonResources) > 0 {
		diff.WriteString(fmt.Sprintf("\n=== Common resources in both clusters ===\n"))
		diff.WriteString(fmt.Sprintf("Total: %d resources\n", len(commonResources)))
	}

	// Summary
	diff.WriteString(fmt.Sprintf("\n=== Summary ===\n"))
	diff.WriteString(fmt.Sprintf("Total resources in %s: %d\n", cluster1Name, len(resources1)))
	diff.WriteString(fmt.Sprintf("Total resources in %s: %d\n", cluster2Name, len(resources2)))
	diff.WriteString(fmt.Sprintf("Only in %s: %d\n", cluster1Name, len(onlyInCluster1)))
	diff.WriteString(fmt.Sprintf("Only in %s: %d\n", cluster2Name, len(onlyInCluster2)))
	diff.WriteString(fmt.Sprintf("Common to both: %d\n", len(commonResources)))

	// Write diff to file
	return os.WriteFile(outputFile, []byte(diff.String()), 0644)
}

// parseResources extracts resource identifiers from YAML content
func parseResources(content string) []string {
	var resources []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		// Look for resource markers (e.g., "--- # Resource: pods")
		if strings.HasPrefix(strings.TrimSpace(line), "--- # Resource:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				resource := strings.TrimSpace(parts[1])
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// findUniqueResources finds resources in set1 that are not in set2
func findUniqueResources(set1, set2 []string) []string {
	set2Map := make(map[string]bool)
	for _, item := range set2 {
		set2Map[item] = true
	}

	var unique []string
	for _, item := range set1 {
		if !set2Map[item] {
			unique = append(unique, item)
		}
	}

	return unique
}

// findCommonResources finds resources present in both sets
func findCommonResources(set1, set2 []string) []string {
	set2Map := make(map[string]bool)
	for _, item := range set2 {
		set2Map[item] = true
	}

	var common []string
	seen := make(map[string]bool)
	for _, item := range set1 {
		if set2Map[item] && !seen[item] {
			common = append(common, item)
			seen[item] = true
		}
	}

	return common
}

// validateMustGatherPath validates the must-gather directory path
func validateMustGatherPath(path string) error {
	// Check if path is empty
	if path == "" {
		return fmt.Errorf("must-gather path cannot be empty")
	}

	// Check if path exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("must-gather directory not found: %s\nPlease verify the path exists and is accessible", path)
	}
	if err != nil {
		return fmt.Errorf("failed to access must-gather directory: %s\nError: %v", path, err)
	}

	// Check if path is a directory
	if !info.IsDir() {
		return fmt.Errorf("must-gather path is not a directory: %s\nPlease provide a path to a directory, not a file", path)
	}

	// Check if directory is readable
	// Try to read the directory to verify permissions
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("cannot read must-gather directory: %s\nError: %v\nPlease check directory permissions", path, err)
	}

	// Check if directory is empty
	if len(entries) == 0 {
		return fmt.Errorf("must-gather directory is empty: %s\nPlease provide a valid must-gather directory with YAML files", path)
	}

	// Optional: Check if it looks like a must-gather directory (contains YAML files)
	hasYamlFiles := false
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		if !info.IsDir() && (strings.HasSuffix(p, ".yaml") || strings.HasSuffix(p, ".yml")) {
			hasYamlFiles = true
			return filepath.SkipAll // Found at least one, we can stop
		}
		return nil
	})

	if err == nil && !hasYamlFiles {
		fmt.Printf("Warning: No YAML files found in must-gather directory: %s\n", path)
		fmt.Println("The directory will be processed, but no resources may be extracted.")
	}

	return nil
}

// runMustGatherMode processes a must-gather directory and outputs resources
func runMustGatherMode() error {
	startTime := time.Now()

	// Validate must-gather path
	if err := validateMustGatherPath(mustGather); err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Processing must-gather directory: %s\n", mustGather)
	}

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

	// Process must-gather directory
	collectedCount, errorCount, err := processMustGatherDirectory(mustGather, outputDir)
	if err != nil {
		return err
	}

	// Print summary
	duration := time.Since(startTime)
	fmt.Printf("\n=== Must-Gather Processing Summary ===\n")
	fmt.Printf("Successfully processed: %d resource types\n", collectedCount)
	fmt.Printf("Errors encountered: %d resource types\n", errorCount)
	fmt.Printf("Output directory: %s\n", outputDir)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("====================================\n")

	return nil
}

// runMustGatherComparisonMode processes two must-gather directories and generates a diff
func runMustGatherComparisonMode() error {
	fmt.Println("=== Must-Gather Comparison Mode ===")
	fmt.Printf("Must-Gather 1: %s\n", mustGather1)
	fmt.Printf("Must-Gather 2: %s\n", mustGather2)
	fmt.Println()

	// Validate both must-gather paths
	if err := validateMustGatherPath(mustGather1); err != nil {
		return fmt.Errorf("invalid must-gather1: %w", err)
	}
	if err := validateMustGatherPath(mustGather2); err != nil {
		return fmt.Errorf("invalid must-gather2: %w", err)
	}

	// Get must-gather names for output files
	mgName1 := getMustGatherName(mustGather1)
	mgName2 := getMustGatherName(mustGather2)

	// Create comparison output directory
	compareDir := filepath.Join(outputDir, "comparison")
	if err := os.MkdirAll(compareDir, 0755); err != nil {
		return fmt.Errorf("failed to create comparison directory: %w", err)
	}

	// Process from must-gather 1
	fmt.Printf("\n[1/3] Processing must-gather 1: %s\n", mgName1)
	outputFile1 := filepath.Join(compareDir, fmt.Sprintf("%s-resources.yaml", sanitizeClusterName(mgName1)))
	if err := processMustGatherToSingleFile(mustGather1, outputFile1); err != nil {
		return fmt.Errorf("failed to process must-gather 1: %w", err)
	}
	fmt.Printf("✓ Saved to: %s\n", outputFile1)

	// Process from must-gather 2
	fmt.Printf("\n[2/3] Processing must-gather 2: %s\n", mgName2)
	outputFile2 := filepath.Join(compareDir, fmt.Sprintf("%s-resources.yaml", sanitizeClusterName(mgName2)))
	if err := processMustGatherToSingleFile(mustGather2, outputFile2); err != nil {
		return fmt.Errorf("failed to process must-gather 2: %w", err)
	}
	fmt.Printf("✓ Saved to: %s\n", outputFile2)

	// Generate diff
	fmt.Printf("\n[3/3] Generating difference report...\n")
	diffFile := filepath.Join(compareDir, fmt.Sprintf("diff-%s-vs-%s.txt",
		sanitizeClusterName(mgName1),
		sanitizeClusterName(mgName2)))

	if err := generateDiff(outputFile1, outputFile2, diffFile, mgName1, mgName2); err != nil {
		return fmt.Errorf("failed to generate diff: %w", err)
	}
	fmt.Printf("✓ Diff saved to: %s\n", diffFile)

	fmt.Println("\n=== Comparison Complete ===")
	fmt.Printf("Must-Gather 1 (%s): %s\n", mgName1, outputFile1)
	fmt.Printf("Must-Gather 2 (%s): %s\n", mgName2, outputFile2)
	fmt.Printf("Difference:         %s\n", diffFile)

	return nil
}

// getMustGatherName extracts a meaningful name from must-gather path
func getMustGatherName(path string) string {
	// Get the base directory name
	name := filepath.Base(path)

	// If it looks like a must-gather directory (contains timestamp/id), use it
	if strings.Contains(name, "must-gather") {
		return name
	}

	// Otherwise, try to create a meaningful name
	absPath, err := filepath.Abs(path)
	if err == nil {
		name = filepath.Base(absPath)
	}

	if name == "." || name == "/" {
		name = "must-gather"
	}

	return name
}

// processMustGatherToSingleFile processes a must-gather directory into a single file
func processMustGatherToSingleFile(mustGatherPath, outputFile string) error {
	resourceMap := make(map[string][]interface{})

	// Walk through the must-gather directory
	err := filepath.Walk(mustGatherPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Process only YAML files
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		// Read and parse the YAML file
		processMustGatherFile(path, resourceMap)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk must-gather directory: %w", err)
	}

	// Build single file output
	var allResourcesYaml strings.Builder

	// Sort keys for consistent output
	var keys []string
	for key := range resourceMap {
		keys = append(keys, key)
	}

	for _, key := range keys {
		items := resourceMap[key]
		if len(items) == 0 {
			continue
		}

		// Add resource comment
		allResourcesYaml.WriteString(fmt.Sprintf("--- # Resource: %s\n", key))

		// Create a list structure
		list := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "List",
			"items":      items,
		}

		// Marshal to YAML
		yamlData, err := yaml.Marshal(list)
		if err != nil {
			continue
		}

		allResourcesYaml.WriteString(string(yamlData))
		allResourcesYaml.WriteString("\n")
	}

	// Write to file
	return os.WriteFile(outputFile, []byte(allResourcesYaml.String()), 0644)
}

// processMustGatherDirectory walks through the must-gather directory and processes YAML files
func processMustGatherDirectory(mustGatherPath, outputPath string) (int, int, error) {
	resourceMap := make(map[string][]interface{}) // key: groupVersion-resource, value: list of items
	collectedCount := 0
	errorCount := 0

	// Walk through the must-gather directory
	err := filepath.Walk(mustGatherPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if verbose {
				fmt.Printf("Warning: failed to access %s: %v\n", path, err)
			}
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Process only YAML files
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		if verbose {
			fmt.Printf("Processing file: %s\n", path)
		}

		// Read and parse the YAML file
		if err := processMustGatherFile(path, resourceMap); err != nil {
			if verbose {
				fmt.Printf("  Error processing %s: %v\n", path, err)
			}
			errorCount++
		}

		return nil
	})

	if err != nil {
		return 0, 0, fmt.Errorf("failed to walk must-gather directory: %w", err)
	}

	// Write organized resources to output directory
	for key, items := range resourceMap {
		if len(items) == 0 {
			continue
		}

		// Create output file
		filename := fmt.Sprintf("%s.yaml", key)
		filePath := filepath.Join(outputPath, filename)

		// Create a list structure
		list := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "List",
			"items":      items,
		}

		// Marshal to YAML
		yamlData, err := yaml.Marshal(list)
		if err != nil {
			if verbose {
				fmt.Printf("Error marshaling %s: %v\n", key, err)
			}
			errorCount++
			continue
		}

		// Create header
		parts := strings.SplitN(key, "-", 2)
		groupVersion := parts[0]
		resourceName := ""
		if len(parts) > 1 {
			resourceName = parts[1]
		}

		header := formatHeader(resourceName, groupVersion)
		finalYaml := header + string(yamlData)

		// Write to file
		if err := os.WriteFile(filePath, []byte(finalYaml), 0644); err != nil {
			if verbose {
				fmt.Printf("Error writing %s: %v\n", filePath, err)
			}
			errorCount++
			continue
		}

		if verbose {
			fmt.Printf("  %s: SUCCESS - Saved %d items to %s\n", key, len(items), filePath)
		}
		collectedCount++
	}

	return collectedCount, errorCount, nil
}

// processMustGatherFile reads a YAML file and extracts resources
func processMustGatherFile(filePath string, resourceMap map[string][]interface{}) error {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Split by document separator
	docs := strings.Split(string(data), "\n---")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" || strings.HasPrefix(doc, "#") {
			continue
		}

		// Parse YAML document
		var resource map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &resource); err != nil {
			// Skip invalid YAML
			continue
		}

		// Extract apiVersion and kind
		apiVersion, ok := resource["apiVersion"].(string)
		if !ok || apiVersion == "" {
			continue
		}

		kind, ok := resource["kind"].(string)
		if !ok || kind == "" {
			continue
		}

		// Skip List kinds - we'll process individual items
		if kind == "List" {
			items, ok := resource["items"].([]interface{})
			if ok {
				for _, item := range items {
					itemMap, ok := item.(map[string]interface{})
					if !ok {
						continue
					}

					// Extract item details
					itemApiVersion, _ := itemMap["apiVersion"].(string)
					itemKind, _ := itemMap["kind"].(string)
					if itemApiVersion != "" && itemKind != "" {
						key := makeResourceKey(itemApiVersion, itemKind)
						resourceMap[key] = append(resourceMap[key], itemMap)
					}
				}
			}
			continue
		}

		// Create a key for this resource type
		key := makeResourceKey(apiVersion, kind)

		// Add to resource map
		resourceMap[key] = append(resourceMap[key], resource)
	}

	return nil
}

// makeResourceKey creates a consistent key for resource types
func makeResourceKey(apiVersion, kind string) string {
	// Convert kind to lowercase plural (simple approach)
	resource := strings.ToLower(kind)
	if !strings.HasSuffix(resource, "s") {
		resource += "s"
	}

	// Format: groupVersion-resource
	return fmt.Sprintf("%s-%s", strings.ReplaceAll(apiVersion, "/", "-"), resource)
}
