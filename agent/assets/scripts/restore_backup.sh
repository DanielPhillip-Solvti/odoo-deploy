#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

BRANCH="${BRANCH:?BRANCH is required}"

echo "Restore backup completed (stub) for branch $BRANCH"
