package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/macosta/tdd-ai/internal/formatter"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/verify"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Check TDD compliance of the current session",
	Long: `Analyzes the session history for TDD compliance violations.

Checks that each completed spec followed the RED-GREEN-REFACTOR cycle:
- Every spec has a spec_picked event
- A failing test was recorded during RED phase (greenfield mode)
- No phase_set usage (bypassing TDD guardrails)

Returns exit code 0 when compliant, 1 when violations are found.`,
	Example: `  tdd-ai verify
  tdd-ai verify --format json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		result := verify.Analyze(s)

		f := formatter.Format(formatFlag)
		switch f {
		case formatter.FormatJSON:
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("encoding verify result: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
		case formatter.FormatText:
			var b strings.Builder
			fmt.Fprintf(&b, "TDD Compliance: %.0f%%\n", result.Score)
			fmt.Fprintf(&b, "Specs verified: %d, compliant: %d\n", result.SpecsVerified, result.SpecsCompliant)

			if len(result.Violations) > 0 {
				b.WriteString("\nViolations:\n")
				for _, v := range result.Violations {
					if v.SpecID > 0 {
						fmt.Fprintf(&b, "  [spec %d] %s: %s\n", v.SpecID, v.Rule, v.Message)
					} else {
						fmt.Fprintf(&b, "  %s: %s\n", v.Rule, v.Message)
					}
				}
			} else {
				b.WriteString("\nNo violations found.\n")
			}
			fmt.Fprint(cmd.OutOrStdout(), b.String())
		default:
			return fmt.Errorf("unknown format: %q", formatFlag)
		}

		if !result.Compliant {
			return fmt.Errorf("%d violation(s) found", len(result.Violations))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}
