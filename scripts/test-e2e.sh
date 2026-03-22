#!/bin/bash
# test-e2e.sh — bts integration test
set -e

BTS="$(cd "$(dirname "$0")/.." && pwd)/bin/bts"
TEST_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_DIR"' EXIT
cd "$TEST_DIR"

echo "=== bts E2E Test ==="
echo "Binary: $BTS"
echo "Test dir: $TEST_DIR"
echo ""

# 1. Init
$BTS init . > /dev/null
[ -f .claude/skills/bts-verify/SKILL.md ] && echo "✓ 1. init" || { echo "✗ 1. init"; exit 1; }

# 2. Verify (no code, from-scratch spec)
printf "# OAuth2 Design\n\n**Auth component** handles user login.\n**Session manager** stores tokens.\nUses **Express** framework with **Passport.js**.\nData flows from **login form** to **OAuth provider** to **callback handler**.\nOn error, returns **401 Unauthorized**.\n" > spec.md
$BTS verify --no-code spec.md | grep -q '"level"' && echo "✓ 2. verify --no-code (level assessment)" || { echo "✗ 2. verify"; exit 1; }

# 3. Recipe log (verify iteration — backward compatible)
mkdir -p .bts/state/recipes/test-001
echo '{"id":"test-001","type":"blueprint","topic":"OAuth2","phase":"verify","iteration":1,"level":1.5,"started_at":"2026-03-18T00:00:00Z","updated_at":"2026-03-18T00:00:00Z"}' > .bts/state/recipes/test-001/recipe.json
$BTS recipe log test-001 --iteration 1 --critical 2 --major 1 > /dev/null
[ -f .bts/state/recipes/test-001/verify-log.jsonl ] && echo "✓ 3. recipe log (verify-log)" || { echo "✗ 3. verify-log"; exit 1; }

# 4. Recipe log (changelog action)
$BTS recipe log test-001 --action improve --output draft.md > /dev/null
[ -f .bts/state/recipes/test-001/changelog.jsonl ] && echo "✓ 4. recipe log (changelog)" || { echo "✗ 4. changelog"; exit 1; }

# 5. Recipe log (manifest update)
$BTS recipe log test-001 --action research --output research/v1.md --based-on "topic" > /dev/null
[ -f .bts/state/recipes/test-001/manifest.json ] && echo "✓ 5. recipe log (manifest)" || { echo "✗ 5. manifest"; exit 1; }

# 6. Recipe status (with Level)
$BTS recipe status | grep -q "Level" && echo "✓ 6. recipe status (Level shown)" || { echo "✗ 6. status"; exit 1; }

# 7. Debate log (create new)
$BTS debate log --topic "OAuth2 vs JWT" --round 1 --content "Expert 1: OAuth2 is standard-compliant" > /dev/null
$BTS debate list | grep -q "OAuth2 vs JWT" && echo "✓ 7. debate log + list" || { echo "✗ 7. debate"; exit 1; }

# 8. Debate resume
DEBATE_ID=$($BTS debate list 2>/dev/null | tail -1 | awk '{print $1}')
$BTS debate resume "$DEBATE_ID" | grep -q "Expert 1" && echo "✓ 8. debate resume" || { echo "✗ 8. resume"; exit 1; }

# 9. Debate log (add round 2)
$BTS debate log --id "$DEBATE_ID" --round 2 --content "Expert 2: JWT is stateless" > /dev/null
$BTS debate resume "$DEBATE_ID" | grep -q "Expert 2" && echo "✓ 9. debate round 2" || { echo "✗ 9. round 2"; exit 1; }

# 10. Sync check
$BTS sync-check test-001 2>&1 | grep -qE "sync|UNVERIFIED|issue" && echo "✓ 10. sync-check" || { echo "✗ 10. sync-check"; exit 1; }

# 11. Stop hook — should BLOCK (verify-log has critical>0)
RESULT=$(echo '{"session_id":"t","cwd":"'"$TEST_DIR"'","hook_event_name":"stop","content":"<bts>DONE</bts>"}' | $BTS hook stop 2>&1; echo "EXIT:$?")
echo "$RESULT" | grep -q "EXIT:2" && echo "✓ 11. stop hook blocks (critical>0)" || { echo "✗ 11. stop block"; exit 1; }

# 12. Add converged entry, stop hook should ALLOW
echo '{"iteration":2,"critical":0,"major":0,"minor":1,"status":"converged","timestamp":"2026-03-18T00:01:00Z"}' >> .bts/state/recipes/test-001/verify-log.jsonl
RESULT=$(echo '{"session_id":"t","cwd":"'"$TEST_DIR"'","hook_event_name":"stop","content":"<bts>DONE</bts>"}' | $BTS hook stop 2>&1; echo "EXIT:$?")
echo "$RESULT" | grep -q "EXIT:0" && echo "✓ 12. stop hook allows (converged)" || { echo "✗ 12. stop allow"; exit 1; }

echo ""
echo "=== All 12 tests passed ==="
