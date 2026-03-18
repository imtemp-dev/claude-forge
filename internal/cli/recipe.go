package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jlim/bts/internal/state"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(recipeCmd)
	recipeCmd.AddCommand(recipeStatusCmd, recipeListCmd, recipeLogCmd, recipeCancelCmd)
}

var recipeCmd = &cobra.Command{
	Use:     "recipe",
	Short:   "Manage recipe execution state",
	GroupID: "recipe",
}

var recipeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active recipe status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		btsRoot, err := state.FindBTSRoot(cwd)
		if err != nil {
			return fmt.Errorf("not a bts project: %w", err)
		}

		recipe, err := state.GetActiveRecipe(btsRoot)
		if err != nil {
			return fmt.Errorf("read state: %w", err)
		}

		if recipe == nil {
			fmt.Println("No active recipe.")
			return nil
		}

		fmt.Printf("Active recipe: %s\n", recipe.ID)
		fmt.Printf("  Type:      %s\n", recipe.Type)
		fmt.Printf("  Topic:     %s\n", recipe.Topic)
		fmt.Printf("  Phase:     %s\n", recipe.Phase)
		fmt.Printf("  Iteration: %d\n", recipe.Iteration)
		fmt.Printf("  Started:   %s\n", recipe.StartedAt)
		fmt.Printf("  Updated:   %s\n", recipe.UpdatedAt)
		return nil
	},
}

var recipeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recipes",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		btsRoot, err := state.FindBTSRoot(cwd)
		if err != nil {
			return fmt.Errorf("not a bts project: %w", err)
		}

		recipes, err := state.ListRecipes(btsRoot)
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}

		if len(recipes) == 0 {
			fmt.Println("No recipes found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tType\tTopic\tPhase\tIteration\tUpdated")
		for _, r := range recipes {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
				r.ID, r.Type, truncate(r.Topic, 30), r.Phase, r.Iteration, r.UpdatedAt)
		}
		w.Flush()
		return nil
	},
}

var recipeLogCmd = &cobra.Command{
	Use:   "log <recipe-id>",
	Short: "Record a verify iteration result (called by skills via Bash)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		btsRoot, err := state.FindBTSRoot(cwd)
		if err != nil {
			return fmt.Errorf("not a bts project: %w", err)
		}

		iteration, _ := cmd.Flags().GetInt("iteration")
		critical, _ := cmd.Flags().GetInt("critical")
		major, _ := cmd.Flags().GetInt("major")
		minor, _ := cmd.Flags().GetInt("minor")

		status := "continue"
		if critical == 0 && major == 0 {
			status = "converged"
		}

		entry := &state.VerifyLogEntry{
			Iteration: iteration,
			Critical:  critical,
			Major:     major,
			Minor:     minor,
			Status:    status,
		}

		if err := state.AppendVerifyLog(btsRoot, args[0], entry); err != nil {
			return fmt.Errorf("log: %w", err)
		}

		fmt.Printf("Logged iteration %d: critical=%d major=%d minor=%d → %s\n",
			iteration, critical, major, minor, status)
		return nil
	},
}

var recipeCancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel the active recipe",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		btsRoot, err := state.FindBTSRoot(cwd)
		if err != nil {
			return fmt.Errorf("not a bts project: %w", err)
		}

		recipe, err := state.GetActiveRecipe(btsRoot)
		if err != nil || recipe == nil {
			fmt.Println("No active recipe to cancel.")
			return nil
		}

		recipe.Phase = "cancelled"
		if err := state.SaveRecipeState(btsRoot, recipe); err != nil {
			return fmt.Errorf("save: %w", err)
		}

		fmt.Printf("Recipe %s cancelled.\n", recipe.ID)
		return nil
	},
}

func init() {
	recipeLogCmd.Flags().Int("iteration", 0, "Iteration number")
	recipeLogCmd.Flags().Int("critical", 0, "Critical error count")
	recipeLogCmd.Flags().Int("major", 0, "Major error count")
	recipeLogCmd.Flags().Int("minor", 0, "Minor error count")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
