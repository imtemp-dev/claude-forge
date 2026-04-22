package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeArchitectWireframe(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "wireframe.md")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestCheckArchitectDecisionHeader_Valid(t *testing.T) {
	path := writeArchitectWireframe(t, `<!-- architect-decision -->
Selected: arrangement-centric
Rationale: owns word order as a single source of truth.
Rejected:
  - card-centric: duplicates placement across cards
<!-- /architect-decision -->

## Step 1: Components
…`)
	issues := CheckArchitectDecisionHeader(path)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d: %v", len(issues), issues)
	}
}

func TestCheckArchitectDecisionHeader_Missing(t *testing.T) {
	path := writeArchitectWireframe(t, "# Wireframe\n\n(no block here)\n")
	issues := CheckArchitectDecisionHeader(path)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].Claim, "missing_architect_decision_block") {
		t.Errorf("wrong claim: %s", issues[0].Claim)
	}
	if issues[0].Severity != "major" {
		t.Errorf("want major, got %s", issues[0].Severity)
	}
}

func TestCheckArchitectDecisionHeader_MissingSelected(t *testing.T) {
	path := writeArchitectWireframe(t, `<!-- architect-decision -->
Rationale: no Selected line present
Rejected: none
<!-- /architect-decision -->`)
	issues := CheckArchitectDecisionHeader(path)
	if len(issues) != 1 {
		t.Fatalf("want 1 issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].Claim, "missing_selected") {
		t.Errorf("wrong claim: %s", issues[0].Claim)
	}
}

func TestCheckArchitectDecisionHeader_DuplicateBlocks(t *testing.T) {
	path := writeArchitectWireframe(t, `<!-- architect-decision -->
Selected: first
<!-- /architect-decision -->

<!-- architect-decision -->
Selected: second
<!-- /architect-decision -->`)
	issues := CheckArchitectDecisionHeader(path)
	if len(issues) != 1 {
		t.Fatalf("want 1 issue for duplicate, got %d", len(issues))
	}
	if !strings.Contains(issues[0].Claim, "duplicate") {
		t.Errorf("wrong claim: %s", issues[0].Claim)
	}
}

// Integration: VerifyDocument on wireframe.md should surface this.
func TestVerifyDocument_ChecksArchitectDecision(t *testing.T) {
	path := writeArchitectWireframe(t, "# empty wireframe")
	result, err := VerifyDocument(path, "")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if result.Summary.Major < 1 {
		t.Fatalf("expected major for missing architect-decision; got %+v", result.Summary)
	}
}
