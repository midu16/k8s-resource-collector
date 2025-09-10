#!/bin/bash

# Example script to run k8s-resource-collector in a container

set -e

# Configuration
IMAGE_NAME="k8s-resource-collector"
CONTAINER_NAME="k8s-resource-collector-run"
OUTPUT_DIR="./output"
KUBECONFIG_PATH="${KUBECONFIG:-$HOME/.kube/config}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building k8s-resource-collector container...${NC}"
docker build -t $IMAGE_NAME .

echo -e "${GREEN}Creating output directory...${NC}"
mkdir -p $OUTPUT_DIR

echo -e "${GREEN}Running k8s-resource-collector container...${NC}"
echo -e "${YELLOW}Note: Make sure your kubeconfig is accessible at: $KUBECONFIG_PATH${NC}"

# Run the container with kubeconfig mounted
docker run --rm \
  --name $CONTAINER_NAME \
  -v "$KUBECONFIG_PATH:/root/.kube/config:ro" \
  -v "$(pwd)/$OUTPUT_DIR:/app/output" \
  $IMAGE_NAME \
  --verbose \
  --output /app/output

echo -e "${GREEN}Collection completed! Check the output directory: $OUTPUT_DIR${NC}"

# Alternative: Run in single file mode
echo -e "${YELLOW}To run in single file mode, use:${NC}"
echo "docker run --rm \\"
echo "  --name $CONTAINER_NAME \\"
echo "  -v \"$KUBECONFIG_PATH:/root/.kube/config:ro\" \\"
echo "  -v \"\$(pwd)/$OUTPUT_DIR:/app/output\" \\"
echo "  $IMAGE_NAME \\"
echo "  --single-file \\"
echo "  --output-file /app/output/all-resources.yaml \\"
echo "  --verbose"
