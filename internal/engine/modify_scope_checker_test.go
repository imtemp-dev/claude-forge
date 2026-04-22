package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupScope(t *testing.T, finalBody, tasksBody, sourceFile, sourceBody string) (finalPath, tasksPath, projectRoot string) {
	t.Helper()
	projectRoot = t.TempDir()
	finalPath = filepath.Join(projectRoot, "final.md")
	tasksPath = filepath.Join(projectRoot, "tasks.json")
	if err := os.WriteFile(finalPath, []byte(finalBody), 0644); err != nil {
		t.Fatalf("final: %v", err)
	}
	if err := os.WriteFile(tasksPath, []byte(tasksBody), 0644); err != nil {
		t.Fatalf("tasks: %v", err)
	}
	if sourceFile != "" {
		full := filepath.Join(projectRoot, sourceFile)
		_ = os.MkdirAll(filepath.Dir(full), 0755)
		if err := os.WriteFile(full, []byte(sourceBody), 0644); err != nil {
			t.Fatalf("source: %v", err)
		}
	}
	return finalPath, tasksPath, projectRoot
}

// Modify action with scope= in anchor + matching ModifyScope in tasks +
// symbols present in source → no issues.
func TestCheckModifyScope_Aligned(t *testing.T) {
	finalMd := `<!-- task-anchor: src/oauth.ts modify scope=validateToken,refreshSession -->`
	tasksJson := `{
  "tasks": [
    {
      "id": "t-001",
      "file": "src/oauth.ts",
      "action": "modify",
      "status": "done",
      "description": "x",
      "anchor": "src/oauth.ts modify",
      "modify_scope": ["validateToken", "refreshSession"]
    }
  ]
}`
	source := `export function validateToken(x) {}
export function refreshSession(y) {}
`
	finalPath, tasksPath, root := setupScope(t, finalMd, tasksJson, "src/oauth.ts", source)
	issues := CheckModifyScope(finalPath, tasksPath, root)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %v", issues)
	}
}

// No scope in either anchor or ModifyScope → modify_scope_required.
func TestCheckModifyScope_RequiredMissing(t *testing.T) {
	finalMd := `<!-- task-anchor: src/a.ts modify -->`
	tasksJson := `{
  "tasks": [
    {"id":"t-001","file":"src/a.ts","action":"modify","status":"done","description":"x","anchor":"src/a.ts modify"}
  ]
}`
	finalPath, tasksPath, root := setupScope(t, finalMd, tasksJson, "", "")
	issues := CheckModifyScope(finalPath, tasksPath, root)
	if len(issues) != 1 {
		t.Fatalf("expected 1 required issue, got %v", issues)
	}
	if issues[0].Severity != "major" || !strings.Contains(issues[0].Claim, "modify_scope_required") {
		t.Errorf("unexpected: %+v", issues[0])
	}
}

// scope= in anchor differs from ModifyScope → scope_mismatch.
func TestCheckModifyScope_Mismatch(t *testing.T) {
	finalMd := `<!-- task-anchor: src/a.ts modify scope=alpha,beta -->`
	tasksJson := `{
  "tasks": [
    {
      "id": "t-001", "file": "src/a.ts", "action": "modify", "status": "done", "description": "x",
      "anchor": "src/a.ts modify", "modify_scope": ["alpha", "gamma"]
    }
  ]
}`
	finalPath, tasksPath, root := setupScope(t, finalMd, tasksJson, "", "")
	issues := CheckModifyScope(finalPath, tasksPath, root)
	if len(issues) < 1 {
		t.Fatalf("expected mismatch, got none")
	}
	found := false
	for _, i := range issues {
		if strings.Contains(i.Claim, "scope_mismatch") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected scope_mismatch, got %v", issues)
	}
}

// Scope symbol absent from source file → scope_symbol_missing critical.
func TestCheckModifyScope_SymbolMissingCritical(t *testing.T) {
	finalMd := `<!-- task-anchor: src/a.ts modify scope=doesNotExist -->`
	tasksJson := `{
  "tasks": [
    {
      "id": "t-001", "file": "src/a.ts", "action": "modify", "status": "done", "description": "x",
      "anchor": "src/a.ts modify", "modify_scope": ["doesNotExist"]
    }
  ]
}`
	source := `function unrelated() {}
function alsoUnrelated() {}
`
	finalPath, tasksPath, root := setupScope(t, finalMd, tasksJson, "src/a.ts", source)
	issues := CheckModifyScope(finalPath, tasksPath, root)
	var critical *Issue
	for i := range issues {
		if issues[i].Severity == "critical" {
			critical = &issues[i]
		}
	}
	if critical == nil {
		t.Fatalf("expected critical scope_symbol_missing, got %v", issues)
	}
	if !strings.Contains(critical.Claim, "scope_symbol_missing") ||
		!strings.Contains(critical.Claim, "doesNotExist") {
		t.Errorf("wrong claim: %s", critical.Claim)
	}
}

// create action should be ignored (scope rules apply only to modify).
func TestCheckModifyScope_CreateActionIgnored(t *testing.T) {
	finalMd := `<!-- task-anchor: src/new.ts create -->`
	tasksJson := `{
  "tasks": [
    {"id":"t-001","file":"src/new.ts","action":"create","status":"done","description":"x","anchor":"src/new.ts create"}
  ]
}`
	finalPath, tasksPath, root := setupScope(t, finalMd, tasksJson, "", "")
	issues := CheckModifyScope(finalPath, tasksPath, root)
	if len(issues) != 0 {
		t.Fatalf("create tasks should not trigger modify_scope checks, got %v", issues)
	}
}

// projectRoot empty → symbol check skipped; only scope_required /
// mismatch layers run. A task with missing symbol but no root given
// should not produce a scope_symbol_missing finding.
func TestCheckModifyScope_EmptyRootSkipsSymbolCheck(t *testing.T) {
	finalMd := `<!-- task-anchor: src/a.ts modify scope=imaginary -->`
	tasksJson := `{
  "tasks": [
    {"id":"t-001","file":"src/a.ts","action":"modify","status":"done","description":"x","anchor":"src/a.ts modify","modify_scope":["imaginary"]}
  ]
}`
	finalPath, tasksPath, _ := setupScope(t, finalMd, tasksJson, "", "")
	issues := CheckModifyScope(finalPath, tasksPath, "")
	for _, i := range issues {
		if strings.Contains(i.Claim, "scope_symbol_missing") {
			t.Errorf("empty root should skip symbol check, got %+v", i)
		}
	}
}
