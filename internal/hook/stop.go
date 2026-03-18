package hook

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jlim/bts/internal/state"
)

type stopHandler struct{}

func NewStopHandler() Handler {
	return &stopHandler{}
}

func (h *stopHandler) EventType() EventType {
	return EventStop
}

func (h *stopHandler) Handle(input *HookInput) (*HookOutput, error) {
	btsRoot, err := state.FindBTSRoot(input.CWD)
	if err != nil {
		return &HookOutput{}, nil
	}

	recipe, err := state.GetActiveRecipe(btsRoot)
	if err != nil || recipe == nil {
		return &HookOutput{}, nil
	}

	// Check if DONE marker is present in the stop content
	if !strings.Contains(input.StopHookContent, "<bts>DONE</bts>") {
		// No DONE marker — just remind about active recipe
		return &HookOutput{
			HookSpecificOutput: &HookSpecificOutput{
				AdditionalContext: fmt.Sprintf(
					"[bts] Recipe \"%s\" is still active (Step: %s). Continue or use /recipe cancel.",
					recipe.Topic, recipe.Phase,
				),
			},
		}, nil
	}

	// DONE marker found — verify the last iteration passed
	logPath := filepath.Join(state.RecipeDir(btsRoot, recipe.ID), "verify-log.jsonl")
	lastEntry, err := readLastVerifyEntry(logPath)
	if err != nil {
		// No verify log — block
		return blockOutput("No verification log found. Run verification before completing."), nil
	}

	if lastEntry.Critical > 0 || lastEntry.Major > 0 {
		return blockOutput(fmt.Sprintf(
			"Verification not passed: %d critical, %d major errors remain. Fix and re-verify.",
			lastEntry.Critical, lastEntry.Major,
		)), nil
	}

	// All clear — allow completion
	recipe.Phase = "finalize"
	_ = state.SaveRecipeState(btsRoot, recipe)

	return &HookOutput{}, nil
}

func blockOutput(reason string) *HookOutput {
	return &HookOutput{
		Decision: "block",
		Reason:   reason,
	}
}

func readLastVerifyEntry(path string) (*state.VerifyLogEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var last state.VerifyLogEntry
	found := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry state.VerifyLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		last = entry
		found = true
	}

	if !found {
		return nil, fmt.Errorf("empty verify log")
	}

	return &last, nil
}
