package hook

import (
	"fmt"

	"github.com/jlim/bts/internal/state"
)

type sessionStartHandler struct{}

func NewSessionStartHandler() Handler {
	return &sessionStartHandler{}
}

func (h *sessionStartHandler) EventType() EventType {
	return EventSessionStart
}

func (h *sessionStartHandler) Handle(input *HookInput) (*HookOutput, error) {
	btsRoot, err := state.FindBTSRoot(input.CWD)
	if err != nil {
		// No .bts/ — not a bts project, silently pass
		return &HookOutput{}, nil
	}

	recipe, err := state.GetActiveRecipe(btsRoot)
	if err != nil || recipe == nil {
		return &HookOutput{}, nil
	}

	msg := fmt.Sprintf(
		"[bts] Active recipe: %s \"%s\" (Step: %s, Iteration: %d)\n"+
			"Run /recipe resume to continue, or /recipe cancel to abort.",
		recipe.Type, recipe.Topic, recipe.Phase, recipe.Iteration,
	)

	return &HookOutput{
		HookSpecificOutput: &HookSpecificOutput{
			AdditionalContext: msg,
		},
	}, nil
}
