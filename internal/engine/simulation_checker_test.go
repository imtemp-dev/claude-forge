package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSim(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "001-scenarios.md")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestCheckSimulationScenarios_MeetsCrossBoundaryRatio(t *testing.T) {
	// 2 out of 5 are cross-boundary = 40% > 30%.
	body := `# Scenarios

## Scenario 1 [single-axis: Drag]
Body.

## Scenario 2 [cross-boundary: axes=Drag,Arrangement]
Body.

## Scenario 3 [single-axis: Animation]
Body.

## Scenario 4 [cross-boundary: axes=Arrangement,Validation]
Body.

## Scenario 5 [single-axis: Drag]
Body.
`
	path := writeSim(t, body)
	issues := CheckSimulationScenarios(path, DefaultCrossBoundaryRatio)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d: %v", len(issues), issues)
	}
}

func TestCheckSimulationScenarios_BelowThreshold(t *testing.T) {
	// 1 out of 5 = 20% < 30%.
	body := `## Scenario 1 [cross-boundary: axes=A,B]
## Scenario 2 [single-axis: A]
## Scenario 3 [single-axis: A]
## Scenario 4 [single-axis: B]
## Scenario 5 [single-axis: C]
`
	path := writeSim(t, body)
	issues := CheckSimulationScenarios(path, DefaultCrossBoundaryRatio)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d: %v", len(issues), issues)
	}
	if issues[0].Severity != "critical" {
		t.Errorf("want critical, got %s", issues[0].Severity)
	}
	if !strings.Contains(issues[0].Claim, "insufficient_cross_boundary_coverage") {
		t.Errorf("wrong claim: %s", issues[0].Claim)
	}
}

func TestCheckSimulationScenarios_UntaggedFlagged(t *testing.T) {
	body := `## Scenario 1 [cross-boundary: axes=A,B]
## Scenario 2
No tag here.
`
	path := writeSim(t, body)
	issues := CheckSimulationScenarios(path, DefaultCrossBoundaryRatio)
	// "untagged_scenarios" major expected. Depending on ratio, no critical.
	var untaggedSeen bool
	for _, i := range issues {
		if strings.Contains(i.Claim, "untagged_scenarios") {
			untaggedSeen = true
		}
	}
	if !untaggedSeen {
		t.Fatalf("expected untagged_scenarios major, got %v", issues)
	}
}

// Illegal-cell scenarios are themselves cross-boundary by definition;
// they count toward the ratio.
func TestCheckSimulationScenarios_IllegalCellCountsTowardRatio(t *testing.T) {
	body := `## Scenario 1 [single-axis: A]
## Scenario 2 [single-axis: A]
## Scenario 3 [illegal-cell: drag×slot=dropping,filled]
`
	path := writeSim(t, body)
	// 1 illegal / 3 = 33% > 30%
	issues := CheckSimulationScenarios(path, DefaultCrossBoundaryRatio)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

// Full integration: domain.md with 2 illegal cells, simulations cover 1.
func TestCheckIllegalCellCoverage_UncoveredCellCritical(t *testing.T) {
	dir := t.TempDir()
	domain := filepath.Join(dir, "domain.md")
	domainBody := `# Domain

## 4. Combinatorial State Space

| col1 | col2 |
|------|------|
| A    | ILLEGAL: double-fill |
| B    | ILLEGAL: mid-snap-back |
`
	if err := os.WriteFile(domain, []byte(domainBody), 0644); err != nil {
		t.Fatalf("write domain: %v", err)
	}
	simsDir := filepath.Join(dir, "simulations")
	if err := os.MkdirAll(simsDir, 0755); err != nil {
		t.Fatalf("mkdir sims: %v", err)
	}
	simBody := `## Scenario 1 [illegal-cell: double-fill]`
	if err := os.WriteFile(filepath.Join(simsDir, "001.md"), []byte(simBody), 0644); err != nil {
		t.Fatalf("write sim: %v", err)
	}

	issues := CheckIllegalCellCoverage(domain, dir)
	if len(issues) != 1 {
		t.Fatalf("want 1 issue (mid-snap-back uncovered), got %d: %v", len(issues), issues)
	}
	if issues[0].Severity != "critical" {
		t.Errorf("want critical, got %s", issues[0].Severity)
	}
	if !strings.Contains(issues[0].Claim, "mid-snap-back") {
		t.Errorf("claim should mention uncovered cell, got %s", issues[0].Claim)
	}
}

func TestCheckIllegalCellCoverage_AllCellsCoveredPasses(t *testing.T) {
	dir := t.TempDir()
	domain := filepath.Join(dir, "domain.md")
	_ = os.WriteFile(domain, []byte(`## 4. Combinatorial State Space

| x | ILLEGAL: A |
| y | ILLEGAL: B |
`), 0644)
	simsDir := filepath.Join(dir, "simulations")
	_ = os.MkdirAll(simsDir, 0755)
	_ = os.WriteFile(filepath.Join(simsDir, "001.md"),
		[]byte(`## S1 [illegal-cell: A]
## S2 [illegal-cell: B]
`), 0644)

	issues := CheckIllegalCellCoverage(domain, dir)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %v", issues)
	}
}
