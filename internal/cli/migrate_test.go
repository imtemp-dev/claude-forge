package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUpgradeVerifyLogLine_LegacyMinor(t *testing.T) {
	in := `{"iteration":1,"critical":0,"major":0,"minor":3,"status":"continue","timestamp":"2025-01-01T00:00:00Z"}`
	out, changed, err := upgradeVerifyLogLine(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	if !strings.Contains(out, `"minor_resolvable":3`) {
		t.Errorf("expected minor_resolvable=3 in %q", out)
	}
	// minor_deferred=0 gets omitempty'd out — that's fine, consumers default to 0.
	// But minor_resolvable=3 proves the migration happened.
	// status should remain continue (resolvable minor blocks convergence)
	if !strings.Contains(out, `"status":"continue"`) {
		t.Errorf("status should stay continue (resolvable minor blocks convergence), got %q", out)
	}
}

func TestUpgradeVerifyLogLine_AlreadyMigrated(t *testing.T) {
	in := `{"iteration":1,"critical":0,"major":0,"minor_resolvable":0,"minor_deferred":2,"status":"converged","timestamp":"2025-01-01T00:00:00Z"}`
	out, changed, err := upgradeVerifyLogLine(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if changed {
		t.Fatal("should not change already-migrated entry")
	}
	if out != in {
		t.Errorf("output should equal input byte-for-byte\n want %q\n got  %q", in, out)
	}
}

func TestUpgradeVerifyLogLine_ZeroMinor(t *testing.T) {
	in := `{"iteration":1,"critical":0,"major":0,"minor":0,"status":"converged","timestamp":"2025-01-01T00:00:00Z"}`
	out, changed, err := upgradeVerifyLogLine(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// changed=true because we rewrite to carry the explicit split fields.
	if !changed {
		t.Fatal("expected change (rewriting to new schema)")
	}
	// Ensure no regression: converged status preserved.
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if parsed["status"] != "converged" {
		t.Errorf("status should stay converged, got %v", parsed["status"])
	}
}

func TestUpgradeVerifyLogLine_BadJSON(t *testing.T) {
	in := `not json at all`
	out, changed, err := upgradeVerifyLogLine(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if changed {
		t.Error("bad json should be left alone, not changed")
	}
	if out != in {
		t.Errorf("bad json should pass through unchanged")
	}
}

// ---- settings migration -----------------------------------------------

func TestInjectSettingKey_IntoExistingBlock(t *testing.T) {
	in := "implement:\n  max_build_retries: 5\n  max_test_iterations: 5\n\nsimulate:\n  min_scenarios: 5\n"
	ins := settingsInsertion{
		Parent:  "implement",
		Key:     "midrun_review_every",
		Value:   "5",
		Comment: "Emit reviews every N tasks.",
		Since:   "v0.5.0",
	}
	out, ok := injectSettingKey(in, ins)
	if !ok {
		t.Fatal("expected injection to succeed")
	}
	if !strings.Contains(out, "  midrun_review_every: 5") {
		t.Errorf("inserted key missing: %s", out)
	}
	if !strings.Contains(out, "# Emit reviews every N tasks. (added in v0.5.0)") {
		t.Errorf("comment missing: %s", out)
	}
	// Must preserve the simulate: section after the blank line.
	if !strings.Contains(out, "\n\nsimulate:\n") {
		t.Errorf("blank separator before simulate: lost: %q", out)
	}
	// Must place the new key inside implement, before simulate.
	if strings.Index(out, "midrun_review_every") > strings.Index(out, "simulate:") {
		t.Errorf("new key landed after sibling section: %s", out)
	}
}

func TestInjectSettingKey_NoParentSection_Appends(t *testing.T) {
	in := "simulate:\n  min_scenarios: 5\n"
	ins := settingsInsertion{
		Parent: "implement", Key: "midrun_review_every",
		Value: "5", Comment: "X.", Since: "v0.5.0",
	}
	out, ok := injectSettingKey(in, ins)
	if !ok {
		t.Fatal("expected injection to succeed")
	}
	if !strings.Contains(out, "implement:\n  # X. (added in v0.5.0)\n  midrun_review_every: 5\n") {
		t.Errorf("fresh section not appended: %s", out)
	}
}

func TestInjectSettingKey_NestedBlock_ReturnsToParentIndent(t *testing.T) {
	// When the block ends with a deeper-indented child (retry_ladder),
	// the new key must land at the parent indent level, not under the child.
	in := "implement:\n  max_build_retries: 5\n  retry_ladder:\n    syntactic_max: 3\n"
	ins := settingsInsertion{
		Parent: "implement", Key: "midrun_review_every",
		Value: "5", Comment: "X.", Since: "v0.5.0",
	}
	out, _ := injectSettingKey(in, ins)
	if !strings.Contains(out, "\n  midrun_review_every: 5") {
		t.Errorf("key should land at 2-space indent, not nested: %q", out)
	}
}

func TestHasNestedKey(t *testing.T) {
	content := "implement:\n  max_build_retries: 5\n  midrun_review_every: 5\nsimulate:\n  max_build_retries: 9\n"
	if !hasNestedKey(content, "implement", "midrun_review_every") {
		t.Error("should detect midrun inside implement")
	}
	if hasNestedKey(content, "simulate", "midrun_review_every") {
		t.Error("should NOT detect midrun inside simulate block")
	}
	if hasNestedKey(content, "verify", "max_iterations") {
		t.Error("missing parent → false")
	}
}

func TestHasCommentedKey(t *testing.T) {
	if !hasCommentedKey("  # midrun_review_every: 5\n", "midrun_review_every") {
		t.Error("should detect commented-out key")
	}
	if hasCommentedKey("  midrun_review_every: 5\n", "midrun_review_every") {
		t.Error("active key is not a commented key")
	}
}

func TestDetectBlockIndent(t *testing.T) {
	if got := detectBlockIndent("\n  foo: 1\n"); got != "  " {
		t.Errorf("two-space block → got %q", got)
	}
	if got := detectBlockIndent("\n    foo: 1\n"); got != "    " {
		t.Errorf("four-space block → got %q", got)
	}
	if got := detectBlockIndent("\n\n"); got != "  " {
		t.Errorf("empty block → default two spaces, got %q", got)
	}
}
