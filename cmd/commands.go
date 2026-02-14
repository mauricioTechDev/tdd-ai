package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/macosta/tdd-ai/internal/formatter"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// commandFlag describes a single CLI flag.
type commandFlag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     string `json:"default"`
	Description string `json:"description"`
}

// commandEntry describes a single CLI command (flattened, e.g. "spec add").
type commandEntry struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Usage       string        `json:"usage"`
	Flags       []commandFlag `json:"flags"`
}

// sessionSummary is a snapshot of session state for the commands output.
type sessionSummary struct {
	Active      bool   `json:"active"`
	Phase       string `json:"phase"`
	Mode        string `json:"mode"`
	TestCmd     string `json:"test_cmd,omitempty"`
	TotalSpecs  int    `json:"total_specs"`
	ActiveSpecs int    `json:"active_specs"`
	DoneSpecs   int    `json:"done_specs"`
}

// commandsOutput is the top-level structure for tdd-ai commands.
type commandsOutput struct {
	Version     string          `json:"version"`
	Workflow    []string        `json:"workflow"`
	Commands    []commandEntry  `json:"commands"`
	GlobalFlags []commandFlag   `json:"global_flags"`
	Session     *sessionSummary `json:"session"`
}

var commandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "Show all commands, flags, and workflow in one call",
	Long: `Dumps the entire CLI reference: all commands with their flags, the recommended
workflow, global flags, and current session state (if any).

Designed for AI agents to learn the full API in a single call.`,
	Example: `  tdd-ai commands
  tdd-ai commands --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output := buildCommandsOutput()

		f := formatter.Format(formatFlag)
		switch f {
		case formatter.FormatJSON:
			data, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return fmt.Errorf("encoding commands: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
		case formatter.FormatText:
			fmt.Fprint(cmd.OutOrStdout(), formatCommandsText(output))
		default:
			return fmt.Errorf("unknown format: %q", f)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commandsCmd)
}

func buildCommandsOutput() commandsOutput {
	output := commandsOutput{
		Version:  version,
		Workflow: workflowSteps(),
		Commands: buildCommandList(rootCmd),
	}

	// Global persistent flags
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		output.GlobalFlags = append(output.GlobalFlags, commandFlag{
			Name:        "--" + f.Name,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Description: f.Usage,
		})
	})

	// Session state (soft load, no error if missing)
	dir := getWorkDir()
	if session.Exists(dir) {
		if s, err := session.Load(dir); err == nil {
			active := s.ActiveSpecs()
			output.Session = &sessionSummary{
				Active:      true,
				Phase:       string(s.Phase),
				Mode:        string(s.GetMode()),
				TestCmd:     s.TestCmd,
				TotalSpecs:  len(s.Specs),
				ActiveSpecs: len(active),
				DoneSpecs:   len(s.Specs) - len(active),
			}
		}
	}

	return output
}

func workflowSteps() []string {
	return []string{
		"1. tdd-ai init [--retrofit] [--test-cmd \"...\"]",
		"2. tdd-ai spec add \"desc1\" \"desc2\" ...",
		"3. tdd-ai guide (get phase instructions)",
		"4. Write code following the instructions",
		"5. tdd-ai test (run tests and record result)",
		"6. tdd-ai phase next (advance when phase criteria met)",
		"7. In refactor: tdd-ai refactor reflect <n> --answer \"...\" (answer all reflection questions)",
		"8. Repeat steps 3-7 for red -> green -> refactor",
		"9. tdd-ai complete (finish cycle, mark all specs done)",
	}
}

func buildCommandList(root *cobra.Command) []commandEntry {
	var cmds []commandEntry
	for _, c := range root.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "commands" || c.Name() == "completion" {
			continue
		}
		subs := c.Commands()
		if len(subs) > 0 {
			// Parent with subcommands (spec, phase): include parent if it has a run function
			if c.RunE != nil || c.Run != nil {
				cmds = append(cmds, entryFrom(c, ""))
			}
			for _, sub := range subs {
				if !sub.Hidden && sub.Name() != "help" {
					cmds = append(cmds, entryFrom(sub, c.Name()))
				}
			}
		} else {
			cmds = append(cmds, entryFrom(c, ""))
		}
	}
	return cmds
}

func entryFrom(c *cobra.Command, parentName string) commandEntry {
	name := c.Name()
	usage := "tdd-ai " + name
	if parentName != "" {
		name = parentName + " " + name
		usage = "tdd-ai " + name
	}

	// Append args placeholder from Use field
	parts := strings.Fields(c.Use)
	if len(parts) > 1 {
		usage += " " + strings.Join(parts[1:], " ")
	}

	entry := commandEntry{
		Name:        name,
		Description: c.Short,
		Usage:       usage,
		Flags:       []commandFlag{},
	}

	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		entry.Flags = append(entry.Flags, commandFlag{
			Name:        "--" + f.Name,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Description: f.Usage,
		})
	})

	return entry
}

func formatCommandsText(output commandsOutput) string {
	var b strings.Builder

	fmt.Fprintf(&b, "tdd-ai %s - TDD guardrails for AI coding agents\n\n", output.Version)

	// Workflow
	b.WriteString("Workflow:\n")
	for _, step := range output.Workflow {
		fmt.Fprintf(&b, "  %s\n", step)
	}
	b.WriteString("\n")

	// Commands
	b.WriteString("Commands:\n")
	for _, cmd := range output.Commands {
		fmt.Fprintf(&b, "  %-18s%s\n", cmd.Name, cmd.Description)
		for _, f := range cmd.Flags {
			flagDisplay := f.Name
			if f.Type != "bool" {
				flagDisplay += " " + strings.ToUpper(f.Type)
			}
			fmt.Fprintf(&b, "  %-18s  %-20s%s\n", "", flagDisplay, f.Description)
		}
	}
	b.WriteString("\n")

	// Global flags
	b.WriteString("Global flags:\n")
	for _, f := range output.GlobalFlags {
		flagDisplay := f.Name
		if f.Type != "bool" {
			flagDisplay += " " + strings.ToUpper(f.Type)
		}
		fmt.Fprintf(&b, "  %-20s%s (default: %s)\n", flagDisplay, f.Description, f.Default)
	}
	b.WriteString("\n")

	// Session
	if output.Session != nil {
		fmt.Fprintf(&b, "Session: %s phase, %s mode, %d/%d specs active\n",
			output.Session.Phase, output.Session.Mode,
			output.Session.ActiveSpecs, output.Session.TotalSpecs)
	} else {
		b.WriteString("Session: none (run 'tdd-ai init' to start)\n")
	}

	return b.String()
}
