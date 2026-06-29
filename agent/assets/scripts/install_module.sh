#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

BRANCH="${BRANCH:?BRANCH is required}"
MODULE_NAME="${MODULE_NAME:?MODULE_NAME is required}"
CONTAINER=$(container "$BRANCH")

docker exec "$CONTAINER" odoo -c /etc/odoo/odoo.conf -d "$BRANCH" -i "$MODULE_NAME" --stop-after-init

echo "Installed module $MODULE_NAME on branch $BRANCH"
