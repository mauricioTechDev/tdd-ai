package cmd

import (
	"fmt"

	"github.com/macosta/tdd-ai/internal/formatter"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current TDD session status",
	Long:  "Display a full overview of the TDD session: current phase, mode, spec summary, and recommended next action.",
	Example: `  tdd-ai status
  tdd-ai status --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		out, err := formatter.FormatFullStatus(s, formatter.Format(formatFlag))
		if err != nil {
			return err
		}

		fmt.Fprint(cmd.OutOrStdout(), out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
