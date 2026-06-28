#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

BRANCH="${BRANCH:?BRANCH is required}"
TEMPLATE="_latest"
CONTAINER=$(container "$BRANCH")

db_restore_from_template "$BRANCH" "$TEMPLATE"
docker restart "$CONTAINER" 2>/dev/null || true

echo "Restored $BRANCH from template $TEMPLATE"
