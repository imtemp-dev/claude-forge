package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jlim/bts/internal/engine"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(verifyCmd)
}

var verifyCmd = &cobra.Command{
	Use:     "verify <file>",
	Short:   "Deterministic fact-check a document against source code",
	Args:    cobra.ExactArgs(1),
	GroupID: "tools",
	RunE:    runVerify,
}

func runVerify(cmd *cobra.Command, args []string) error {
	docPath := args[0]

	// Find project root (cwd by default)
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	// Make doc path absolute if relative
	if !filepath.IsAbs(docPath) {
		docPath = filepath.Join(projectRoot, docPath)
	}

	result, err := engine.VerifyFile(docPath, projectRoot)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}

	// Output as JSON (machine-readable for skills to parse)
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	fmt.Println(string(data))

	// Exit code based on results
	if result.Summary.Critical > 0 || result.Summary.Major > 0 {
		os.Exit(1)
	}

	return nil
}
