SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_DIR="$(dirname "$SCRIPT_DIR")"

ENV_BASE="${ENV_BASE:-/data/deploy-agent/envs}"
ADDONS_BASE="${ADDONS_BASE:-/data/deploy-agent/addons}"
CADDYFILE="${CADDYFILE:-$AGENT_DIR/Caddyfile}"
BACKUP_DIR="${BACKUP_DIR:-/data/deploy-agent/backups}"
DOMAIN="${DOMAIN:-localhost}"

env_dir()     { echo "$ENV_BASE/$1"; }
addons_dir()  { echo "$ADDONS_BASE/$1"; }
container()   { echo "$1"; }

caddy_add_route() {
  local branch=$1 domain=$2 safe
  safe="${branch//-/_}"
  local pattern="@${safe} host ${branch}.${domain}"
  if ! grep -qF "$pattern" "$CADDYFILE"; then
    sed -i "s/^    # Branch routes are inserted here by deploy scripts/    ${pattern}\n    handle @${safe} {\n        reverse_proxy ${branch}:8069\n    }\n    # Branch routes are inserted here by deploy scripts/" "$CADDYFILE"
    docker exec deploy-caddy caddy reload --config /etc/caddy/Caddyfile || echo "Warning: Caddy reload failed" >&2
  fi
}

caddy_remove_route() {
  local branch=$1 domain=$2 safe
  safe="${branch//-/_}"
  sed -i "/^    @${safe} host ${branch}\.${domain}$/,/^    }$/d" "$CADDYFILE"
  docker exec deploy-caddy caddy reload --config /etc/caddy/Caddyfile || echo "Warning: Caddy reload failed" >&2
}

db_terminate_connections() {
  local db=$1
  docker exec deploy-db psql -U odoo -d postgres -c \
    "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='$db' AND pid <> pg_backend_pid();" \
    > /dev/null 2>&1 || true
}

db_drop() {
  docker exec deploy-db dropdb -U odoo --if-exists "$1" 2>/dev/null || true
}

db_create() {
  docker exec deploy-db createdb -U odoo "$@"
}

db_template_create() {
  local source=$1 template=$2
  db_terminate_connections "$source"
  db_drop "$template"
  db_create "$template" -T "$source"
}

db_restore_from_template() {
  local target=$1 template=$2
  db_drop "$target"
  db_create "$target" -T "$template"
}

clone_or_pull() {
  local repo=$1 dir=$2
  if [ -d "$dir/.git" ]; then
    git -C "$dir" pull --ff-only
  else
    git clone "$repo" "$dir"
  fi
}
