package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/macosta/tdd-ai/internal/phase"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
	"github.com/spf13/cobra"
)

var completeTestResultFlag string
var completeSummaryFlag bool

var completeCmd = &cobra.Command{
	Use:   "complete",
	Short: "Finish the current TDD cycle (advance to done + mark all specs complete)",
	Long: `Shortcut to wrap up a TDD cycle. This command:

1. Validates tests pass (runs test command if configured, or uses --test-result pass)
2. Advances through remaining phases to done
3. Marks all active specs as completed

This is the "I'm done, wrap it up" command.`,
	Example: `  tdd-ai complete
  tdd-ai complete --test-result pass`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		if s.Phase == types.PhaseDone && len(s.ActiveSpecs()) == 0 {
			return fmt.Errorf("nothing to complete: already in done phase with no active specs")
		}

		// Determine test result: explicit flag > cached session result > run test command
		testResult := completeTestResultFlag

		// If no explicit flag, check for a cached result from a recent 'tdd-ai test' run
		if testResult == "" && s.LastTestResult != "" {
			testResult = s.LastTestResult
			fmt.Fprintf(cmd.OutOrStdout(), "Using last test result from session: %s\n\n", testResult)
		}

		// If still no result and a test command is configured, run it
		if testResult == "" && s.TestCmd != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Running: %s\n\n", s.TestCmd)

			parts := strings.Fields(s.TestCmd)
			c := exec.Command(parts[0], parts[1:]...)
			c.Dir = dir
			output, execErr := c.CombinedOutput()

			if len(output) > 0 {
				printTestOutput(cmd, string(output), completeSummaryFlag)
			}

			testResult = classifyTestResult(string(output), execErr)
			fmt.Fprintf(cmd.OutOrStdout(), "\nTest result: %s\n\n", strings.ToUpper(testResult))
		}

		// Validate test result
		if testResult == "" {
			return fmt.Errorf("cannot complete: no test result available. Either configure --test-cmd, run 'tdd-ai test', or pass --test-result pass")
		}
		if testResult == "error" {
			return fmt.Errorf("cannot complete: last test run was an infrastructure/environment error (not a test failure). Fix the environment and re-run 'tdd-ai test'")
		}
		if testResult != "pass" {
			return fmt.Errorf("cannot complete: tests are failing. Fix tests before completing the cycle")
		}

		// Block complete when in refactor with unanswered reflections
		if s.Phase == types.PhaseRefactor && len(s.Reflections) > 0 && !s.AllReflectionsAnswered() {
			pending := s.PendingReflections()
			return fmt.Errorf("cannot complete: %d reflection question(s) unanswered. Use 'tdd-ai refactor status' to see them", len(pending))
		}

		// Advance through remaining phases to done
		phasesAdvanced := 0
		mode := s.GetMode()
		for s.Phase != types.PhaseDone {
			next, err := phase.NextWithMode(s.Phase, mode)
			if err != nil {
				return fmt.Errorf("advancing phase: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Phase: %s -> %s\n", s.Phase, next)
			s.Phase = next
			phasesAdvanced++
		}

		// Mark all active specs as completed
		specsCompleted := s.CompleteAllSpecs()

		// Record completion event
		s.AddEvent("complete", func(e *types.Event) {
			e.Result = testResult
			e.SpecCount = specsCompleted
		})

		// Clear last test result
		s.LastTestResult = ""

		if err := session.Save(dir, s); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\nCycle complete: advanced %d phase(s), marked %d spec(s) as done\n", phasesAdvanced, specsCompleted)
		fmt.Fprintln(cmd.OutOrStdout(), "Next: add more specs or run 'tdd-ai reset' to start over")
		return nil
	},
}

func init() {
	completeCmd.Flags().StringVar(&completeTestResultFlag, "test-result", "", "test outcome: 'pass' (required if no test command configured)")
	completeCmd.Flags().BoolVar(&completeSummaryFlag, "summary", false, "show only the last 20 lines of test output (saves LLM context window)")
	rootCmd.AddCommand(completeCmd)
}
