package cmd

import (
	"fmt"

	"github.com/macosta/tdd-ai/internal/phase"
	"github.com/macosta/tdd-ai/internal/reflection"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
	"github.com/spf13/cobra"
)

var phaseCmd = &cobra.Command{
	Use:   "phase",
	Short: "Show or manage the current TDD phase",
	Long:  "Show the current TDD phase, advance to the next phase, or manually set a phase.",
	Example: `  tdd-ai phase
  tdd-ai phase next
  tdd-ai phase set green`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), s.Phase)
		return nil
	},
}

var testResultFlag string

var phaseNextCmd = &cobra.Command{
	Use:   "next",
	Short: "Advance to the next TDD phase",
	Long: `Advance to the next phase in the TDD cycle: red -> green -> refactor -> done.

Use --test-result to validate that tests are in the expected state before advancing.`,
	Example: `  tdd-ai phase next
  tdd-ai phase next --test-result fail`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		current := s.Phase
		if current == types.PhaseRed && len(s.ActiveSpecs()) == 0 {
			return fmt.Errorf("cannot advance from red phase: no active specs. Add specs with 'tdd-ai spec add'")
		}

		mode := s.GetMode()
		expected := phase.ExpectedTestResult(current, mode)

		// Determine the test result: explicit flag > session's last_test_result > warning
		effectiveResult := testResultFlag
		if effectiveResult == "" && s.LastTestResult != "" {
			effectiveResult = s.LastTestResult
			fmt.Fprintf(cmd.ErrOrStderr(), "Using last test result from session: %s\n", effectiveResult)
		}

		if effectiveResult != "" {
			if effectiveResult == "error" {
				return fmt.Errorf("cannot advance: last test run was an infrastructure/environment error (not a test failure). Fix the environment and re-run 'tdd-ai test'")
			}
			if effectiveResult != "pass" && effectiveResult != "fail" {
				return fmt.Errorf("--test-result must be 'pass' or 'fail', got %q", effectiveResult)
			}
			if effectiveResult != expected {
				return fmt.Errorf("cannot advance: %s phase expects tests to %s, but got test result %s", current, expected, effectiveResult)
			}
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: advancing without test result. The %s phase expects tests to %s.\n", current, expected)
		}

		// Block advancing from refactor when reflection questions are unanswered
		if current == types.PhaseRefactor && len(s.Reflections) > 0 && !s.AllReflectionsAnswered() {
			pending := s.PendingReflections()
			return fmt.Errorf("cannot advance: %d reflection question(s) unanswered. Use 'tdd-ai refactor status' to see them", len(pending))
		}

		// Clear last test result after consuming it
		if s.LastTestResult != "" {
			s.LastTestResult = ""
		}

		next, err := phase.NextWithMode(current, mode)
		if err != nil {
			return err
		}

		s.Phase = next
		if next == types.PhaseRefactor {
			s.Reflections = reflection.DefaultQuestions()
		}
		s.AddEvent("phase_next", func(e *types.Event) {
			e.From = string(current)
			e.To = string(next)
			if effectiveResult != "" {
				e.Result = effectiveResult
			}
		})
		if err := session.Save(dir, s); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Phase: %s -> %s\n", current, next)

		if next == types.PhaseDone {
			fmt.Fprintln(cmd.OutOrStdout(), "Next: mark completed specs with 'tdd-ai spec done --all' or 'tdd-ai spec done <id>'")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Next: run 'tdd-ai guide --format json' for phase instructions")
		}
		return nil
	},
}

var phaseSetCmd = &cobra.Command{
	Use:   "set <red|green|refactor|done>",
	Short: "Manually set the TDD phase",
	Long:  "Override the current phase. Use with caution — this bypasses the normal TDD progression.",
	Example: `  tdd-ai phase set red
  tdd-ai phase set green
  tdd-ai phase set refactor`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		p := types.Phase(args[0])
		if !p.IsValid() {
			return fmt.Errorf("invalid phase %q. Valid phases: red, green, refactor, done", args[0])
		}

		old := s.Phase
		s.Phase = p
		if p == types.PhaseRefactor && len(s.Reflections) == 0 {
			s.Reflections = reflection.DefaultQuestions()
		}
		s.AddEvent("phase_set", func(e *types.Event) {
			e.From = string(old)
			e.To = string(p)
		})
		if err := session.Save(dir, s); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Phase set to: %s\n", p)
		fmt.Fprintln(cmd.OutOrStdout(), "Next: run 'tdd-ai guide --format json' for phase instructions")
		return nil
	},
}

func init() {
	phaseNextCmd.Flags().StringVar(&testResultFlag, "test-result", "", "test outcome: 'pass' or 'fail'")
	phaseCmd.AddCommand(phaseNextCmd)
	phaseCmd.AddCommand(phaseSetCmd)
	rootCmd.AddCommand(phaseCmd)
}
