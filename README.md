# k8s-resource-collector

A Go-based tool that collects all Kubernetes API resources from an OpenShift/Kubernetes cluster and saves them as YAML files. This tool replicates the functionality of the shell script:

```bash
for r in $(oc api-resources --verbs=list,get -o name | sort -u); do
  echo "--- # Resource: $r" >> all-resources.yaml
  oc get "$r" --all-namespaces -o yaml >> all-resources.yaml 2>/dev/null
done
```

## Features

- **Four Collection Modes**:
  - Directory mode: Creates individual YAML files for each resource type
  - Single file mode: Creates one file with all resources (like the original script)
  - Must-gather mode: Process OpenShift must-gather directories offline
  - Import mode: Splits existing all-resources.yaml files into individual ClusterResource files
- **Intelligent Deprecation Handling**: Automatically detects Kubernetes/OpenShift versions and uses non-deprecated replacement APIs to prevent warnings
- **Multi-Cluster Comparison**: Compare resources between two Kubernetes clusters and generate diff reports
- **Flexible Configuration**: Supports `--kubeconfig` flag, `KUBECONFIG` environment variable, or `--must-gather` for offline processing
- **Verbose Logging**: Optional detailed output during collection
- **Clean Mode**: Option to clean output directories before collection
- **Cross-Platform**: Works on Linux, macOS, and Windows
- **Container Support**: Includes Dockerfile for containerized deployment
- **Native Kubernetes Client**: Uses official k8s.io/client-go libraries

## Project Structure

```
k8s-resource-collector/
├── bin/                              # Binary output directory
│   └── k8s-resource-collector        # Compiled binary
├── cmd/                              # Application source code
│   └── main.go                       # Main application (native Kubernetes client)
├── tests/                            # Test suite
│   ├── functional_test.go            # Go-based functional tests
│   ├── test_runner.sh                # Comprehensive test runner
│   ├── simple_test_runner.sh         # Simple test runner
│   ├── test_config.yaml              # Test configuration
│   └── README.md                     # Testing documentation
├── collector_test.go                 # Unit tests
├── CONTRIBUTING.md                   # Contribution guidelines
├── Dockerfile                        # Container build file
├── go.mod                            # Go module dependencies
├── go.sum                            # Dependency checksums
├── LICENSE                           # MIT License
├── Makefile                          # Build system
├── README.md                         # This file
└── run-container-example.sh          # Container usage example
```

### Output Directories (created at runtime)
```
output/                               # Default collection output
├── comparison/                       # Multi-cluster comparison results
│   ├── {cluster1}-resources.yaml
│   ├── {cluster2}-resources.yaml
│   └── diff-{cluster1}-vs-{cluster2}.txt
└── *.yaml                            # Individual resource files
```

## Quick Start

### Build and Run

```bash
# Build the binary
make build

# Show help
./bin/k8s-resource-collector --help

# Collect resources to individual files
./bin/k8s-resource-collector --verbose --output ./output

# Collect all resources to a single file
./bin/k8s-resource-collector --single-file --file ./output/all-resources.yaml

# Use with custom kubeconfig
./bin/k8s-resource-collector --kubeconfig /path/to/kubeconfig --verbose

# Compare resources between two clusters
./bin/k8s-resource-collector --kubeconfig1 ~/.kube/config-prod --kubeconfig2 ~/.kube/config-staging
```

## Collection Modes

### 1. Directory Mode (Default)
Creates individual YAML files for each resource type:

```bash
./bin/k8s-resource-collector --verbose --output ./output
```

Output:
```
output/
├── v1-pods.yaml
├── v1-services.yaml
├── v1-configmaps.yaml
├── apps-v1-deployments.yaml
└── ...
```

### 2. Single File Mode
Creates one file with all resources (replicates original script):

```bash
./bin/k8s-resource-collector --single-file --file ./output/all-resources.yaml
```

Output:
```yaml
--- # Resource: pods
apiVersion: v1
kind: List
items:
- apiVersion: v1
  kind: Pod
  metadata:
    name: example-pod
    namespace: default
  ...
---
--- # Resource: services
apiVersion: v1
kind: List
items:
- apiVersion: v1
  kind: Service
  metadata:
    name: example-service
    namespace: default
  ...
---
```

### 3. Multi-Cluster Comparison Mode
Compare resources between two Kubernetes clusters:

```bash
# Basic comparison
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/config-cluster1 \
  --kubeconfig2 ~/.kube/config-cluster2

# With verbose output
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/prod-config \
  --kubeconfig2 ~/.kube/staging-config \
  --verbose

# Custom output directory
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/config1 \
  --kubeconfig2 ~/.kube/config2 \
  --output ./my-comparison
```

Output:
```
output/comparison/
├── prod-cluster-resources.yaml                    # All resources from cluster 1
├── staging-cluster-resources.yaml                 # All resources from cluster 2
└── diff-prod-cluster-vs-staging-cluster.txt       # Difference report
```

The diff report includes:
- Resources only in cluster 1
- Resources only in cluster 2
- Common resources in both clusters
- Statistical summary

**📘 For detailed documentation, see [CLUSTER_COMPARISON.md](CLUSTER_COMPARISON.md)**

## Command Line Options

| Flag | Description | Default | Notes |
|------|-------------|---------|-------|
| `--kubeconfig` | Path to kubeconfig file | `$KUBECONFIG` or `~/.kube/config` | Mutually exclusive with `--must-gather*` |
| `--kubeconfig1` | First kubeconfig for comparison | - | Fallback if `--kubeconfig` not specified |
| `--kubeconfig2` | Second kubeconfig for comparison | - | For comparison mode |
| `--must-gather` | Path to must-gather directory | - | Mutually exclusive with kubeconfig flags |
| `--must-gather1` | First must-gather for comparison | - | Requires `--must-gather2` |
| `--must-gather2` | Second must-gather for comparison | - | Requires `--must-gather1` |
| `--output` | Output directory | `./output` | |
| `--file` | Output file for single file mode | - | |
| `--verbose` | Enable verbose output | `false` | |
| `--single-file` | Collect to a single YAML file | `false` | |
| `--clean` | Clean output directory before collection | `false` | |
| `--compare` | Enable comparison mode | `false` | |

## Example Workflows

### Scenario 1: Regular Collection
```bash
# Collect all resources from current cluster
./bin/k8s-resource-collector --verbose --output ./my-output
```

### Scenario 2: Must-Gather Processing
```bash
# Process an OpenShift must-gather directory (offline)
./bin/k8s-resource-collector \
  --must-gather ./must-gather.local.5498831487182099551/ \
  --output ./output/ \
  --verbose
```

### Scenario 3: Production Backup
```bash
# Create a single-file backup of production cluster
./bin/k8s-resource-collector \
  --kubeconfig ~/.kube/prod-config \
  --single-file \
  --file ./backups/prod-$(date +%Y%m%d).yaml \
  --verbose
```

### Scenario 3: Environment Comparison
```bash
# Compare prod vs staging
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/prod-config \
  --kubeconfig2 ~/.kube/staging-config \
  --output ./comparison \
  --verbose
```

### Scenario 4: Multi-Region Validation
```bash
# Verify consistency across regions
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/us-east-config \
  --kubeconfig2 ~/.kube/eu-west-config
```

### Scenario 5: Must-Gather Comparison
```bash
# Compare two must-gather snapshots (e.g., before and after change)
./bin/k8s-resource-collector \
  --must-gather1 ./must-gather-before/ \
  --must-gather2 ./must-gather-after/ \
  --output ./comparison/
```

## Verbose Output Example

```bash
./bin/k8s-resource-collector --verbose --output ./output
```

Output:
```
Starting resource collection to directory: ./output
Detected Kubernetes version: 1.30
Detected OpenShift cluster (estimated version: 4.17)
Using discovery.k8s.io/v1/endpointslices instead of deprecated v1/endpoints
Skipping deprecated v1/componentstatuses (no replacement available)
Collecting resource: endpointslices (discovery.k8s.io/v1)
  endpointslices: SUCCESS - Saved to ./output/discovery.k8s.io-v1-endpointslices.yaml
Collecting resource: pods (v1)
  pods: SUCCESS - Saved to ./output/v1-pods.yaml
Collecting resource: services (v1)
  services: SUCCESS - Saved to ./output/v1-services.yaml

=== Collection Summary ===
Successfully collected: 45 resources
Skipped deprecated: 2 resources
Errors encountered: 0 resources
Output directory: ./output
Duration: 2m30s
========================
```

## Development

### Build System

The project uses a comprehensive Makefile with multiple targets:

```bash
# Build binary
make build

# Build for multiple platforms
make build-all

# Run unit tests
make test-unit

# Run all tests
make test-all

# Format code
make fmt

# Run linters
make lint

# Clean build artifacts
make clean

# Show all available targets
make help
```

### Testing

```bash
# Run unit tests
go test -v ./...

# Run unit tests with coverage
make test-unit

# Run functional tests
make test-go

# Run all tests
make test-all
```

## Requirements

- **Go**: Version 1.21 or higher
- **Kubernetes/OpenShift**: Access to a cluster with valid kubeconfig
- **RBAC Permissions**: Read access to cluster resources
- **Dependencies**: All managed via `go.mod` (k8s.io/client-go, k8s.io/apimachinery, etc.)

### For Comparison Mode
- Two valid kubeconfig files
- Network access to both clusters
- Appropriate RBAC permissions in both clusters

## Container Usage

### Build Container
```bash
docker build -t k8s-resource-collector .
```

### Run Container

**Directory mode:**
```bash
docker run --rm \
  -v "$KUBECONFIG:/root/.kube/config:ro" \
  -v "$(pwd)/output:/app/output" \
  k8s-resource-collector \
  --verbose --output /app/output
```

**Single file mode:**
```bash
docker run --rm \
  -v "$KUBECONFIG:/root/.kube/config:ro" \
  -v "$(pwd)/output:/app/output" \
  k8s-resource-collector \
  --single-file --file /app/output/all-resources.yaml
```

**Comparison mode:**
```bash
docker run --rm \
  -v ~/.kube/config-1:/root/.kube/config-1:ro \
  -v ~/.kube/config-2:/root/.kube/config-2:ro \
  -v "$(pwd)/output:/app/output" \
  k8s-resource-collector \
  --kubeconfig1 /root/.kube/config-1 \
  --kubeconfig2 /root/.kube/config-2 \
  --output /app/output
```

### Using the Example Script
```bash
chmod +x run-container-example.sh
./run-container-example.sh
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on:

- How to set up your development environment
- Code style and standards
- Testing requirements
- Submitting pull requests
- Review process

Quick start:
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes following our [coding standards](CONTRIBUTING.md#coding-standards)
4. Add tests and ensure they pass (`make test-all`)
5. Commit your changes (`git commit -m 'feat: add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

For detailed guidelines, please read [CONTRIBUTING.md](CONTRIBUTING.md).

## Troubleshooting

### Common Issues

**Issue: "failed to get kubeconfig"**
- Ensure the kubeconfig file exists and is readable
- Check that `KUBECONFIG` environment variable is set correctly
- Verify the file has valid YAML syntax

**Issue: "failed to create discovery client"**
- Check network connectivity to the Kubernetes cluster
- Verify the cluster endpoint in kubeconfig is correct
- Ensure you have valid credentials

**Issue: "permission denied" errors**
- Verify your RBAC permissions allow listing resources
- Check if you need to use a service account with appropriate roles
- Some resources may require admin privileges

**Issue: Comparison mode fails**
- Ensure both kubeconfig files are valid
- Check that both clusters are accessible
- Verify current context is set in both kubeconfig files

**Issue: Empty output files**
- Check if the cluster has any resources of that type
- Verify RBAC permissions for the resource types
- Look for errors in verbose output

**Issue: Deprecation warnings from Kubernetes API**
- The tool automatically detects and uses non-deprecated replacement APIs
- Use `--verbose` flag to see which APIs are being used as replacements
- Example: Instead of `v1/endpoints`, the tool collects `discovery.k8s.io/v1/endpointslices`


**Issue: Must-gather directory not found**
- Verify the path exists: `ls -la ./must-gather.local.xxx/`
- Check for typos in the path
- Use absolute paths if relative paths don't work
- Ensure you have read permissions on the directory

**Issue: "mutually exclusive" error with --must-gather and --kubeconfig**
- You cannot use both `--must-gather` and `--kubeconfig` at the same time
- Choose one: either process a must-gather directory (offline) OR connect to a live cluster
- For live cluster: use `--kubeconfig`
- For offline processing: use `--must-gather`

### Performance Tips

1. **Use `--verbose`** to see progress and identify slow resources
2. **Network latency**: Collection time depends on cluster size and network speed
3. **Large clusters**: Consider collecting specific namespaces if available
4. **Comparison mode**: Collects from clusters sequentially; time = sum of both collections

### Getting Help


- Review [test examples](tests/README.md)
- Open an issue on GitHub with verbose output
