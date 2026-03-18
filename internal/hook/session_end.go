package hook

import (
	"github.com/jlim/bts/internal/state"
)

type sessionEndHandler struct{}

func NewSessionEndHandler() Handler {
	return &sessionEndHandler{}
}

func (h *sessionEndHandler) EventType() EventType {
	return EventSessionEnd
}

func (h *sessionEndHandler) Handle(input *HookInput) (*HookOutput, error) {
	btsRoot, err := state.FindBTSRoot(input.CWD)
	if err != nil {
		return &HookOutput{}, nil
	}

	// Save any active recipe state
	recipe, err := state.GetActiveRecipe(btsRoot)
	if err != nil || recipe == nil {
		return &HookOutput{}, nil
	}

	_ = state.SaveRecipeState(btsRoot, recipe)

	return &HookOutput{}, nil
}
