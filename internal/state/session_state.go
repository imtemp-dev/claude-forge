package state

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

// SessionState captures a non-recipe snapshot for context recovery when
// there is no active recipe at compaction time. It stores only runtime
// breadcrumbs (recent tools, open files, latest plan file); the user's
// intent line is intentionally left empty in this first version because
// PreCompact has no reliable source for it.
type SessionState struct {
	Intent      string           `json:"intent,omitempty"`
	RecentTools []ToolTraceEntry `json:"recent_tools,omitempty"`
	OpenFiles   []string         `json:"open_files,omitempty"`
	PendingPlan string           `json:"pending_plan,omitempty"`
	SavedAt     string           `json:"saved_at"`
}

// SessionStatePath returns the path to the session state file.
func SessionStatePath(root string) string {
	return filepath.Join(LocalPath(root), "session-state.json")
}

// SaveSessionState writes the snapshot atomically.
func SaveSessionState(root string, s *SessionState) error {
	if s == nil {
		return nil
	}
	s.SavedAt = time.Now().UTC().Format(time.RFC3339)
	return WriteJSON(SessionStatePath(root), s)
}

// LoadSessionState reads the snapshot file. Returns (nil, err) if not found.
func LoadSessionState(root string) (*SessionState, error) {
	var s SessionState
	if err := ReadJSON(SessionStatePath(root), &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// BuildSessionState aggregates tool-trace breadcrumbs and plan hints.
// Returns nil if there is nothing worth saving.
func BuildSessionState(root string) (*SessionState, error) {
	trace, _ := TailToolTrace(root, 8)
	openFiles := openFilesFromTrace(trace)
	plan := latestPlanFile()

	if len(trace) == 0 && plan == "" {
		return nil, nil
	}

	s := &SessionState{
		OpenFiles:   openFiles,
		PendingPlan: plan,
	}
	// Copy to value slice for JSON determinism
	for _, e := range trace {
		if e != nil {
			s.RecentTools = append(s.RecentTools, *e)
		}
	}
	return s, nil
}

// openFilesFromTrace extracts a unique ordered list of file paths touched
// by Read, Edit, or Write tools in the given trace.
func openFilesFromTrace(trace []*ToolTraceEntry) []string {
	if len(trace) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var files []string
	for _, e := range trace {
		if e == nil || e.File == "" {
			continue
		}
		switch e.ToolName {
		case "Read", "Edit", "Write", "NotebookEdit":
			if !seen[e.File] {
				seen[e.File] = true
				files = append(files, e.File)
			}
		}
	}
	return files
}

// latestPlanFile returns the most recently modified plan file in
// ~/.claude/plans/ if any exists. Empty string otherwise.
func latestPlanFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	plansDir := filepath.Join(home, ".claude", "plans")
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return ""
	}
	type planEntry struct {
		path    string
		modTime time.Time
	}
	var plans []planEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		plans = append(plans, planEntry{filepath.Join(plansDir, e.Name()), info.ModTime()})
	}
	if len(plans) == 0 {
		return ""
	}
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].modTime.After(plans[j].modTime)
	})
	return plans[0].path
}
