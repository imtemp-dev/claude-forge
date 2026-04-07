#!/bin/bash
# BTS stop hook — forwards to bts binary (propagates exit 2 for block)
temp_file=$(mktemp)
trap 'rm -f "$temp_file"' EXIT
cat > "$temp_file"

if command -v bts &> /dev/null; then
  bts hook stop < "$temp_file"
  rc=$?; [ $rc -eq 2 ] && exit 2
  exit 0
fi

# Try local binary
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOCAL_BIN="${SCRIPT_DIR}/../../bin/bts"
if [ -f "$LOCAL_BIN" ]; then
  "$LOCAL_BIN" hook stop < "$temp_file"
  rc=$?; [ $rc -eq 2 ] && exit 2
  exit 0
fi

exit 0
