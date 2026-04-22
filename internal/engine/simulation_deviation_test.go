package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupSims(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	simsDir := filepath.Join(dir, "simulations")
	if err := os.MkdirAll(simsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(simsDir, name), []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// Structured grammar parses verbatim.
func TestExtractSimulationDeviations_Structured(t *testing.T) {
	dir := setupSims(t, map[string]string{
		"002-code.md": `# sim

Some prose.

DEVIATION {id=sim-002.s1} {driver=simulate} {severity=major}: snap order differs.
DEVIATION {id=sim-002.s2} {driver=simulate} {severity=minor}: minor gap.
`,
	})
	sims, err := ExtractSimulationDeviations(dir)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(sims) != 2 {
		t.Fatalf("want 2, got %d: %+v", len(sims), sims)
	}
	if sims[0].ID != "sim-002.s1" || sims[0].Severity != "major" {
		t.Errorf("wrong parse: %+v", sims[0])
	}
	if sims[1].Severity != "minor" {
		t.Errorf("want minor, got %s", sims[1].Severity)
	}
}

// Legacy bullet form with explicit DEVIATION-N id gets a sim-derived id.
func TestExtractSimulationDeviations_LegacyBullet(t *testing.T) {
	dir := setupSims(t, map[string]string{
		"001-scenarios.md": `# sim

- [DEVIATION-001] legacy bullet form.
- [DEVIATION-002] another.
`,
	})
	sims, err := ExtractSimulationDeviations(dir)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(sims) != 2 {
		t.Fatalf("want 2, got %d: %+v", len(sims), sims)
	}
	if sims[0].ID != "sim-001-scenarios.d1" {
		t.Errorf("unexpected id: %s", sims[0].ID)
	}
	if sims[0].Severity != "major" {
		t.Errorf("legacy severity default is major, got %s", sims[0].Severity)
	}
}

// Legacy bare-line form uses .s counter.
func TestExtractSimulationDeviations_LegacyBareLine(t *testing.T) {
	dir := setupSims(t, map[string]string{
		"003-code.md": `DEVIATION: first one
DEVIATION: second one
`,
	})
	sims, err := ExtractSimulationDeviations(dir)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(sims) != 2 {
		t.Fatalf("want 2, got %d", len(sims))
	}
	if sims[0].ID != "sim-003-code.s1" || sims[1].ID != "sim-003-code.s2" {
		t.Errorf("unexpected ids: %+v", sims)
	}
}

// No simulations dir → no entries, no error.
func TestExtractSimulationDeviations_NoDir(t *testing.T) {
	dir := t.TempDir()
	sims, err := ExtractSimulationDeviations(dir)
	if err != nil || len(sims) != 0 {
		t.Errorf("want empty no-error, got err=%v sims=%v", err, sims)
	}
}

// Consumption check: deviation.md cites simulate:{id} → no finding.
func TestCheckSimDeviationConsumption_Covered(t *testing.T) {
	dir := setupSims(t, map[string]string{
		"002-code.md": `DEVIATION {id=sim-002.s1} {driver=simulate} {severity=major}: x.
`,
	})
	deviationMd := `## Deviations
| ID | Item | Spec Says | Code Has | Driver | Severity | Resolution |
|----|------|-----------|----------|--------|----------|------------|
| D-001 | x | y | z | code-diff,simulate:sim-002.s1 | major | synced |
`
	_ = os.WriteFile(filepath.Join(dir, "deviation.md"), []byte(deviationMd), 0644)
	issues := CheckSimDeviationConsumption(dir)
	if len(issues) != 0 {
		t.Fatalf("expected covered, got %v", issues)
	}
}

// Consumption check: sim id missing from deviation.md → major finding.
func TestCheckSimDeviationConsumption_Unconsumed(t *testing.T) {
	dir := setupSims(t, map[string]string{
		"002-code.md": `DEVIATION {id=sim-002.s1} {driver=simulate} {severity=major}: x.
DEVIATION {id=sim-002.s2} {driver=simulate} {severity=minor}: y.
`,
	})
	deviationMd := `## Deviations
| ID | Item | Spec Says | Code Has | Driver | Severity | Resolution |
|----|------|-----------|----------|--------|----------|------------|
| D-001 | only first | a | b | simulate:sim-002.s1 | major | synced |
`
	_ = os.WriteFile(filepath.Join(dir, "deviation.md"), []byte(deviationMd), 0644)
	issues := CheckSimDeviationConsumption(dir)
	if len(issues) != 1 {
		t.Fatalf("want 1 unconsumed (s2), got %v", issues)
	}
	if !strings.Contains(issues[0].Claim, "sim-002.s2") {
		t.Errorf("wrong id cited: %s", issues[0].Claim)
	}
}
