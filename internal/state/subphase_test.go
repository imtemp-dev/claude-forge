package state

import (
	"os"
	"path/filepath"
	"testing"
)

func setupSubphaseRoot(t *testing.T, recipeID string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(RecipeDir(root, recipeID), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return root
}

func TestDebateRound_SaveAndLoad(t *testing.T) {
	recipeID := "r-001-test"
	root := setupSubphaseRoot(t, recipeID)

	s := &DebateRoundState{
		DebateID:     "d-1",
		Round:        2,
		TotalRounds:  3,
		NextPersona:  "security",
		PendingNotes: []string{"consider token rotation"},
	}
	if err := SaveDebateRound(root, recipeID, s); err != nil {
		t.Fatalf("save: %v", err)
	}
	if s.UpdatedAt == "" {
		t.Error("UpdatedAt should be stamped")
	}

	got, err := LoadDebateRound(root, recipeID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.DebateID != "d-1" || got.Round != 2 || got.TotalRounds != 3 {
		t.Errorf("unexpected state: %+v", got)
	}
	if got.NextPersona != "security" {
		t.Errorf("NextPersona: %q", got.NextPersona)
	}
	if len(got.PendingNotes) != 1 {
		t.Errorf("PendingNotes: %v", got.PendingNotes)
	}

	// Verify path uses recipe dir
	if _, err := os.Stat(filepath.Join(RecipeDir(root, recipeID), "debate-state.json")); err != nil {
		t.Errorf("expected file at recipe dir: %v", err)
	}
}

func TestDebateRound_LoadMissing(t *testing.T) {
	recipeID := "r-001-x"
	root := setupSubphaseRoot(t, recipeID)
	got, err := LoadDebateRound(root, recipeID)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestSimulateProgress_SaveAndLoad(t *testing.T) {
	recipeID := "r-002-sim"
	root := setupSubphaseRoot(t, recipeID)

	s := &SimulateProgressState{
		SimulateID:     "s-42",
		ScenarioIdx:    3,
		TotalScenarios: 8,
		FoundGaps:      2,
	}
	if err := SaveSimulateProgress(root, recipeID, s); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := LoadSimulateProgress(root, recipeID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.SimulateID != "s-42" || got.ScenarioIdx != 3 || got.TotalScenarios != 8 || got.FoundGaps != 2 {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestSimulateProgress_LoadMissing(t *testing.T) {
	recipeID := "r-002-missing"
	root := setupSubphaseRoot(t, recipeID)
	got, err := LoadSimulateProgress(root, recipeID)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}
