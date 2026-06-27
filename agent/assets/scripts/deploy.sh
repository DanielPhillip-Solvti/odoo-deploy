#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

BRANCH="${BRANCH:?BRANCH is required}"
IS_PRODUCTION="${IS_PRODUCTION:-false}"
ODOO_VERSION="${ODOO_VERSION:-19.0}"

heartbeat=$(read_heartbeat)

updated=$(echo "$heartbeat" | BRANCH="$BRANCH" IS_PRODUCTION="$IS_PRODUCTION" ODOO_VERSION="$ODOO_VERSION" python3 -c "
import json, os, sys
data = json.load(sys.stdin)
branch = os.environ['BRANCH']
version = os.environ['ODOO_VERSION']
if os.environ['IS_PRODUCTION'] == 'true':
    data['production_branch'] = {'branch': branch, 'status': 'active', 'odoo_version': version}
else:
    data['staging_branches'] = [e for e in data.get('staging_branches', []) if e.get('branch') != branch]
    data['staging_branches'].append({'branch': branch, 'status': 'active', 'odoo_version': version})
json.dump(data, sys.stdout)
")

write_heartbeat "$updated"
echo "Deployed branch $BRANCH (production=$IS_PRODUCTION)"
