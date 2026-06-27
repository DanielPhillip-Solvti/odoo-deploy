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

echo ""
echo "✅ Login complete!"
echo "   - GHCR authenticated"
echo "   - Git configured for HTTPS PAT auth"
echo ""