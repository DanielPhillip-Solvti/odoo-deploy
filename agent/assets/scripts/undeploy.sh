#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

BRANCH="${BRANCH:?BRANCH is required}"
CONTAINER=$(container "$BRANCH")

caddy_remove_route "$BRANCH" "$DOMAIN"
docker rm -f "$CONTAINER" 2>/dev/null || true
db_drop "$BRANCH"
rm -rf "$(env_dir "$BRANCH")"

echo "Undeployed branch $BRANCH"
