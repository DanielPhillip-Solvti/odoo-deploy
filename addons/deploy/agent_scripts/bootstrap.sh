#!/usr/bin/env bash
set -euo pipefail

echo "⚙️ Checking server runtime prerequisites..."

if ! command -v docker >/dev/null 2>&1; then
    echo "❌ Docker is missing. Install Docker first." >&2
    exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
    echo "❌ Docker Compose plugin is missing." >&2
    exit 1
fi

# -----------------------------
# ARGUMENT PARSING
# -----------------------------
ODOO_URL=""
API_KEY=""
REPO_URL=""
POSTGRES_USER=odoo
POSTGRES_PASSWORD=odoo

while [[ $# -gt 0 ]]; do
  case $1 in
    --odoo-url)
      ODOO_URL="$2"; shift 2 ;;
    --api-key)
      API_KEY="$2"; shift 2 ;;
    --repo-url)
      REPO_URL="$2"; shift 2 ;;
    *)
      echo "❌ Unknown parameter: $1"
      exit 1 ;;
  esac
done

# -----------------------------
# VALIDATION
# -----------------------------

if [[ -z "$ODOO_URL" || -z "$API_KEY" ]]; then
    echo "❌ Missing required agent parameters (--odoo-url, --api-key)"
    exit 1
fi

if [[ -z "$POSTGRES_USER" || -z "$POSTGRES_PASSWORD" ]]; then
    echo "❌ Missing postgres credentials"
    exit 1
fi

# -----------------------------
# REPOSITORY PROMPT
# -----------------------------
if [[ -z "$REPO_URL" ]]; then
    echo ""
    echo "📦 No repository URL provided."
    echo "   Enter the Git repository URL for this deployment."
    echo "   Example: https://github.com/my-org/my-repo"
    echo ""
    read -r -p "Repository URL: " REPO_URL
    echo ""
fi

if [[ -z "$REPO_URL" ]]; then
    echo "❌ Repository URL is required."
    exit 1
fi

# -----------------------------
# SEND REPO URL TO ODOO
# -----------------------------
echo "📤 Sending repository URL to Odoo..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "$ODOO_URL/agent/update_config" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"call\",\"params\":{\"repository_url\":\"$REPO_URL\"}}")

if [[ "$HTTP_CODE" != "200" ]]; then
    echo "⚠️  Failed to update repository URL on Odoo (HTTP $HTTP_CODE). You can set it manually in the agent form."
else
    echo "✅ Repository URL registered with Odoo."
fi

# -----------------------------
# CONFIG
# -----------------------------
TARGET_DIR="/data/deploy-agent"

echo "📂 Setting up workspace at $TARGET_DIR..."

sudo mkdir -p "$TARGET_DIR"
sudo mkdir -p /var/run/postgresql
sudo chmod 777 /var/run/postgresql

sudo chown -R "$USER:$USER" "$TARGET_DIR"
cd "$TARGET_DIR"

# -----------------------------
# FETCH AGENT BINARY & EXTRACT ASSETS
# -----------------------------
# fetch release https://github.com/DanielPhillip-Solvti/odoo-deploy-agent/releases/tag/v1.0.0
echo "⬇️ Downloading agent binary..."
curl -L https://github.com/DanielPhillip-Solvti/odoo-deploy/releases/download/v1.0.0/agent -o "$TARGET_DIR/agent"
chmod +x "$TARGET_DIR/agent"

echo "📂 Extracting runtime assets (scripts, Caddyfile, docker-compose.yml)..."
"$TARGET_DIR/agent" --extract-assets "$TARGET_DIR"

# -----------------------------
# ENV CONFIG
# -----------------------------
echo "📝 Writing environment configuration..."

cat > .env <<EOF
# Agent Config
ODOO_URL=$ODOO_URL
API_KEY=$API_KEY
REPO_URL=$REPO_URL

# Database Config
POSTGRES_USER=$POSTGRES_USER
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
EOF

chmod 600 .env

# -----------------------------
# OPTIONAL LOGIN STEP
# -----------------------------
echo "🔐 Running login bootstrap (GHCR / Git access)..."

if [[ -f "./login.sh" ]]; then
    bash ./login.sh || echo "⚠️ Login script failed or skipped"
fi

# -----------------------------
# DOCKER DEPLOYMENT
# -----------------------------
echo "🚀 Deploying container stack..."

docker compose pull
docker compose down || true
docker compose up -d

# -----------------------------
# HEALTH CHECK
# -----------------------------
echo "🧪 Waiting for services to stabilize..."

sleep 5

if docker compose ps | grep -q "Exit"; then
    echo "⚠️ Some containers are not healthy. Check logs:"
    docker compose ps
    exit 1
fi

# -----------------------------
# START AGENT
# -----------------------------
set -a && . "$TARGET_DIR/.env"
"$TARGET_DIR/agent"