package hook

import (
	"github.com/jlim/bts/internal/state"
)

type preCompactHandler struct{}

func NewPreCompactHandler() Handler {
	return &preCompactHandler{}
}

func (h *preCompactHandler) EventType() EventType {
	return EventPreCompact
}

func (h *preCompactHandler) Handle(input *HookInput) (*HookOutput, error) {
	btsRoot, err := state.FindBTSRoot(input.CWD)
	if err != nil {
		return &HookOutput{}, nil
	}

	// Save current recipe state as snapshot before compaction
	recipe, err := state.GetActiveRecipe(btsRoot)
	if err != nil || recipe == nil {
		return &HookOutput{}, nil
	}

	// Re-save to ensure UpdatedAt is current
	if err := state.SaveRecipeState(btsRoot, recipe); err != nil {
		// Non-fatal: log but don't block compaction
		return &HookOutput{}, nil
	}

	return &HookOutput{
		HookSpecificOutput: &HookSpecificOutput{
			AdditionalContext: "[bts] Recipe state saved before compaction.",
		},
	}, nil
}
