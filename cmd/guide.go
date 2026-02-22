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
	Short: "Show current TDD state: phase, specs, blockers, and expected test result",
	Long: `Outputs the current TDD session state including phase, mode, active specs,
expected test result, blockers preventing advancement, and reflections.

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
