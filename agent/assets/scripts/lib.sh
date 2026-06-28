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
  local branch=$1 domain=$2 entry
  entry="http://${branch}.${domain} { reverse_proxy ${branch}:8069 }"
  if ! grep -qxF "$entry" "$CADDYFILE"; then
    echo "$entry" >> "$CADDYFILE"
    docker exec deploy-caddy caddy reload --config /etc/caddy/Caddyfile || echo "Warning: Caddy reload failed" >&2
  fi
}

caddy_remove_route() {
  local branch=$1 domain=$2
  sed -i "/http:\/\/${branch}\.${domain} {/,/^}/d" "$CADDYFILE"
  docker exec deploy-caddy caddy reload --config /etc/caddy/Caddyfile || echo "Warning: Caddy reload failed" >&2
}

db_drop() {
  docker exec deploy-db dropdb -U odoo --if-exists "$1" 2>/dev/null || true
}

db_create() {
  docker exec deploy-db createdb -U odoo "$@"
}

db_template_create() {
  local source=$1 template=$2
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
