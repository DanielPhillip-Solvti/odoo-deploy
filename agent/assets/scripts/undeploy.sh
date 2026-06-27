#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

BRANCH="${BRANCH:?BRANCH is required}"

heartbeat=$(read_heartbeat)

updated=$(echo "$heartbeat" | BRANCH="$BRANCH" python3 -c "
import json, os, sys
data = json.load(sys.stdin)
branch = os.environ['BRANCH']
if data.get('production_branch', {}).get('branch') == branch:
    data['production_branch'] = None
data['staging_branches'] = [e for e in data.get('staging_branches', []) if e.get('branch') != branch]
json.dump(data, sys.stdout)
")

write_heartbeat "$updated"
echo "Undeployed branch $BRANCH"
