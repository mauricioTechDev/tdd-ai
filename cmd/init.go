package cmd

import (
	"fmt"

	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
	"github.com/spf13/cobra"
)

var (
	retrofitFlag bool
	testCmdFlag  string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new TDD session",
	Long: `Creates a .tdd-ai.json file in the current directory to track TDD state.

Use --retrofit when adding tests to existing code. In retrofit mode, the RED phase
expects tests to PASS (since implementation exists) and the GREEN phase is skipped.

Use --test-cmd to configure the project's test command. This enables the 'tdd-ai test'
command and auto-populates the test result for 'phase next'.`,
	Example: `  tdd-ai init
  tdd-ai init --retrofit
  tdd-ai init --test-cmd "go test ./..."
  tdd-ai init --retrofit --test-cmd "dotnet test MyProject.Tests"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getWorkDir()

		if session.Exists(dir) {
			return fmt.Errorf("TDD session already exists. Use 'tdd-ai reset' to start over")
		}

		mode := types.ModeGreenfield
		if retrofitFlag {
			mode = types.ModeRetrofit
		}

		s, err := session.CreateWithMode(dir, mode)
		if err != nil {
			return err
		}

		if testCmdFlag != "" {
			s.TestCmd = testCmdFlag
		}

		s.AddEvent("init", func(e *types.Event) {
			e.Result = string(s.GetMode())
		})

		if err := session.Save(dir, s); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "TDD session initialized (phase: %s, mode: %s)\n", s.Phase, s.GetMode())
		if s.TestCmd != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Test command: %s\n", s.TestCmd)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Next: add specs with 'tdd-ai spec add \"desc1\" \"desc2\" ...'")
		fmt.Fprintln(cmd.OutOrStdout(), "Available commands: spec add, guide, test, phase next, complete, status, reset")
		fmt.Fprintln(cmd.OutOrStdout(), "Run 'tdd-ai commands' for full reference.")
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&retrofitFlag, "retrofit", false, "use retrofit mode for testing existing code")
	initCmd.Flags().StringVar(&testCmdFlag, "test-cmd", "", "test command to run (e.g. 'go test ./...', 'npm test')")
	rootCmd.AddCommand(initCmd)
}
