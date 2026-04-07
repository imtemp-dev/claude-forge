#!/bin/bash
# BTS session-end hook — forwards to bts binary
temp_file=$(mktemp)
trap 'rm -f "$temp_file"' EXIT
cat > "$temp_file"

if command -v bts &> /dev/null; then
  bts hook session-end < "$temp_file"
  exit 0
fi

# Try local binary
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOCAL_BIN="${SCRIPT_DIR}/../../bin/bts"
if [ -f "$LOCAL_BIN" ]; then
  "$LOCAL_BIN" hook session-end < "$temp_file"
  exit 0
fi

exit 0
