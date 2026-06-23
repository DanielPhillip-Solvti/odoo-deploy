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
AGENT_VM_ID=""
AGENT_OTP=""
POSTGRES_USER=""
POSTGRES_PASSWORD=""
ENV="production"

while [[ $# -gt 0 ]]; do
  case $1 in
    --odoo-url)
      ODOO_URL="$2"; shift 2 ;;
    --vm-id)
      AGENT_VM_ID="$2"; shift 2 ;;
    --otp)
      AGENT_OTP="$2"; shift 2 ;;
    --postgres-user)
      POSTGRES_USER="$2"; shift 2 ;;
    --postgres-password)
      POSTGRES_PASSWORD="$2"; shift 2 ;;
    --env)
      ENV="$2"; shift 2 ;;
    *)
      echo "❌ Unknown parameter: $1"
      exit 1 ;;
  esac
done

# -----------------------------
# VALIDATION
# -----------------------------

if [[ -z "$ODOO_URL" || -z "$AGENT_VM_ID" || -z "$AGENT_OTP" ]]; then
    echo "❌ Missing required agent parameters (--odoo-url, --vm-id, --otp)"
    exit 1
fi

if [[ -z "$POSTGRES_USER" || -z "$POSTGRES_PASSWORD" ]]; then
    echo "❌ Missing postgres credentials"
    exit 1
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
# FETCH RUNTIME ASSETS
# -----------------------------
echo "⬇️ Downloading runtime configuration files..."

curl -fsSL "$ODOO_URL/agent/get_script/docker-compose/yml" -o docker-compose.yml
curl -fsSL "$ODOO_URL/agent/get_script/Caddyfile/0" -o Caddyfile
curl -fsSL "$ODOO_URL/agent/get_script/login/sh" -o login.sh

chmod +x login.sh

# -----------------------------
# ENV CONFIG
# -----------------------------
echo "📝 Writing environment configuration..."

cat > .env <<EOF
# Agent Config
ODOO_URL=$ODOO_URL
AGENT_VM_ID=$AGENT_VM_ID
AGENT_OTP=$AGENT_OTP

# Database Config
POSTGRES_USER=$POSTGRES_USER
POSTGRES_PASSWORD=$POSTGRES_PASSWORD

# Runtime
ENV=$ENV
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
# SUMMARY
# -----------------------------
echo ""
echo "✅ Deployment complete!"
echo "📍 Environment: $ENV"
echo "📂 Workspace: $TARGET_DIR"
echo ""
echo "📊 Status:"
docker compose ps

echo ""
echo "📌 Useful commands:"
echo "   cd $TARGET_DIR"
echo "   docker compose logs -f"
echo "   docker compose restart"