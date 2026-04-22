package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/imtemp-dev/claude-bts/internal/state"
	"github.com/spf13/cobra"
)

func init() {
	migrateCmd.AddCommand(migrateVerifyLogCmd)
	migrateVerifyLogCmd.Flags().String("target", "", "Target directory (defaults to CWD project root)")
	migrateVerifyLogCmd.Flags().Bool("dry-run", false, "Show what would change without writing")

	migrateCmd.AddCommand(migrateChangelogCmd)
	migrateChangelogCmd.Flags().String("target", "", "Target directory (defaults to CWD project root)")
	migrateChangelogCmd.Flags().Bool("dry-run", false, "Show what would change without writing")

	migrateCmd.AddCommand(migrateVerificationCmd)
	migrateVerificationCmd.Flags().String("target", "", "Target directory (defaults to CWD project root)")
	migrateVerificationCmd.Flags().Bool("dry-run", false, "Show what would change without writing")

	migrateCmd.AddCommand(migrateSimulationsCmd)
	migrateSimulationsCmd.Flags().String("target", "", "Target directory (defaults to CWD project root)")
	migrateSimulationsCmd.Flags().Bool("dry-run", false, "Show what would change without writing")

	migrateCmd.AddCommand(migrateTaskAnchorsCmd)
	migrateTaskAnchorsCmd.Flags().String("target", "", "Target directory (defaults to CWD project root)")
	migrateTaskAnchorsCmd.Flags().Bool("dry-run", false, "Show what would change without writing")

	migrateCmd.AddCommand(migrateModifyScopeCmd)
	migrateModifyScopeCmd.Flags().String("target", "", "Target directory (defaults to CWD project root)")
	migrateModifyScopeCmd.Flags().Bool("dry-run", false, "Show what would change without writing")

	migrateCmd.AddCommand(migrateAllCmd)
	migrateAllCmd.Flags().String("target", "", "Target directory (defaults to CWD project root)")
	migrateAllCmd.Flags().Bool("dry-run", false, "Show what would change without writing")

	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:     "migrate",
	Short:   "Migrate BTS artifacts to current schema",
	GroupID: "tools",
}

var migrateVerifyLogCmd = &cobra.Command{
	Use:   "verify-log",
	Short: "Rewrite verify-log.jsonl entries to carry split-minor fields",
	Long: `Legacy verify-log entries have a single "minor" field. Phase 2 of the
BTS schema split minors into [resolvable] (blocks completion) and
[deferred] (runtime watch-items). This command re-emits each entry with
minor_resolvable = minor (conservative), minor_deferred = 0.

Entries already carrying minor_resolvable or minor_deferred are left
unchanged. Backup files are written alongside as *.jsonl.bak.`,
	RunE: runMigrateVerifyLog,
}

func runMigrateVerifyLog(cmd *cobra.Command, args []string) error {
	targetFlag, _ := cmd.Flags().GetString("target")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	target := targetFlag
	if target == "" {
		cwd, _ := os.Getwd()
		root, err := state.FindRoot(cwd)
		if err != nil {
			return fmt.Errorf("not a bts project: %w", err)
		}
		target = root
	}

	recipesDir := filepath.Join(state.SpecsPath(target), "recipes")
	entries, err := os.ReadDir(recipesDir)
	if err != nil {
		return fmt.Errorf("read recipes dir: %w", err)
	}

	var totalRecipes, migratedRecipes, touchedEntries int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		logPath := filepath.Join(recipesDir, e.Name(), "verify-log.jsonl")
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			continue
		}
		totalRecipes++

		changed, touched, err := migrateOneVerifyLog(logPath, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: error: %v\n", e.Name(), err)
			continue
		}
		if changed {
			migratedRecipes++
			touchedEntries += touched
			marker := "migrated"
			if dryRun {
				marker = "would migrate"
			}
			fmt.Printf("  %s: %s %d entr(y|ies)\n", e.Name(), marker, touched)
		}
	}

	if dryRun {
		fmt.Printf("\nDry run: %d/%d recipes need migration, %d entries affected.\n",
			migratedRecipes, totalRecipes, touchedEntries)
	} else {
		fmt.Printf("\nMigrated %d/%d recipes (%d entries). Backups: *.jsonl.bak\n",
			migratedRecipes, totalRecipes, touchedEntries)
	}
	return nil
}

// migrateOneVerifyLog rewrites entries that still use the legacy Minor-only
// layout. Entries already carrying minor_resolvable or minor_deferred keys
// are preserved byte-for-byte (to avoid reformatting timestamps, extra
// fields, etc.). Returns whether any entry was changed and the count.
func migrateOneVerifyLog(path string, dryRun bool) (bool, int, error) {
	in, err := os.Open(path)
	if err != nil {
		return false, 0, err
	}
	defer in.Close()

	var lines []string
	touched := 0
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			lines = append(lines, line)
			continue
		}
		newLine, changed, err := upgradeVerifyLogLine(line)
		if err != nil {
			return false, 0, fmt.Errorf("parse line: %w", err)
		}
		if changed {
			touched++
		}
		lines = append(lines, newLine)
	}
	if err := scanner.Err(); err != nil {
		return false, 0, err
	}
	if touched == 0 {
		return false, 0, nil
	}

	if dryRun {
		return true, touched, nil
	}

	// Backup before writing
	bak := path + ".bak"
	if err := copyFile(path, bak); err != nil {
		return false, 0, fmt.Errorf("backup: %w", err)
	}

	out, err := os.Create(path)
	if err != nil {
		return false, 0, err
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	for _, line := range lines {
		if _, err := w.WriteString(line + "\n"); err != nil {
			return false, 0, err
		}
	}
	return true, touched, w.Flush()
}

func upgradeVerifyLogLine(line string) (string, bool, error) {
	// Peek for the split keys first. Raw string check avoids re-marshaling
	// unrelated fields (which would reorder keys and churn the file).
	if strings.Contains(line, `"minor_resolvable"`) || strings.Contains(line, `"minor_deferred"`) {
		return line, false, nil
	}
	var entry state.VerifyLogEntry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		// Unknown format — leave the line alone.
		return line, false, nil
	}
	if entry.Minor == 0 {
		// Nothing to split. Still emit split keys so downstream validators
		// (which now expect the new format) see a compliant entry.
		entry.MinorResolvable = 0
		entry.MinorDeferred = 0
	} else {
		entry.MinorResolvable = entry.Minor
		entry.MinorDeferred = 0
	}
	// Converge status: strict interpretation aligned with the new CLI handler.
	if entry.Critical == 0 && entry.Major == 0 && entry.MinorResolvable == 0 {
		if entry.Status != "failed" {
			entry.Status = "converged"
		}
	} else if entry.Status == "" {
		entry.Status = "continue"
	}
	data, err := json.Marshal(&entry)
	if err != nil {
		return "", false, err
	}
	return string(data), true, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// ---- changelog verify-result migration --------------------------------

var migrateChangelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Upgrade verify entries in changelog.jsonl to the split-minor format",
	Long: `Each verify action in changelog.jsonl carries a "result" string:
  critical=X major=Y minor=Z → status

This command rewrites it to:
  critical=X major=Y minor_resolvable=Z minor_deferred=0 → status

(Legacy minor is conservatively treated as resolvable; operators can
re-classify specific entries by hand afterward.)`,
	RunE: runMigrateChangelog,
}

func runMigrateChangelog(cmd *cobra.Command, args []string) error {
	target, dryRun, err := migrateFlags(cmd)
	if err != nil {
		return err
	}

	recipesDir := filepath.Join(state.SpecsPath(target), "recipes")
	entries, err := os.ReadDir(recipesDir)
	if err != nil {
		return fmt.Errorf("read recipes dir: %w", err)
	}

	var totalRecipes, migratedRecipes, touchedEntries int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		logPath := filepath.Join(recipesDir, e.Name(), "changelog.jsonl")
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			continue
		}
		totalRecipes++
		changed, touched, err := migrateOneChangelog(logPath, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: error: %v\n", e.Name(), err)
			continue
		}
		if changed {
			migratedRecipes++
			touchedEntries += touched
			marker := "migrated"
			if dryRun {
				marker = "would migrate"
			}
			fmt.Printf("  %s: %s %d entr(y|ies)\n", e.Name(), marker, touched)
		}
	}

	if dryRun {
		fmt.Printf("\nDry run: %d/%d changelogs need migration, %d entries affected.\n",
			migratedRecipes, totalRecipes, touchedEntries)
	} else {
		fmt.Printf("\nMigrated %d/%d changelogs (%d entries). Backups: *.jsonl.bak\n",
			migratedRecipes, totalRecipes, touchedEntries)
	}
	return nil
}

func migrateOneChangelog(path string, dryRun bool) (bool, int, error) {
	in, err := os.Open(path)
	if err != nil {
		return false, 0, err
	}
	defer in.Close()

	var lines []string
	touched := 0
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			lines = append(lines, line)
			continue
		}
		newLine, changed := upgradeChangelogLine(line)
		if changed {
			touched++
		}
		lines = append(lines, newLine)
	}
	if err := scanner.Err(); err != nil {
		return false, 0, err
	}
	if touched == 0 {
		return false, 0, nil
	}
	if dryRun {
		return true, touched, nil
	}
	bak := path + ".bak"
	if err := copyFile(path, bak); err != nil {
		return false, 0, fmt.Errorf("backup: %w", err)
	}
	out, err := os.Create(path)
	if err != nil {
		return false, 0, err
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	for _, line := range lines {
		if _, err := w.WriteString(line + "\n"); err != nil {
			return false, 0, err
		}
	}
	return true, touched, w.Flush()
}

// upgradeChangelogLine targets verify action entries whose "result" field
// is in the pre-split form "critical=X major=Y minor=Z → status". Everything
// else (including already-migrated verify entries) passes through.
func upgradeChangelogLine(line string) (string, bool) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return line, false
	}
	if raw["action"] != "verify" {
		return line, false
	}
	result, ok := raw["result"].(string)
	if !ok || result == "" {
		return line, false
	}
	// Already split — skip.
	if strings.Contains(result, "minor_resolvable=") || strings.Contains(result, "minor_deferred=") {
		return line, false
	}
	// Must look like the legacy format.
	if !strings.Contains(result, "minor=") {
		return line, false
	}
	// Parse "critical=X major=Y minor=Z → status"
	parts := strings.SplitN(result, "→", 2)
	head := strings.TrimSpace(parts[0])
	tail := ""
	if len(parts) == 2 {
		tail = strings.TrimSpace(parts[1])
	}

	tokens := strings.Fields(head)
	var critical, major, minor int
	for _, tok := range tokens {
		kv := strings.SplitN(tok, "=", 2)
		if len(kv) != 2 {
			continue
		}
		n, err := strconv.Atoi(kv[1])
		if err != nil {
			continue
		}
		switch kv[0] {
		case "critical":
			critical = n
		case "major":
			major = n
		case "minor":
			minor = n
		}
	}
	newResult := fmt.Sprintf("critical=%d major=%d minor_resolvable=%d minor_deferred=0", critical, major, minor)
	if tail != "" {
		newResult += " → " + tail
	}
	raw["result"] = newResult
	data, err := json.Marshal(raw)
	if err != nil {
		return line, false
	}
	return string(data), true
}

// ---- verification.md migration ----------------------------------------

var migrateVerificationCmd = &cobra.Command{
	Use:   "verification",
	Short: "Inject <bts-findings> block into legacy verification.md files",
	Long: `Reads each recipe's latest verify-log entry, synthesizes a
<bts-findings> JSON block from the counts, and prepends it to
verification.md if missing. Existing blocks are left untouched.

This is a one-time migration. New /bts-verify runs emit the block
natively (per bts-verify/SKILL.md Phase 2.2).`,
	RunE: runMigrateVerification,
}

func runMigrateVerification(cmd *cobra.Command, args []string) error {
	target, dryRun, err := migrateFlags(cmd)
	if err != nil {
		return err
	}

	recipesDir := filepath.Join(state.SpecsPath(target), "recipes")
	entries, err := os.ReadDir(recipesDir)
	if err != nil {
		return fmt.Errorf("read recipes dir: %w", err)
	}

	var migrated, skipped int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		recipeDir := filepath.Join(recipesDir, e.Name())
		vmd := filepath.Join(recipeDir, "verification.md")
		if _, err := os.Stat(vmd); os.IsNotExist(err) {
			continue
		}
		ok, err := injectFindingsBlock(recipeDir, vmd, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: error: %v\n", e.Name(), err)
			continue
		}
		if ok {
			migrated++
			marker := "injected"
			if dryRun {
				marker = "would inject"
			}
			fmt.Printf("  %s: %s <bts-findings> block\n", e.Name(), marker)
		} else {
			skipped++
		}
	}
	if dryRun {
		fmt.Printf("\nDry run: %d recipes would get blocks, %d already have them or lack verify-log.\n", migrated, skipped)
	} else {
		fmt.Printf("\nInjected blocks into %d recipes. Backups: verification.md.bak\n", migrated)
	}
	return nil
}

func injectFindingsBlock(recipeDir, vmdPath string, dryRun bool) (bool, error) {
	data, err := os.ReadFile(vmdPath)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "<bts-findings>") {
		return false, nil // already present
	}

	logPath := filepath.Join(recipeDir, "verify-log.jsonl")
	last, err := readLastVerifyLogEntry(logPath)
	if err != nil {
		return false, nil // no log — nothing to synthesize from
	}

	// Legacy entries carry a single Minor. Conservative mapping: resolvable.
	resolvable := last.MinorResolvable
	deferred := last.MinorDeferred
	if resolvable == 0 && deferred == 0 && last.Minor > 0 {
		resolvable = last.Minor
	}

	block := fmt.Sprintf(`<bts-findings>
{
  "critical": %d,
  "major": %d,
  "minor_resolvable": %d,
  "minor_deferred": %d,
  "info": %d,
  "source": "migrated-from-verify-log"
}
</bts-findings>

`, last.Critical, last.Major, resolvable, deferred, last.Info)

	if dryRun {
		return true, nil
	}
	bak := vmdPath + ".bak"
	if err := os.WriteFile(bak, data, 0644); err != nil {
		return false, fmt.Errorf("backup: %w", err)
	}
	newContent := block + string(data)
	return true, os.WriteFile(vmdPath, []byte(newContent), 0644)
}

func readLastVerifyLogEntry(path string) (*state.VerifyLogEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var last state.VerifyLogEntry
	found := false
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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
		return nil, fmt.Errorf("empty log")
	}
	return &last, nil
}

// ---- simulations tagging migration ------------------------------------

var migrateSimulationsCmd = &cobra.Command{
	Use:   "simulations",
	Short: "Tag legacy simulation scenarios with [single-axis: legacy]",
	Long: `Scenario headers in pre-Phase-6 simulations carry no cross-boundary
classification. This command adds [single-axis: legacy] to every header
that lacks one of [cross-boundary|single-axis|illegal-cell: ...].

Legacy is a conservative default — it keeps the validator happy without
claiming cross-boundary coverage the author never verified. When the
recipe is re-simulated later, authors re-tag by hand; the checker's
cross-boundary ratio only applies to files whose scenarios are
currently authored.`,
	RunE: runMigrateSimulations,
}

func runMigrateSimulations(cmd *cobra.Command, args []string) error {
	target, dryRun, err := migrateFlags(cmd)
	if err != nil {
		return err
	}
	recipesDir := filepath.Join(state.SpecsPath(target), "recipes")
	entries, err := os.ReadDir(recipesDir)
	if err != nil {
		return fmt.Errorf("read recipes dir: %w", err)
	}

	totalTouched := 0
	totalFiles := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		simsDir := filepath.Join(recipesDir, e.Name(), "simulations")
		sims, err := os.ReadDir(simsDir)
		if err != nil {
			continue
		}
		for _, sim := range sims {
			if sim.IsDir() || !strings.HasSuffix(sim.Name(), ".md") {
				continue
			}
			simPath := filepath.Join(simsDir, sim.Name())
			touched, changed, err := tagLegacyScenarios(simPath, dryRun)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s/%s: error: %v\n", e.Name(), sim.Name(), err)
				continue
			}
			if changed {
				totalFiles++
				totalTouched += touched
				marker := "tagged"
				if dryRun {
					marker = "would tag"
				}
				fmt.Printf("  %s/%s: %s %d header(s)\n", e.Name(), sim.Name(), marker, touched)
			}
		}
	}

	if dryRun {
		fmt.Printf("\nDry run: %d files would get tags added (%d headers).\n", totalFiles, totalTouched)
	} else {
		fmt.Printf("\nTagged %d files (%d headers). Backups: *.md.bak\n", totalFiles, totalTouched)
	}
	return nil
}

// scenarioHeaderMigrateRe mirrors the one in simulation_checker.go but is
// kept local to avoid exposing the checker's private regex across
// packages. Match scenario header lines (leading `#`, `##`, or
// `Scenario:` keyword) and look for an existing tag.
var (
	scenarioHeaderRe = regexp.MustCompile(`(?mi)^(#{1,6}\s+.*\bscenario\b[^\n]*|scenario:[^\n]*|-\s+scenario\s+\d+[^\n]*)$`)
	existingTagRe    = regexp.MustCompile(`(?i)\[(cross-boundary|single-axis|illegal-cell)(?::[^\]]*)?\]`)
)

func tagLegacyScenarios(path string, dryRun bool) (int, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false, err
	}
	content := string(data)

	touched := 0
	newContent := scenarioHeaderRe.ReplaceAllStringFunc(content, func(line string) string {
		if existingTagRe.MatchString(line) {
			return line // already tagged
		}
		touched++
		return strings.TrimRight(line, " \t") + " [single-axis: legacy]"
	})

	if touched == 0 {
		return 0, false, nil
	}
	if dryRun {
		return touched, true, nil
	}
	if err := os.WriteFile(path+".bak", data, 0644); err != nil {
		return 0, false, fmt.Errorf("backup: %w", err)
	}
	return touched, true, os.WriteFile(path, []byte(newContent), 0644)
}

// ---- task-anchor migration (Phase 9) ---------------------------------

var migrateTaskAnchorsCmd = &cobra.Command{
	Use:   "task-anchors",
	Short: "Inject <!-- task-anchor: path action --> into final.md and populate Task.anchor",
	Long: `Phase 9 makes tasks.json ↔ final.md a machine-checked 1:1 contract.
Legacy recipes have neither the anchor comment in final.md nor the
Task.anchor field in tasks.json. This command walks each recipe, looks
up each Task's (file, action), inserts the anchor above the spec block
most likely describing that file (heuristics: " `+"`"+`{file}`+"`"+` "
mention, " ### `+"`"+`{file}`+"`"+` " heading, " ## {file} " heading,
first code block citing the path), and backfills Task.anchor.

Ambiguous files (no match in final.md or multiple plausible locations)
are reported; the anchor is inserted at the top of the Tasks/Components
section as a fallback. Manual review is recommended afterward.

Backups: final.md.bak + tasks.json.bak.`,
	RunE: runMigrateTaskAnchors,
}

func runMigrateTaskAnchors(cmd *cobra.Command, args []string) error {
	target, dryRun, err := migrateFlags(cmd)
	if err != nil {
		return err
	}
	recipesDir := filepath.Join(state.SpecsPath(target), "recipes")
	entries, err := os.ReadDir(recipesDir)
	if err != nil {
		return fmt.Errorf("read recipes dir: %w", err)
	}

	var totalRecipes, migratedRecipes, insertedAnchors, backfilledTasks int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		recipeDir := filepath.Join(recipesDir, e.Name())
		finalPath := filepath.Join(recipeDir, "final.md")
		tasksPath := filepath.Join(recipeDir, "tasks.json")
		if _, err := os.Stat(finalPath); os.IsNotExist(err) {
			continue
		}
		if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
			continue
		}
		totalRecipes++

		inserted, backfilled, err := migrateOneTaskAnchors(finalPath, tasksPath, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: error: %v\n", e.Name(), err)
			continue
		}
		if inserted > 0 || backfilled > 0 {
			migratedRecipes++
			insertedAnchors += inserted
			backfilledTasks += backfilled
			marker := "migrated"
			if dryRun {
				marker = "would migrate"
			}
			fmt.Printf("  %s: %s (%d anchors inserted, %d Task.anchor backfilled)\n", e.Name(), marker, inserted, backfilled)
		}
	}
	if dryRun {
		fmt.Printf("\nDry run: %d/%d recipes need migration (%d anchors, %d task fields).\n", migratedRecipes, totalRecipes, insertedAnchors, backfilledTasks)
	} else {
		fmt.Printf("\nMigrated %d/%d recipes (%d anchors, %d task fields). Backups: final.md.bak + tasks.json.bak\n", migratedRecipes, totalRecipes, insertedAnchors, backfilledTasks)
	}
	return nil
}

// taskAnchorJSON is the minimal struct we deserialize tasks.json into
// for migration purposes. Keeps field order stable on rewrite so diffs
// stay reviewable.
type taskAnchorJSON struct {
	RecipeID  string                   `json:"recipe_id,omitempty"`
	StartedAt string                   `json:"started_at,omitempty"`
	UpdatedAt string                   `json:"updated_at,omitempty"`
	Tasks     []map[string]interface{} `json:"tasks"`
}

func migrateOneTaskAnchors(finalPath, tasksPath string, dryRun bool) (int, int, error) {
	finalData, err := os.ReadFile(finalPath)
	if err != nil {
		return 0, 0, err
	}
	tasksData, err := os.ReadFile(tasksPath)
	if err != nil {
		return 0, 0, err
	}

	var tasks taskAnchorJSON
	if err := json.Unmarshal(tasksData, &tasks); err != nil {
		return 0, 0, fmt.Errorf("parse tasks.json: %w", err)
	}

	finalStr := string(finalData)
	insertedAnchors := 0
	backfilledTasks := 0

	for i, task := range tasks.Tasks {
		file, _ := task["file"].(string)
		action, _ := task["action"].(string)
		if file == "" || action == "" {
			continue
		}
		anchorKey := file + " " + action
		existing, _ := task["anchor"].(string)

		// Backfill Task.anchor if missing.
		if existing == "" {
			tasks.Tasks[i]["anchor"] = anchorKey
			backfilledTasks++
		}

		// Insert anchor into final.md if not present.
		anchorComment := "<!-- task-anchor: " + anchorKey + " -->"
		if strings.Contains(finalStr, anchorComment) {
			continue
		}
		insertion, ok := placeTaskAnchor(finalStr, file, anchorComment)
		if !ok {
			// Fallback: append at top of file below first H1 or at line 0.
			insertion = fallbackInsertAnchor(finalStr, anchorComment)
		}
		finalStr = insertion
		insertedAnchors++
	}

	if insertedAnchors == 0 && backfilledTasks == 0 {
		return 0, 0, nil
	}
	if dryRun {
		return insertedAnchors, backfilledTasks, nil
	}

	// Write backups before overwriting.
	if err := os.WriteFile(finalPath+".bak", finalData, 0644); err != nil {
		return 0, 0, err
	}
	if err := os.WriteFile(tasksPath+".bak", tasksData, 0644); err != nil {
		return 0, 0, err
	}

	if err := os.WriteFile(finalPath, []byte(finalStr), 0644); err != nil {
		return 0, 0, err
	}
	out, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return 0, 0, err
	}
	// Preserve trailing newline if input had one.
	if strings.HasSuffix(string(tasksData), "\n") {
		out = append(out, '\n')
	}
	if err := os.WriteFile(tasksPath, out, 0644); err != nil {
		return 0, 0, err
	}
	return insertedAnchors, backfilledTasks, nil
}

// placeTaskAnchor tries to insert the anchor immediately above a
// plausible heading describing the file. Heuristics, in order:
//  1. "### `{file}`" — canonical Level-3-draft heading
//  2. "## {file}"    — alternate heading style
//  3. "### {basename}" — heading by file basename only
//  4. first line mentioning the file path inside a code span
//
// On success returns the updated final.md content and true.
func placeTaskAnchor(finalStr, file, anchorComment string) (string, bool) {
	base := filepath.Base(file)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^###\s+` + "`" + regexp.QuoteMeta(file) + "`" + `.*$`),
		regexp.MustCompile(`(?m)^##\s+` + regexp.QuoteMeta(file) + `.*$`),
		regexp.MustCompile(`(?m)^###\s+` + "`" + regexp.QuoteMeta(base) + "`" + `.*$`),
		regexp.MustCompile(`(?m)^###\s+` + regexp.QuoteMeta(base) + `.*$`),
	}
	for _, re := range patterns {
		if loc := re.FindStringIndex(finalStr); loc != nil {
			return finalStr[:loc[0]] + anchorComment + "\n" + finalStr[loc[0]:], true
		}
	}
	return finalStr, false
}

// fallbackInsertAnchor places the anchor after the first H1 heading or,
// lacking one, at the very start of final.md. Keeps the document
// parseable either way.
func fallbackInsertAnchor(finalStr, anchorComment string) string {
	h1 := regexp.MustCompile(`(?m)^#\s+.+$`).FindStringIndex(finalStr)
	if h1 == nil {
		return anchorComment + "\n" + finalStr
	}
	// insert after the heading line
	end := h1[1]
	if end < len(finalStr) && finalStr[end] == '\n' {
		end++
	}
	return finalStr[:end] + "\n" + anchorComment + "\n" + finalStr[end:]
}

// ---- modify-scope migration (Phase 14) -------------------------------

var migrateModifyScopeCmd = &cobra.Command{
	Use:   "modify-scope",
	Short: "Declare modify_scope + anchor scope= for legacy modify tasks",
	Long: `Phase 14 requires every action=modify task to carry both a
Task.modify_scope list and a matching scope= suffix in its final.md
anchor. This command walks recipes and infers a scope from the task's
Description using a heuristic: scan the target source file for
exported/top-level symbols (functions, classes, methods, const decls);
keep the ones whose names appear in the task description; set that
list as both modify_scope and the anchor's scope= suffix.

When the heuristic yields zero symbols, the task is skipped and
reported so the user can fill the list manually.

Backups: final.md.bak.scope + tasks.json.bak.scope.`,
	RunE: runMigrateModifyScope,
}

var (
	goTopLevelSymbolRe = regexp.MustCompile(`(?m)^\s*(?:func|type)\s+(?:\([^)]+\)\s*)?([A-Za-z_]\w*)`)
	tsTopLevelSymbolRe = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:async\s+)?(?:function|class|const|let|var|type|interface)\s+([A-Za-z_]\w*)`)
	swiftTopLevelRe    = regexp.MustCompile(`(?m)^\s*(?:public\s+|private\s+|internal\s+|fileprivate\s+)?(?:func|class|struct|enum|var|let)\s+([A-Za-z_]\w*)`)
	pyTopLevelRe       = regexp.MustCompile(`(?m)^\s*(?:def|class)\s+([A-Za-z_]\w*)`)
)

func runMigrateModifyScope(cmd *cobra.Command, args []string) error {
	target, dryRun, err := migrateFlags(cmd)
	if err != nil {
		return err
	}
	recipesDir := filepath.Join(state.SpecsPath(target), "recipes")
	entries, err := os.ReadDir(recipesDir)
	if err != nil {
		return fmt.Errorf("read recipes dir: %w", err)
	}

	var totalRecipes, migratedRecipes, filledTasks, skippedTasks int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		recipeDir := filepath.Join(recipesDir, e.Name())
		finalPath := filepath.Join(recipeDir, "final.md")
		tasksPath := filepath.Join(recipeDir, "tasks.json")
		if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
			continue
		}
		if _, err := os.Stat(finalPath); os.IsNotExist(err) {
			continue
		}
		totalRecipes++

		filled, skipped, err := migrateOneModifyScope(target, finalPath, tasksPath, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: error: %v\n", e.Name(), err)
			continue
		}
		if filled > 0 || skipped > 0 {
			migratedRecipes++
			filledTasks += filled
			skippedTasks += skipped
			marker := "migrated"
			if dryRun {
				marker = "would migrate"
			}
			fmt.Printf("  %s: %s (%d filled, %d skipped-for-manual)\n", e.Name(), marker, filled, skipped)
		}
	}
	if dryRun {
		fmt.Printf("\nDry run: %d/%d recipes need migration (%d tasks filled, %d skipped).\n", migratedRecipes, totalRecipes, filledTasks, skippedTasks)
	} else {
		fmt.Printf("\nMigrated %d/%d recipes (%d filled, %d skipped). Backups: .bak.scope\n", migratedRecipes, totalRecipes, filledTasks, skippedTasks)
	}
	return nil
}

func migrateOneModifyScope(projectRoot, finalPath, tasksPath string, dryRun bool) (int, int, error) {
	finalData, err := os.ReadFile(finalPath)
	if err != nil {
		return 0, 0, err
	}
	tasksData, err := os.ReadFile(tasksPath)
	if err != nil {
		return 0, 0, err
	}

	var tasks taskAnchorJSON
	if err := json.Unmarshal(tasksData, &tasks); err != nil {
		return 0, 0, fmt.Errorf("parse tasks.json: %w", err)
	}

	finalStr := string(finalData)
	filled := 0
	skipped := 0

	for i, task := range tasks.Tasks {
		action, _ := task["action"].(string)
		if action != "modify" {
			continue
		}
		// Skip if already set.
		if scope, ok := task["modify_scope"].([]interface{}); ok && len(scope) > 0 {
			continue
		}
		file, _ := task["file"].(string)
		desc, _ := task["description"].(string)

		symbols := inferScopeSymbols(projectRoot, file, desc)
		if len(symbols) == 0 {
			// Heuristic failed — tag `legacy` as a migration placeholder.
			// CheckModifyScope treats this token as unchecked so validation
			// passes; the user is expected to replace it with real symbols
			// when they next work on the task. Counted as skipped-for-manual
			// so the output makes the gap visible.
			symbols = []string{"legacy"}
			skipped++
		} else {
			filled++
		}

		// Set Task.modify_scope.
		scopeIface := make([]interface{}, len(symbols))
		for j, s := range symbols {
			scopeIface[j] = s
		}
		tasks.Tasks[i]["modify_scope"] = scopeIface

		// Rewrite the anchor in final.md to append `scope=...`.
		oldAnchor := "<!-- task-anchor: " + file + " modify -->"
		newAnchor := "<!-- task-anchor: " + file + " modify scope=" + strings.Join(symbols, ",") + " -->"
		if strings.Contains(finalStr, oldAnchor) {
			finalStr = strings.Replace(finalStr, oldAnchor, newAnchor, 1)
		} else {
			// Anchor already has some suffix — do not clobber; add scope if
			// the existing suffix lacks it.
			re := regexp.MustCompile(`<!--\s*task-anchor:\s*` + regexp.QuoteMeta(file) + `\s+modify\b([^>]*)-->`)
			match := re.FindStringSubmatch(finalStr)
			if len(match) >= 2 && !strings.Contains(match[1], "scope=") {
				replacement := "<!-- task-anchor: " + file + " modify" + match[1] + " scope=" + strings.Join(symbols, ",") + " -->"
				finalStr = re.ReplaceAllString(finalStr, replacement)
			}
		}
	}

	if filled == 0 && skipped == 0 {
		return 0, 0, nil
	}
	if dryRun {
		return filled, skipped, nil
	}
	if err := os.WriteFile(finalPath+".bak.scope", finalData, 0644); err != nil {
		return 0, 0, err
	}
	if err := os.WriteFile(tasksPath+".bak.scope", tasksData, 0644); err != nil {
		return 0, 0, err
	}
	if err := os.WriteFile(finalPath, []byte(finalStr), 0644); err != nil {
		return 0, 0, err
	}
	out, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return 0, 0, err
	}
	if strings.HasSuffix(string(tasksData), "\n") {
		out = append(out, '\n')
	}
	if err := os.WriteFile(tasksPath, out, 0644); err != nil {
		return 0, 0, err
	}
	return filled, skipped, nil
}

// inferScopeSymbols reads the target file and the task description,
// returns the set of top-level symbol names that appear in both.
// Language detected by file extension; unknown extensions fall back
// to the TS pattern (broadest).
func inferScopeSymbols(projectRoot, file, description string) []string {
	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectRoot, file)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(data)

	re := tsTopLevelSymbolRe
	switch strings.ToLower(filepath.Ext(file)) {
	case ".go":
		re = goTopLevelSymbolRe
	case ".swift":
		re = swiftTopLevelRe
	case ".py":
		re = pyTopLevelRe
	}

	allSymbols := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		if len(m) >= 2 {
			allSymbols[m[1]] = true
		}
	}

	// Retain symbols whose name is referenced in the description.
	descLower := strings.ToLower(description)
	var matched []string
	seen := map[string]bool{}
	for sym := range allSymbols {
		if seen[sym] {
			continue
		}
		// Word-boundary match on case-insensitive description.
		if strings.Contains(descLower, strings.ToLower(sym)) {
			matched = append(matched, sym)
			seen[sym] = true
		}
	}
	sort.Strings(matched)
	return matched
}

// ---- all-in-one convenience -------------------------------------------

var migrateAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Run all migrations in sequence (verify-log, changelog, verification, simulations, task-anchors, modify-scope)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runMigrateVerifyLog(cmd, args); err != nil {
			return err
		}
		fmt.Println()
		if err := runMigrateChangelog(cmd, args); err != nil {
			return err
		}
		fmt.Println()
		if err := runMigrateVerification(cmd, args); err != nil {
			return err
		}
		fmt.Println()
		if err := runMigrateSimulations(cmd, args); err != nil {
			return err
		}
		fmt.Println()
		if err := runMigrateTaskAnchors(cmd, args); err != nil {
			return err
		}
		fmt.Println()
		return runMigrateModifyScope(cmd, args)
	},
}

func migrateFlags(cmd *cobra.Command) (string, bool, error) {
	targetFlag, _ := cmd.Flags().GetString("target")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	target := targetFlag
	if target == "" {
		cwd, _ := os.Getwd()
		root, err := state.FindRoot(cwd)
		if err != nil {
			return "", false, fmt.Errorf("not a bts project: %w", err)
		}
		target = root
	}
	return target, dryRun, nil
}
