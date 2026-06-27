#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

WITH_DUMP="${WITH_DUMP:-false}"
BRANCH="${BRANCH:-unknown}"

if [ "$WITH_DUMP" = "true" ]; then
    BINARY_DIR="$(dirname "$SCRIPT_DIR")"
    BACKUP_DIR="$BINARY_DIR/backups"
    mkdir -p "$BACKUP_DIR"
    DATE=$(date -u +%y%m%d)

    echo "Backup test file for $BRANCH at $(date -u)" > "$BACKUP_DIR/${BRANCH}_${DATE}.dump"
    echo "Backup test file (neutralised) for $BRANCH at $(date -u)" > "$BACKUP_DIR/${BRANCH}_${DATE}_neutralised.dump"

    UPDATED=$(python3 -c "
import json, sys
data = json.load(sys.stdin)
backups = data.get('backups', [])
for f in ['${BRANCH}_${DATE}.dump', '${BRANCH}_${DATE}_neutralised.dump']:
    if f not in backups:
        backups.append(f)
data['backups'] = backups
json.dump(data, sys.stdout)
" < "$HEARTBEAT_FILE")
    write_heartbeat "$UPDATED"
fi

echo "Backup completed (with_dump=$WITH_DUMP, branch=$BRANCH)"
