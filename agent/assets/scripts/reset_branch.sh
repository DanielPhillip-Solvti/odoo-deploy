#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

BRANCH="${BRANCH:?BRANCH is required}"
ADDONS_DIR=$(addons_dir "$BRANCH")
CONTAINER=$(container "$BRANCH")

if [ -d "$ADDONS_DIR/.git" ]; then
  git -C "$ADDONS_DIR" fetch origin "$BRANCH"
  git -C "$ADDONS_DIR" reset --hard "origin/$BRANCH"
  git -C "$ADDONS_DIR" clean -fd
fi

ENV_DIR=$(env_dir "$BRANCH")
docker build -t "odoo-${BRANCH}" "$ENV_DIR"

TEMPLATE="_latest"
db_drop "$BRANCH"

if docker exec deploy-db psql -U odoo -tAc "SELECT 1 FROM pg_database WHERE datname='$TEMPLATE'" | grep -q 1; then
  db_create "$BRANCH" -T "$TEMPLATE"
  docker restart "$CONTAINER" 2>/dev/null || true
else
  db_create "$BRANCH"
  docker rm -f "$CONTAINER" 2>/dev/null || true
  docker run --rm --name "${BRANCH}-init" --network deploy-db-network \
    -v "$ADDONS_DIR:/mnt/extra-addons" \
    -v "$ENV_DIR/odoo.conf:/etc/odoo/odoo.conf" \
    "odoo-${BRANCH}" \
    odoo -c /etc/odoo/odoo.conf -i base --stop-after-init
  docker run -d --name "$CONTAINER" --network web --restart unless-stopped \
    -v "$ADDONS_DIR:/mnt/extra-addons" \
    -v "$ENV_DIR/odoo.conf:/etc/odoo/odoo.conf" \
    "odoo-${BRANCH}" \
    odoo -c /etc/odoo/odoo.conf
  docker network connect deploy-db-network "$CONTAINER"
fi

echo "Reset branch $BRANCH to origin/$BRANCH"
