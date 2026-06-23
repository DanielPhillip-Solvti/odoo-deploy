#!/usr/bin/env bash
set -euo pipefail

echo "🔐 GitHub / GHCR Login Setup"

# Prompt for username
read -rp "GitHub Username: " GITHUB_USER

# Prompt for PAT (hidden input)
read -srp "GitHub PAT: " GITHUB_PAT
echo ""

# Validate inputs
if [[ -z "$GITHUB_USER" || -z "$GITHUB_PAT" ]]; then
  echo "❌ Username or PAT cannot be empty"
  exit 1
fi

echo "🐳 Logging into GHCR..."

echo "$GITHUB_PAT" | docker login ghcr.io -u "$GITHUB_USER" --password-stdin

echo "⚙️ Configuring Git to use PAT for GitHub HTTPS URLs..."

git config --global url."https://x-access-token:${GITHUB_PAT}@github.com/".insteadOf "https://github.com/"

echo "🧹 Optionally storing credential helper (recommended fallback)..."

git config --global credential.helper store

# Log in to deploy-agent container if it's running to ensure it can pull from GHCR

if docker ps --format '{{.Names}}' | grep -qx 'deploy-agent'; then
echo "🐳 Logging deploy-agent container into GHCR..."

docker exec -i deploy-agent sh -c "
mkdir -p /root/.docker &&
cat >/tmp/ghcr_pat &&
docker login ghcr.io -u '$GITHUB_USER' --password-stdin </tmp/ghcr_pat &&
rm -f /tmp/ghcr_pat
" <<< "$GITHUB_PAT"

echo "✅ deploy-agent authenticated"
else
echo "⚠️ deploy-agent container not running, skipping container login"
fi

echo ""
echo "✅ Login complete!"
echo "   - GHCR authenticated"
echo "   - Git configured for HTTPS PAT auth"
echo ""
echo "⚠️ Note: PAT is stored in Git config rewrite rule (not ideal for shared machines)"