package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imtemp-dev/claude-bts/internal/state"
)

func setupStopRoot(t *testing.T) (root, recipeID string) {
	t.Helper()
	root = t.TempDir()
	recipeID = "r-001-test"
	recipeDir := filepath.Join(root, ".bts", "specs", "recipes", recipeID)
	if err := os.MkdirAll(recipeDir, 0755); err != nil {
		t.Fatalf("mkdir recipe: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".bts", "local"), 0755); err != nil {
		t.Fatalf("mkdir local: %v", err)
	}
	configDir := filepath.Join(root, ".bts", "config")
	_ = os.MkdirAll(configDir, 0755)
	_ = os.WriteFile(filepath.Join(configDir, ".template-version"), []byte("test"), 0644)
	t.Setenv("HOME", t.TempDir())

	recipe := &state.RecipeState{
		ID:    recipeID,
		Type:  "blueprint",
		Phase: "verify",
	}
	if err := state.SaveRecipeState(root, recipe); err != nil {
		t.Fatalf("save recipe: %v", err)
	}
	// verification.md must exist for the gate to proceed
	verifyPath := filepath.Join(recipeDir, "verification.md")
	if err := os.WriteFile(verifyPath, []byte("stub"), 0644); err != nil {
		t.Fatalf("write verification.md: %v", err)
	}
	return root, recipeID
}

func writeVerifyLog(t *testing.T, root, recipeID string, entries []state.VerifyLogEntry) {
	t.Helper()
	path := filepath.Join(state.RecipeDir(root, recipeID), "verify-log.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create log: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for i := range entries {
		if err := enc.Encode(&entries[i]); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
}

// TestStopSpecDone_BlocksOnResolvableMinor — Phase 1.3 core guarantee:
// a converged-looking entry (critical=0, major=0) with an outstanding
// [resolvable] minor must still block completion.
func TestStopSpecDone_BlocksOnResolvableMinor(t *testing.T) {
	root, recipeID := setupStopRoot(t)
	writeVerifyLog(t, root, recipeID, []state.VerifyLogEntry{
		{Iteration: 1, Critical: 0, Major: 0, MinorResolvable: 1, MinorDeferred: 0, Status: "continue"},
	})

	h := NewStopHandler()
	out, err := h.Handle(&HookInput{CWD: root, StopHookContent: "<bts>DONE</bts>"})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if out.Decision != "block" {
		t.Fatalf("expected block, got decision=%q", out.Decision)
	}
	if !strings.Contains(out.Reason, "minor [resolvable]") {
		t.Errorf("reason should cite minor [resolvable], got %q", out.Reason)
	}
}

// TestStopSpecDone_AllowsOnlyDeferredMinors — [deferred] minors are
// runtime watch-items and do not block completion.
func TestStopSpecDone_AllowsOnlyDeferredMinors(t *testing.T) {
	root, recipeID := setupStopRoot(t)
	writeVerifyLog(t, root, recipeID, []state.VerifyLogEntry{
		{Iteration: 1, Critical: 0, Major: 0, MinorResolvable: 0, MinorDeferred: 2, Status: "converged"},
	})

	h := NewStopHandler()
	out, err := h.Handle(&HookInput{CWD: root, StopHookContent: "<bts>DONE</bts>"})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if out.Decision == "block" {
		t.Fatalf("expected pass-through, got block: %s", out.Reason)
	}
}

// TestStopSpecDone_LegacyMinorFieldBlocks — legacy log entries predate the
// resolvable/deferred split. EffectiveResolvable() treats legacy Minor>0
// as resolvable (conservative). Ensures existing recipes do not silently
// slip past the new gate.
func TestStopSpecDone_LegacyMinorFieldBlocks(t *testing.T) {
	root, recipeID := setupStopRoot(t)
	writeVerifyLog(t, root, recipeID, []state.VerifyLogEntry{
		{Iteration: 1, Critical: 0, Major: 0, Minor: 3, Status: "continue"},
	})

	h := NewStopHandler()
	out, err := h.Handle(&HookInput{CWD: root, StopHookContent: "<bts>DONE</bts>"})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if out.Decision != "block" {
		t.Fatalf("legacy Minor=3 must block; got decision=%q", out.Decision)
	}
}
