package cmd

import (
	"fmt"

	"github.com/macosta/tdd-ai/internal/formatter"
	"github.com/macosta/tdd-ai/internal/guide"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/spf13/cobra"
)

var guideCmd = &cobra.Command{
	Use:   "guide",
	Short: "Get phase-appropriate TDD instructions for the AI agent",
	Long: `Outputs structured guidance based on the current TDD phase.

Use --format json for machine-readable output that AI agents can parse.
Use --format text (default) for human-readable output.`,
	Example: `  tdd-ai guide
  tdd-ai guide --format json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		g := guide.Generate(s)
		out, err := formatter.FormatGuidance(g, formatter.Format(formatFlag))
		if err != nil {
			return err
		}

		fmt.Fprint(cmd.OutOrStdout(), out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(guideCmd)
}
