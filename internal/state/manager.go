package state

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindRoot searches for .forge/ directory starting from cwd upward.
// Automatically migrates old .forge/state/ layout to specs/ + local/ if needed.
func FindRoot(cwd string) (string, error) {
	dir := cwd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".forge")); err == nil {
			_ = maybeMigrate(dir)
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf(".forge/ not found from %s", cwd)
}

// SpecsPath returns the path to the specs directory (git tracked).
func SpecsPath(root string) string {
	return filepath.Join(root, ".forge", "specs")
}

// LocalPath returns the path to the local runtime directory (gitignored).
func LocalPath(root string) string {
	return filepath.Join(root, ".forge", "local")
}

// maybeMigrate migrates old .forge/state/ layout to .forge/specs/ + .forge/local/.
func maybeMigrate(root string) error {
	stateDir := filepath.Join(root, ".forge", "state")
	specsDir := filepath.Join(root, ".forge", "specs")
	localDir := filepath.Join(root, ".forge", "local")

	// Only migrate if old state/ directory still exists
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		return nil
	}

	// Check if there's anything left to migrate
	entries, err := os.ReadDir(stateDir)
	if err != nil || len(entries) == 0 {
		_ = os.Remove(stateDir) // clean up empty dir
		return nil
	}

	fmt.Fprintf(os.Stderr, "[forge] Migrating .forge/state/ → .forge/specs/ + .forge/local/\n")

	_ = os.MkdirAll(specsDir, 0755)
	_ = os.MkdirAll(localDir, 0755)

	// Move spec files to specs/
	for _, f := range []string{"vision.md", "roadmap.md", "project-status.md", "project-map.md"} {
		moveIfExists(filepath.Join(stateDir, f), filepath.Join(specsDir, f))
	}

	// Move spec directories to specs/
	for _, d := range []string{"recipes", "debates", "layers"} {
		moveIfExists(filepath.Join(stateDir, d), filepath.Join(specsDir, d))
	}

	// Move local files to local/
	for _, f := range []string{"metrics.jsonl", "work-state.json", "active-agent.json", ".metrics-token-ts"} {
		moveIfExists(filepath.Join(stateDir, f), filepath.Join(localDir, f))
	}

	// Move per-recipe metrics.jsonl to local/recipes/{id}/
	recipesDir := filepath.Join(specsDir, "recipes")
	if entries, err := os.ReadDir(recipesDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			src := filepath.Join(recipesDir, entry.Name(), "metrics.jsonl")
			if _, err := os.Stat(src); err == nil {
				dstDir := filepath.Join(localDir, "recipes", entry.Name())
				_ = os.MkdirAll(dstDir, 0755)
				moveIfExists(src, filepath.Join(dstDir, "metrics.jsonl"))
			}
		}
	}

	// Remove state/ if empty
	_ = os.Remove(stateDir)

	// Update .gitignore
	updateGitignore(root)

	return nil
}

func moveIfExists(src, dst string) {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return
	}
	_ = os.MkdirAll(filepath.Dir(dst), 0755)
	_ = os.Rename(src, dst)
}

func updateGitignore(root string) {
	path := filepath.Join(root, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	content := string(data)
	if strings.Contains(content, ".forge/local/") {
		return // already updated
	}

	// Replace old pattern or append new
	if strings.Contains(content, ".forge/state/") {
		content = strings.Replace(content, ".forge/state/", ".forge/local/", 1)
	} else {
		// Read lines, replace the comment too if present
		var lines []string
		scanner := bufio.NewScanner(strings.NewReader(content))
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		lines = append(lines, "# forge local data (not committed)")
		lines = append(lines, ".forge/local/")
		content = strings.Join(lines, "\n") + "\n"
	}

	_ = os.WriteFile(path, []byte(content), 0644)
}

// ReadJSON reads a JSON file into the target struct.
func ReadJSON(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// WriteJSON writes a struct to a JSON file atomically (temp + rename).
func WriteJSON(path string, data interface{}) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".forge-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(bytes); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp: %w", err)
	}
	tmp.Close()

	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// AppendJSONL appends a JSON line to a JSONL file.
func AppendJSONL(path string, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", bytes)
	return err
}
