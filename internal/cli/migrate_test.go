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
