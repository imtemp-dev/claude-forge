package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// ---- all-in-one convenience -------------------------------------------

var migrateAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Run verify-log + changelog + verification migrations in sequence",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runMigrateVerifyLog(cmd, args); err != nil {
			return err
		}
		fmt.Println()
		if err := runMigrateChangelog(cmd, args); err != nil {
			return err
		}
		fmt.Println()
		return runMigrateVerification(cmd, args)
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
