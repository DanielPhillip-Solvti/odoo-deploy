#!/bin/bash
set -euo pipefail

BRANCH="${BRANCH:?BRANCH is required}"
IS_PRODUCTION="${IS_PRODUCTION:-false}"
ODOO_VERSION="${ODOO_VERSION:-19.0}"
CONTAINER_NAME="deploy-${BRANCH}"

# Remove existing container if any
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true

# Create a lightweight log-generating container with Python (for future shell access)
docker run -d \
    --name "$CONTAINER_NAME" \
    --label "deploy.branch=${BRANCH}" \
    --label "deploy.is_production=${IS_PRODUCTION}" \
    --label "deploy.odoo_version=${ODOO_VERSION}" \
    -e BRANCH="${BRANCH}" \
    python:3-alpine \
    python3 -c "
import sys, time, os
from datetime import datetime, timezone
i = 0
while True:
    i += 1
    ts = datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M:%S')
    print(f'[{ts}] [{os.environ[\"BRANCH\"]}] [INFO] Log entry #{i}: Environment running')
    sys.stdout.flush()
    time.sleep(3)
"

echo "Deployed branch $BRANCH (production=$IS_PRODUCTION)"
