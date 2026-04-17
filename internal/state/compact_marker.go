package state

import (
	"os"
	"path/filepath"
	"time"
)

// CompactMarker is a transient flag written by PreCompact and consumed
// by SessionStart to deterministically detect a compaction event.
type CompactMarker struct {
	SessionID string `json:"session_id,omitempty"`
	RecipeID  string `json:"recipe_id,omitempty"`
	Phase     string `json:"phase,omitempty"`
	CreatedAt string `json:"created_at"`
}

// CompactMarkerPath returns the path to the compact marker file.
func CompactMarkerPath(root string) string {
	return filepath.Join(LocalPath(root), "compact-pending.json")
}

// WriteCompactMarker writes (or overwrites) the marker file atomically.
// Always stamps CreatedAt to now.
func WriteCompactMarker(root string, m *CompactMarker) error {
	if m == nil {
		m = &CompactMarker{}
	}
	m.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	return WriteJSON(CompactMarkerPath(root), m)
}

// ConsumeCompactMarker reads and deletes the marker.
// Returns (nil, nil) if no marker exists. Returns (m, nil) and removes the
// file if a marker was present. If the file is malformed, it is deleted
// and (nil, nil) is returned so the caller can fall back gracefully.
func ConsumeCompactMarker(root string) (*CompactMarker, error) {
	path := CompactMarkerPath(root)
	var m CompactMarker
	if err := ReadJSON(path, &m); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		// Malformed — remove and treat as absent
		_ = os.Remove(path)
		return nil, nil
	}
	_ = os.Remove(path)
	return &m, nil
}
