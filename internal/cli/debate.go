package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jlim/bts/internal/state"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(debateCmd)
	debateCmd.AddCommand(debateListCmd, debateExportCmd)
}

var debateCmd = &cobra.Command{
	Use:     "debate",
	Short:   "Manage debate state",
	GroupID: "tools",
}

var debateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all debates",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		btsRoot, err := state.FindBTSRoot(cwd)
		if err != nil {
			return fmt.Errorf("not a bts project: %w", err)
		}

		debates, err := state.ListDebates(btsRoot)
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}

		if len(debates) == 0 {
			fmt.Println("No debates found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTopic\tRounds\tDecided\tUpdated")
		for _, d := range debates {
			fmt.Fprintf(w, "%s\t%s\t%d\t%v\t%s\n",
				d.ID, truncate(d.Topic, 30), d.Rounds, d.Decided, d.UpdatedAt)
		}
		w.Flush()
		return nil
	},
}

var debateExportCmd = &cobra.Command{
	Use:   "export <debate-id>",
	Short: "Export debate as markdown",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		btsRoot, err := state.FindBTSRoot(cwd)
		if err != nil {
			return fmt.Errorf("not a bts project: %w", err)
		}

		ds, err := state.LoadDebateState(btsRoot, args[0])
		if err != nil {
			return fmt.Errorf("load debate: %w", err)
		}

		fmt.Printf("# Debate: %s\n\n", ds.Topic)
		fmt.Printf("Rounds: %d | Decided: %v\n\n", ds.Rounds, ds.Decided)
		if ds.Conclusion != "" {
			fmt.Printf("## Conclusion\n%s\n\n", ds.Conclusion)
		}

		// Print round files if they exist
		for i := 1; i <= ds.Rounds; i++ {
			roundPath := fmt.Sprintf("%s/round-%d.md", state.DebateDir(btsRoot, args[0]), i)
			data, err := os.ReadFile(roundPath)
			if err == nil {
				fmt.Printf("## Round %d\n%s\n\n", i, string(data))
			}
		}

		return nil
	},
}
