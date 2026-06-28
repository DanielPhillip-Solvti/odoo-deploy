#!/bin/bash
set -euo pipefail

BRANCH="${BRANCH:?BRANCH is required}"
CONTAINER_NAME="deploy-${BRANCH}"

# Remove the container
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true

echo "Undeployed branch $BRANCH"
