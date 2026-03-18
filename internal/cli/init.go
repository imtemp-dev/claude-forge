package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jlim/bts/internal/template"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().Bool("force", false, "Reinitialize (overwrites existing bts files)")
}

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize bts in a project",
	Long:  "Deploy skills, agents, hooks, and rules to .claude/ and create .bts/ for state management.",
	Args:  cobra.MaximumNArgs(1),
	GroupID: "project",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine project root
	projectRoot := "."
	if len(args) > 0 {
		projectRoot = args[0]
	}

	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Check if already initialized
	btsDir := filepath.Join(absRoot, ".bts")
	force, _ := cmd.Flags().GetBool("force")
	if _, err := os.Stat(btsDir); err == nil && !force {
		return fmt.Errorf(".bts/ already exists. Use --force to reinitialize")
	}

	fmt.Println("Initializing bts...")

	// Create .bts directories
	stateDirs := []string{
		filepath.Join(btsDir, "config"),
		filepath.Join(btsDir, "state", "recipes"),
		filepath.Join(btsDir, "state", "debates"),
	}
	for _, dir := range stateDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	// Deploy templates
	created, err := template.Deploy(absRoot)
	if err != nil {
		return fmt.Errorf("deploy templates: %w", err)
	}

	fmt.Printf("\nbts initialized successfully.\n")
	fmt.Printf("  Files created: %d\n", len(created))
	fmt.Printf("  Skills:        .claude/skills/bts/\n")
	fmt.Printf("  Agents:        .claude/agents/bts/\n")
	fmt.Printf("  Commands:      .claude/commands/bts/\n")
	fmt.Printf("  Rules:         .claude/rules/bts/\n")
	fmt.Printf("  Hooks:         .claude/hooks/bts/\n")
	fmt.Printf("  State:         .bts/\n")
	fmt.Printf("\nStart Claude Code and try: /recipe blueprint \"your feature\"\n")

	return nil
}
