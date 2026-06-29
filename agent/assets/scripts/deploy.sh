#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

BRANCH="${BRANCH:?BRANCH is required}"
IS_PRODUCTION="${IS_PRODUCTION:-false}"
ODOO_VERSION="${ODOO_VERSION:-19.0}"
if [[ "$ODOO_VERSION" == "<nil>" || -z "$ODOO_VERSION" ]]; then ODOO_VERSION="19.0"; fi
ADDONS_REPOSITORY="${ADDONS_REPOSITORY:?ADDONS_REPOSITORY is required}"

ENV_DIR=$(env_dir "$BRANCH")
ADDONS_DIR=$(addons_dir "$BRANCH")
CONTAINER=$(container "$BRANCH")

mkdir -p "$ENV_DIR" "$ADDONS_DIR" "$BACKUP_DIR"

caddy_add_route "$BRANCH" "$DOMAIN"

clone_or_pull "$ADDONS_REPOSITORY" "$ADDONS_DIR"

cat > "$ENV_DIR/Dockerfile" <<DOCKERFILE
FROM odoo:$ODOO_VERSION
DOCKERFILE

if [ -f "$ADDONS_DIR/requirements.txt" ]; then
  cp "$ADDONS_DIR/requirements.txt" "$ENV_DIR/requirements.txt"
  cat >> "$ENV_DIR/Dockerfile" <<DOCKERFILE
COPY requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt
DOCKERFILE
fi

cat > "$ENV_DIR/odoo.conf" <<ODOO
[options]
addons_path = /mnt/extra-addons,/usr/lib/python3/dist-packages/odoo/addons
data_dir = /var/lib/odoo
db_host = deploy-db
db_user = odoo
db_password = ${POSTGRES_PASSWORD:-odoo}
db_template = template0
ODOO

docker build -t "odoo-${BRANCH}" "$ENV_DIR"
docker rm -f "$CONTAINER" 2>/dev/null || true

DB_EXISTS=false
if docker exec deploy-db psql -U odoo -tAc "SELECT 1 FROM pg_database WHERE datname='$BRANCH'" | grep -q 1; then
  DB_EXISTS=true
fi

if [ "$DB_EXISTS" = "false" ]; then
  TEMPLATE="_latest"
  if docker exec deploy-db psql -U odoo -tAc "SELECT 1 FROM pg_database WHERE datname='$TEMPLATE'" | grep -q 1; then
    db_create "$BRANCH" -T "$TEMPLATE"
  else
    db_create "$BRANCH"
  fi
fi

if [ "$DB_EXISTS" = "false" ]; then
  docker run --rm --name "${BRANCH}-init" --network deploy-db-network \
    -v "$ADDONS_DIR:/mnt/extra-addons" \
    -v "$ENV_DIR/odoo.conf:/etc/odoo/odoo.conf" \
    "odoo-${BRANCH}" \
    odoo -c /etc/odoo/odoo.conf -d "$BRANCH" -i base --stop-after-init
fi

docker run -d --name "$CONTAINER" --network web --restart unless-stopped \
  --label "deploy.branch=$BRANCH" \
  --label "deploy.is_production=$IS_PRODUCTION" \
  --label "deploy.odoo_version=$ODOO_VERSION" \
  -v "$ADDONS_DIR:/mnt/extra-addons" \
  -v "$ENV_DIR/odoo.conf:/etc/odoo/odoo.conf" \
  "odoo-${BRANCH}" \
  odoo -c /etc/odoo/odoo.conf -d "$BRANCH"

docker network connect deploy-db-network "$CONTAINER"

echo "Deployed branch $BRANCH (production=$IS_PRODUCTION)"
