package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDraft(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "draft.md")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestCheckInterfaceJustification_GoSingleImplFlagged(t *testing.T) {
	path := writeDraft(t, "```go\ntype Store interface {\n    Get() string\n}\n\n// implements Store\ntype MemStore struct{}\n```\n")
	issues := CheckInterfaceJustification(path)
	if len(issues) != 1 {
		t.Fatalf("want 1 issue for single impl, got %d: %v", len(issues), issues)
	}
	if !strings.Contains(issues[0].Claim, "single_impl_interface: Store") {
		t.Errorf("wrong claim: %s", issues[0].Claim)
	}
}

func TestCheckInterfaceJustification_GoYagniJustifiedPasses(t *testing.T) {
	path := writeDraft(t, "```go\n// YAGNI-justified: external consumers mock this in tests.\ntype Store interface {\n    Get() string\n}\n\n// implements Store\ntype MemStore struct{}\n```\n")
	issues := CheckInterfaceJustification(path)
	if len(issues) != 0 {
		t.Fatalf("justified interface should pass, got %d issues: %v", len(issues), issues)
	}
}

func TestCheckInterfaceJustification_GoMultipleImplsPass(t *testing.T) {
	path := writeDraft(t, "```go\ntype Store interface {\n    Get() string\n}\n\n// implements Store\ntype MemStore struct{}\n\n// implements Store\ntype FileStore struct{}\n```\n")
	issues := CheckInterfaceJustification(path)
	if len(issues) != 0 {
		t.Fatalf("2 impls should pass, got %v", issues)
	}
}

func TestCheckInterfaceJustification_TSImplementsFlagged(t *testing.T) {
	path := writeDraft(t, "```ts\nexport interface Shape {\n  area(): number;\n}\n\nclass Circle implements Shape {\n  area() { return 0; }\n}\n```\n")
	issues := CheckInterfaceJustification(path)
	if len(issues) != 1 {
		t.Fatalf("want 1 issue, got %d", len(issues))
	}
}

func TestCheckInterfaceJustification_PyProtocolFlagged(t *testing.T) {
	path := writeDraft(t, "```python\nclass Repo(Protocol):\n    def get(self) -> str: ...\n\nclass InMemoryRepo(Repo):\n    def get(self): return ''\n```\n")
	issues := CheckInterfaceJustification(path)
	if len(issues) != 1 {
		t.Fatalf("want 1 issue, got %d", len(issues))
	}
}

func TestCheckInterfaceJustification_InlineYagniComment(t *testing.T) {
	// YAGNI marker on the declaration line itself
	path := writeDraft(t, "```go\ntype Store interface { // YAGNI-justified: protocol boundary\n    Get() string\n}\n// implements Store\ntype MemStore struct{}\n```\n")
	issues := CheckInterfaceJustification(path)
	if len(issues) != 0 {
		t.Fatalf("inline YAGNI should pass, got %v", issues)
	}
}
