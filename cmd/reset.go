package cmd

import (
	"fmt"
	"os"

	"github.com/macosta/tdd-ai/internal/session"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:     "reset",
	Short:   "Clear the current TDD session",
	Long:    "Removes the .tdd-ai.json file, allowing you to start fresh with 'tdd-ai init'.",
	Example: `  tdd-ai reset`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir := getWorkDir()

		if !session.Exists(dir) {
			return fmt.Errorf("no TDD session found")
		}

		if err := os.Remove(session.FilePath(dir)); err != nil {
			return fmt.Errorf("removing session file: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "TDD session cleared. Run 'tdd-ai init' to start a new one.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}
