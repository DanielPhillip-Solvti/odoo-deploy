#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

BRANCH="${BRANCH:?BRANCH is required}"
WITH_DUMP="${WITH_DUMP:-false}"
TEMPLATE="_latest"

db_template_create "$BRANCH" "$TEMPLATE"

if [ "$WITH_DUMP" = "true" ]; then
  mkdir -p "$BACKUP_DIR"
  DATE=$(date -u +%y%m%d_%H%M%S)
  docker exec deploy-db pg_dump -U odoo -Fc "$TEMPLATE" > "$BACKUP_DIR/${BRANCH}_${DATE}.dump"
  echo "Dump created: ${BRANCH}_${DATE}.dump"
fi

echo "Backup completed for branch $BRANCH"
