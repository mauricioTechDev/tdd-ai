package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
	"github.com/spf13/cobra"
)

var testSummaryFlag bool

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run the configured test command and record the result",
	Long: `Runs the test command configured via 'tdd-ai init --test-cmd' and records
whether tests passed or failed. The result is stored in the session and
automatically used by 'tdd-ai phase next' when --test-result is not provided.

Use --summary to show only the last 20 lines of test output. This is useful
for AI agents where full output wastes context window on verbose stack traces.`,
	Example: `  tdd-ai test
  tdd-ai test --summary`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		if s.TestCmd == "" {
			return fmt.Errorf("no test command configured. Use 'tdd-ai init --test-cmd \"your test command\"' to set one")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Running: %s\n\n", s.TestCmd)

		// Split the command for exec
		parts := strings.Fields(s.TestCmd)
		c := exec.Command(parts[0], parts[1:]...)
		c.Dir = dir
		output, execErr := c.CombinedOutput()

		// Print the test output (full or summarized)
		if len(output) > 0 {
			printTestOutput(cmd, string(output), testSummaryFlag)
		}

		// Classify result: pass, fail, or error (infrastructure failure)
		result := classifyTestResult(string(output), execErr)

		// Store result and record event
		s.LastTestResult = result
		s.AddEvent("test_run", func(e *types.Event) {
			e.Result = result
		})
		if err := session.Save(dir, s); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\nTest result: %s\n", strings.ToUpper(result))
		if result == "error" {
			fmt.Fprintln(cmd.OutOrStdout(), "This looks like an infrastructure/environment error, not a test failure.")
			fmt.Fprintln(cmd.OutOrStdout(), "Fix the environment issue and re-run 'tdd-ai test'.")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Next: run 'tdd-ai phase next' (test result stored, will be used automatically)")
		}
		return nil
	},
}

// infraErrorPatterns are substrings that indicate an infrastructure/environment
// failure rather than an actual test failure (e.g. missing binary, broken deps).
var infraErrorPatterns = []string{
	"command not found",
	"Executable doesn't exist",
	"Cannot find module",
	"ENOENT",
	"No such file or directory",
	"MODULE_NOT_FOUND",
	"not recognized as",
	"Permission denied",
}

// classifyTestResult determines whether a failed test run is an actual test
// failure or an infrastructure/environment error by scanning the output.
func classifyTestResult(output string, execErr error) string {
	if execErr == nil {
		return "pass"
	}
	for _, pattern := range infraErrorPatterns {
		if strings.Contains(output, pattern) {
			return "error"
		}
	}
	return "fail"
}

const summaryMaxLines = 20

// printTestOutput prints test output, optionally truncating to the last N lines.
func printTestOutput(cmd *cobra.Command, output string, summary bool) {
	if !summary {
		fmt.Fprint(cmd.OutOrStdout(), output)
		if !strings.HasSuffix(output, "\n") {
			fmt.Fprintln(cmd.OutOrStdout())
		}
		return
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	totalLines := len(lines)
	if totalLines <= summaryMaxLines {
		fmt.Fprint(cmd.OutOrStdout(), output)
		if !strings.HasSuffix(output, "\n") {
			fmt.Fprintln(cmd.OutOrStdout())
		}
		return
	}

	fmt.Fprintf(cmd.OutOrStdout(), "... (%d lines truncated, showing last %d) ...\n", totalLines-summaryMaxLines, summaryMaxLines)
	for _, line := range lines[totalLines-summaryMaxLines:] {
		fmt.Fprintln(cmd.OutOrStdout(), line)
	}
}

func init() {
	testCmd.Flags().BoolVar(&testSummaryFlag, "summary", false, "show only the last 20 lines of test output (saves LLM context window)")
	rootCmd.AddCommand(testCmd)
}
