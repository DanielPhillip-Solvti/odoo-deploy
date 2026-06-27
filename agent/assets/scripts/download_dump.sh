#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

IS_PRODUCTION="${IS_PRODUCTION:-false}"
BINARY_DIR="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="$BINARY_DIR/backups"

if [ ! -d "$BACKUP_DIR" ]; then
    echo "ERROR: No backups directory found" >&2
    exit 1
fi

echo "Download dump completed (stub, production=$IS_PRODUCTION)"
