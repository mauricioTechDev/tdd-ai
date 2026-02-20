package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/macosta/tdd-ai/internal/formatter"
	"github.com/macosta/tdd-ai/internal/phase"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
	"github.com/spf13/cobra"
)

type blockersOutput struct {
	Phase      types.Phase `json:"phase"`
	Blockers   []string    `json:"blockers"`
	CanAdvance bool        `json:"can_advance"`
}

var blockersCmd = &cobra.Command{
	Use:   "blockers",
	Short: "Show what's preventing phase advancement",
	Long:  "Returns the current blockers that must be resolved before advancing to the next phase.",
	Example: `  tdd-ai blockers
  tdd-ai blockers --format json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		blockers := phase.GetBlockers(s)
		out := blockersOutput{
			Phase:      s.Phase,
			Blockers:   blockers,
			CanAdvance: len(blockers) == 0,
		}

		f := formatter.Format(formatFlag)
		switch f {
		case formatter.FormatJSON:
			data, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return fmt.Errorf("encoding blockers: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
		case formatter.FormatText:
			if len(blockers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no blockers)")
			} else {
				var b strings.Builder
				fmt.Fprintf(&b, "Phase: %s\n", strings.ToUpper(string(s.Phase)))
				b.WriteString("Blockers:\n")
				for _, bl := range blockers {
					fmt.Fprintf(&b, "  - %s\n", bl)
				}
				fmt.Fprint(cmd.OutOrStdout(), b.String())
			}
		default:
			return fmt.Errorf("unknown format: %q", f)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(blockersCmd)
}
