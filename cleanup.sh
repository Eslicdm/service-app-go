#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔥 Deleting Kind Kubernetes cluster 'service-app'...${NC}"

if kind get clusters | grep -q "^service-app$"; then
    kind delete cluster --name service-app
    echo -e "${GREEN}✅ Cluster 'service-app' deleted successfully.${NC}"
else
    echo -e "${GREEN}Kind cluster 'service-app' does not exist. Nothing to do.${NC}"
fi
