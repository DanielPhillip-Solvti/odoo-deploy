#!/bin/bash
set -euo pipefail

WITH_DUMP="${WITH_DUMP:-false}"
BRANCH="${BRANCH:-unknown}"

if [ "$WITH_DUMP" = "true" ]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    BINARY_DIR="$(dirname "$SCRIPT_DIR")"
    BACKUP_DIR="$BINARY_DIR/backups"
    mkdir -p "$BACKUP_DIR"
    DATE=$(date -u +%y%m%d)

    echo "Backup test file for $BRANCH at $(date -u)" > "$BACKUP_DIR/${BRANCH}_${DATE}.dump"
    echo "Backup test file (neutralised) for $BRANCH at $(date -u)" > "$BACKUP_DIR/${BRANCH}_${DATE}_neutralised.dump"
fi

echo "Backup completed (with_dump=$WITH_DUMP, branch=$BRANCH)"
