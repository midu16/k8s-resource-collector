package main

import (
	"fmt"
	"os"

	"github.com/midu/k8s-resource-collector/pkg"
	"github.com/spf13/cobra"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
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
	var rootCmd = &cobra.Command{
		Use:   "k8s-resource-collector",
		Short: "Collect all Kubernetes API resources and save them as YAML files",
		Long: `A tool that discovers all API resources in a Kubernetes/OpenShift cluster
and saves each resource type as a separate YAML file or all resources to a single file.
Can be used as an oc plugin.

This tool replicates the functionality of:
for r in $(oc api-resources --verbs=list,get -o name | sort -u); do
  echo "--- # Resource: $r" >> all-resources.yaml
  oc get "$r" --all-namespaces -o yaml >> all-resources.yaml 2>/dev/null
done`,
		RunE: runCollector,
	}

	rootCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "./output", "Output directory for collected resources")
	rootCmd.Flags().StringVarP(&outputFile, "file", "f", "", "Output file for single file mode (overrides output directory)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolVar(&singleFile, "single-file", false, "Collect all resources to a single YAML file")
	rootCmd.Flags().BoolVar(&clean, "clean", false, "Clean output directory before collection")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCollector(cmd *cobra.Command, args []string) error {
	// Initialize tools
	vibeTools := pkg.NewVibeTools(verbose)
	formatter := pkg.NewFormatter()

	// Determine output mode
	if outputFile != "" {
		singleFile = true
	} else if singleFile {
		outputFile = "./output/all-resources.yaml"
	}

	// Parse kubeconfig
	config, err := pkg.ParseKubeConfig(kubeconfig)
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

	// Create parser
	parser := pkg.NewParser(client, discoveryClient, dynamicClient, formatter, verbose)

	if singleFile {
		// Single file mode
		if outputFile == "" {
			outputFile = "./output/all-resources.yaml"
		}

		// Ensure output directory exists
		outputDir := fmt.Sprintf("%s", outputFile)
		if err := vibeTools.EnsureDirectory(outputDir); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Clean file if requested
		if clean {
			if err := os.Remove(outputFile); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to clean output file: %w", err)
			}
		}

		return parser.CollectAllResourcesToSingleFile(outputFile)
	} else {
		// Directory mode
		// Ensure output directory exists
		if err := vibeTools.EnsureDirectory(outputDir); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Clean directory if requested
		if clean {
			if err := vibeTools.CleanDirectory(outputDir); err != nil {
				return fmt.Errorf("failed to clean output directory: %w", err)
			}
		}

		return parser.CollectResources(outputDir)
	}
}
