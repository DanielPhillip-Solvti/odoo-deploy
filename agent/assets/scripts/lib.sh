#!/bin/bash

HEARTBEAT_FILE="${HEARTBEAT_FILE:-/data/deploy-agent/heartbeat.json}"

read_heartbeat() {
  if [ -f "$HEARTBEAT_FILE" ]; then
    cat "$HEARTBEAT_FILE"
  else
    echo '{"repo_url":"","production_branch":null,"staging_branches":[]}'
  fi
}

write_heartbeat() {
  local tmp
  tmp=$(mktemp)
  echo "$1" > "$tmp"
  mv "$tmp" "$HEARTBEAT_FILE"
}
