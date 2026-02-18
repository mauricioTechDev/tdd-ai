package cmd

import (
	"fmt"

	"github.com/macosta/tdd-ai/internal/formatter"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Show a compact checkpoint for recovering agent context after compression",
	Long: `Outputs a minimal situation briefing: current phase, working spec, any blockers,
and the single next action to take.

Designed to be run as the first command by a new agent or after context compression
to quickly re-orient to the TDD session state without reading the full history.`,
	Example: `  tdd-ai resume
  tdd-ai resume --format json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		out, err := formatter.FormatResume(s, formatter.Format(formatFlag))
		if err != nil {
			return err
		}

		fmt.Fprint(cmd.OutOrStdout(), out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
