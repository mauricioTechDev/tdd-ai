package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	version    = "dev"
	formatFlag string
)

var rootCmd = &cobra.Command{
	Use:   "tdd-ai",
	Short: "TDD guardrails for AI coding agents",
	Long: `tdd-ai is a TDD state machine that keeps AI coding agents disciplined.

It tracks specs (what to build), phases (red/green/refactor), and provides
structured guidance telling the AI what to do and what NOT to do.

The CLI does NOT run tests — the AI agent runs tests itself. This tool
provides the guardrails and feedback loop that LLMs lack on their own.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Auto-detect format: default to JSON when stdout is not a terminal
		// (i.e., when an AI agent is running the CLI via pipe/redirect).
		// Explicit --format flag always overrides.
		if !cmd.Flags().Changed("format") && !isTerminal() {
			formatFlag = "json"
		}
	},
}

// isTerminal reports whether stdout is connected to a terminal.
// Extracted for testability.
var isTerminal = func() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "text", "output format: text or json (default: json when non-interactive)")
}

func getWorkDir() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	return dir
}
