package cmd

import (
	"fmt"
	"strconv"

	"github.com/macosta/tdd-ai/internal/formatter"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
	"github.com/spf13/cobra"
)

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Manage TDD specifications",
	Long:  "Add, list, or complete specs that define what to implement.",
	Example: `  tdd-ai spec add "User can login with email"
  tdd-ai spec list
  tdd-ai spec done 1`,
}

var specAddCmd = &cobra.Command{
	Use:   "add \"description\"",
	Short: "Add a new spec to implement",
	Long:  "Add one or more specs to the current TDD session. Each argument is a separate spec description.",
	Example: `  tdd-ai spec add "User can login with email and password"
  tdd-ai spec add "Returns 404 when not found" "Returns 400 for invalid input"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		for _, desc := range args {
			id := s.AddSpec(desc)
			fmt.Fprintf(cmd.OutOrStdout(), "Spec [%d] added: %s\n", id, desc)
		}

		s.AddEvent("spec_add", func(e *types.Event) {
			e.SpecCount = len(args)
		})

		if err := session.Save(dir, s); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Next: run 'tdd-ai guide --format json' for phase instructions")
		return nil
	},
}

var specListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all specs",
	Long:  "Display all specs in the current session with their status (active or done).",
	Example: `  tdd-ai spec list
  tdd-ai spec list --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		if len(s.Specs) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No specs defined. Add specs with 'tdd-ai spec add \"desc1\" \"desc2\" ...'")
			return nil
		}

		out, err := formatter.FormatStatus(s, formatter.Format(formatFlag))
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), out)
		return nil
	},
}

var specDoneAll bool

var specDoneCmd = &cobra.Command{
	Use:   "done <id> [id...]",
	Short: "Mark a spec as completed",
	Long:  "Mark one or more specs as completed by their ID. Use --all to mark every active spec as done.",
	Example: `  tdd-ai spec done 1
  tdd-ai spec done 1 2 3
  tdd-ai spec done --all`,
	Args: cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		if specDoneAll && len(args) > 0 {
			return fmt.Errorf("cannot use --all with specific spec IDs")
		}
		if !specDoneAll && len(args) == 0 {
			return fmt.Errorf("provide at least one spec ID, or use --all")
		}

		dir := getWorkDir()
		s, err := session.LoadOrFail(dir)
		if err != nil {
			return err
		}

		if specDoneAll {
			count := s.CompleteAllSpecs()
			if count == 0 {
				return fmt.Errorf("no active specs to mark as done")
			}
			s.AddEvent("spec_done", func(e *types.Event) {
				e.SpecCount = count
			})
			if err := session.Save(dir, s); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Marked %d spec(s) as done\n", count)
			fmt.Fprintln(cmd.OutOrStdout(), "Next: add more specs or run 'tdd-ai reset' to start over")
			return nil
		}

		for _, arg := range args {
			id, err := strconv.Atoi(arg)
			if err != nil {
				return fmt.Errorf("spec ID must be a number, got %q", arg)
			}
			if err := s.CompleteSpec(id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Spec [%d] marked as done\n", id)
		}

		s.AddEvent("spec_done", func(e *types.Event) {
			e.SpecCount = len(args)
		})

		if err := session.Save(dir, s); err != nil {
			return err
		}

		if len(s.ActiveSpecs()) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Next: add more specs or run 'tdd-ai reset' to start over")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Next: run 'tdd-ai guide --format json' for phase instructions")
		}
		return nil
	},
}

func init() {
	specDoneCmd.Flags().BoolVar(&specDoneAll, "all", false, "mark all active specs as done")
	specCmd.AddCommand(specAddCmd)
	specCmd.AddCommand(specListCmd)
	specCmd.AddCommand(specDoneCmd)
	rootCmd.AddCommand(specCmd)
}
