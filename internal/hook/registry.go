package hook

import "fmt"

// Registry maps event types to handlers.
type Registry struct {
	handlers map[EventType]Handler
}

// NewRegistry creates a registry with all handlers.
func NewRegistry(handlers ...Handler) *Registry {
	r := &Registry{
		handlers: make(map[EventType]Handler),
	}
	for _, h := range handlers {
		r.handlers[h.EventType()] = h
	}
	return r
}

// Dispatch finds and executes the handler for the given event.
func (r *Registry) Dispatch(input *HookInput) (*HookOutput, error) {
	eventType := EventType(input.HookEventName)

	handler, ok := r.handlers[eventType]
	if !ok {
		// Unknown event — exit silently (don't break Claude Code)
		return &HookOutput{}, nil
	}

	output, err := handler.Handle(input)
	if err != nil {
		return nil, fmt.Errorf("handler %s: %w", eventType, err)
	}

	if output == nil {
		output = &HookOutput{}
	}

	return output, nil
}
