# Multi-Cluster Comparison Feature

## Overview

The k8s-resource-collector now supports comparing resources between two Kubernetes clusters. This feature allows you to:
- Collect resources from two different clusters
- Generate unique filenames based on cluster names
- Create a diff report showing differences between clusters

## Usage

### Basic Comparison

Compare resources between two clusters:

```bash
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/config-cluster1 \
  --kubeconfig2 ~/.kube/config-cluster2
```

### With Compare Flag

Explicitly enable comparison mode:

```bash
./bin/k8s-resource-collector \
  --compare \
  --kubeconfig1 ~/.kube/config-cluster1 \
  --kubeconfig2 ~/.kube/config-cluster2
```

### Custom Output Directory

Specify where comparison results should be saved:

```bash
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/config-cluster1 \
  --kubeconfig2 ~/.kube/config-cluster2 \
  --output ./my-comparison
```

### With Verbose Output

See detailed progress:

```bash
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/config-cluster1 \
  --kubeconfig2 ~/.kube/config-cluster2 \
  --verbose
```

## Output Structure

When running in comparison mode, the tool creates:

```
output/
└── comparison/
    ├── cluster1-name-resources.yaml   # All resources from cluster 1
    ├── cluster2-name-resources.yaml   # All resources from cluster 2
    └── diff-cluster1-name-vs-cluster2-name.txt  # Diff report
```

## Diff Report Format

The diff report includes:

### Header
- Generation timestamp
- Cluster names
- Total resource counts

### Resources Only in Cluster 1
Lists resources that exist only in the first cluster

### Resources Only in Cluster 2
Lists resources that exist only in the second cluster

### Common Resources
Lists resources present in both clusters

### Summary
- Total resources in each cluster
- Count of unique resources per cluster
- Count of common resources

## Example Output

```
=== Multi-Cluster Comparison Mode ===
Cluster 1: ~/.kube/config-prod
Cluster 2: ~/.kube/config-staging

[1/3] Collecting from cluster 1: prod-cluster
✓ Saved to: output/comparison/prod-cluster-resources.yaml

[2/3] Collecting from cluster 2: staging-cluster
✓ Saved to: output/comparison/staging-cluster-resources.yaml

[3/3] Generating difference report...
✓ Diff saved to: output/comparison/diff-prod-cluster-vs-staging-cluster.txt

=== Comparison Complete ===
Cluster 1 (prod-cluster): output/comparison/prod-cluster-resources.yaml
Cluster 2 (staging-cluster): output/comparison/staging-cluster-resources.yaml
Difference:     output/comparison/diff-prod-cluster-vs-staging-cluster.txt
```

## Use Cases

### 1. Production vs Staging Comparison
Verify that your staging environment matches production:

```bash
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/prod-config \
  --kubeconfig2 ~/.kube/staging-config
```

### 2. Pre/Post Migration Validation
Compare clusters before and after migration:

```bash
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/old-cluster \
  --kubeconfig2 ~/.kube/new-cluster
```

### 3. Multi-Region Comparison
Compare resources across regions:

```bash
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/us-east-config \
  --kubeconfig2 ~/.kube/eu-west-config
```

### 4. Disaster Recovery Validation
Verify DR cluster matches primary:

```bash
./bin/k8s-resource-collector \
  --kubeconfig1 ~/.kube/primary-cluster \
  --kubeconfig2 ~/.kube/dr-cluster
```

## File Naming

Filenames are automatically generated based on the cluster name extracted from the kubeconfig:
- Special characters are replaced with hyphens
- The format is: `{cluster-name}-resources.yaml`
- Diff files use: `diff-{cluster1}-vs-{cluster2}.txt`

## Requirements

- Both kubeconfig files must be valid and accessible
- Each kubeconfig must have a current context set
- The user must have appropriate RBAC permissions in both clusters
- Both clusters must be reachable from the machine running the tool

## Limitations

- Currently compares resource types, not individual resource instances
- Does not perform deep object comparison
- Resource order in YAML files may differ between clusters
- Very large clusters may take significant time to collect

## Tips

1. **Use specific contexts**: Ensure your kubeconfig files point to the correct contexts
2. **Clean old comparisons**: Use `--clean` to remove old comparison data
3. **Run during off-peak**: Collection can be resource-intensive on large clusters
4. **Review diff carefully**: The diff shows resource types, not detailed differences

## Troubleshooting

### Error: "comparison mode requires both --kubeconfig1 and --kubeconfig2"
**Solution**: Provide both kubeconfig files when using comparison mode

### Error: "failed to get cluster name from kubeconfig"
**Solution**: Ensure the kubeconfig file has a valid current-context set

### Error: "no current context set in kubeconfig"
**Solution**: Set a current context in your kubeconfig:
```bash
kubectl config use-context <context-name> --kubeconfig=<path>
```

### Empty diff report
**Possible causes**:
- Both clusters have identical resources (expected)
- Collection failed silently (check individual YAML files)
- No resources found in one or both clusters

## Advanced Usage

### Makefile Integration

Add comparison to your Makefile:

```makefile
compare-clusters:
	./bin/k8s-resource-collector \
		--kubeconfig1 $(CLUSTER1_CONFIG) \
		--kubeconfig2 $(CLUSTER2_CONFIG) \
		--output ./comparison-results \
		--verbose
```

### CI/CD Pipeline

Use in automation:

```yaml
- name: Compare Clusters
  run: |
    ./bin/k8s-resource-collector \
      --kubeconfig1 ${KUBECONFIG_PROD} \
      --kubeconfig2 ${KUBECONFIG_STAGING} \
      --output ./comparison
    
    # Check if diff is empty (clusters are identical)
    if [ -s ./comparison/diff-*.txt ]; then
      echo "Differences found between clusters"
      cat ./comparison/diff-*.txt
      exit 1
    fi
```

## Contributing

To enhance the comparison feature:
1. Fork the repository
2. Add your improvements to `cmd/main.go`
3. Update tests in `collector_test.go`
4. Submit a pull request

## See Also

- [Main README](README.md)
- [Usage Examples](run-container-example.sh)
- [Testing Guide](tests/README.md)

