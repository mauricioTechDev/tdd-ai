package formatter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/macosta/tdd-ai/internal/types"
)

// sortSpecsByID returns a copy of specs sorted by ID ascending.
func sortSpecsByID(specs []types.Spec) []types.Spec {
	sorted := make([]types.Spec, len(specs))
	copy(sorted, specs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

// Format specifies the output format.
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// FormatGuidance renders guidance in the specified format.
func FormatGuidance(g types.Guidance, f Format) (string, error) {
	switch f {
	case FormatJSON:
		return formatJSON(g)
	case FormatText:
		return formatText(g), nil
	default:
		return "", fmt.Errorf("unknown format: %q", f)
	}
}

func formatJSON(g types.Guidance) (string, error) {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding guidance: %w", err)
	}
	return string(data), nil
}

func formatText(g types.Guidance) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Phase: %s\n", strings.ToUpper(g.Phase.String()))
	fmt.Fprintf(&b, "Mode: %s\n", g.Mode)
	if g.NextPhase != "" {
		fmt.Fprintf(&b, "Next Phase: %s\n", strings.ToUpper(g.NextPhase.String()))
	}
	if g.TestCmd != "" {
		fmt.Fprintf(&b, "Test Command: %s\n", g.TestCmd)
	}
	if g.CurrentSpec != nil {
		fmt.Fprintf(&b, "Current Spec: [%d] %s\n", g.CurrentSpec.ID, g.CurrentSpec.Description)
	}
	if g.Iteration > 0 {
		fmt.Fprintf(&b, "Iteration: %d\n", g.Iteration)
	}
	b.WriteString("\n")

	if len(g.Specs) > 0 {
		b.WriteString("Active Specs:\n")
		for _, s := range sortSpecsByID(g.Specs) {
			fmt.Fprintf(&b, "  [%d] %s\n", s.ID, s.Description)
		}
		b.WriteString("\n")
	}

	if len(g.Instructions) > 0 {
		b.WriteString("Instructions:\n")
		for _, inst := range g.Instructions {
			fmt.Fprintf(&b, "  - %s\n", inst)
		}
		b.WriteString("\n")
	}

	if len(g.Rules) > 0 {
		b.WriteString("Rules:\n")
		for _, rule := range g.Rules {
			fmt.Fprintf(&b, "  - %s\n", rule)
		}
		b.WriteString("\n")
	}

	if len(g.Reflections) > 0 {
		answered := 0
		for _, r := range g.Reflections {
			if r.Answer != "" {
				answered++
			}
		}
		fmt.Fprintf(&b, "Reflections (%d/%d answered):\n", answered, len(g.Reflections))
		for _, r := range g.Reflections {
			status := "pending"
			if r.Answer != "" {
				status = "answered"
			}
			fmt.Fprintf(&b, "  [%d] (%s) %s\n", r.ID, status, r.Question)
			if r.Answer != "" {
				fmt.Fprintf(&b, "      -> %q\n", r.Answer)
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// nextAction returns a recommended next step based on session state.
func nextAction(s *types.Session) string {
	if len(s.Specs) == 0 {
		return "Next: add specs with 'tdd-ai spec add \"desc1\" \"desc2\" ...'"
	}
	if s.Phase == types.PhaseDone {
		if len(s.ActiveSpecs()) == 0 {
			return "Next: all specs complete. Add more specs or run 'tdd-ai reset' to start over"
		}
		return "Next: mark completed specs with 'tdd-ai spec done --all' or 'tdd-ai spec done <id>'"
	}
	return "Next: run 'tdd-ai guide --format json' for phase instructions"
}

// FormatFullStatus renders a rich session overview.
func FormatFullStatus(s *types.Session, f Format) (string, error) {
	type fullStatusOutput struct {
		Phase         types.Phase   `json:"phase"`
		Mode          string        `json:"mode"`
		TestCmd       string        `json:"test_cmd,omitempty"`
		CurrentSpecID *int          `json:"current_spec_id,omitempty"`
		Iteration     int           `json:"iteration,omitempty"`
		TotalSpecs    int           `json:"total_specs"`
		ActiveSpecs   int           `json:"active_specs"`
		DoneSpecs     int           `json:"done_specs"`
		Specs         []types.Spec  `json:"specs"`
		History       []types.Event `json:"history,omitempty"`
		NextAction    string        `json:"next_action"`
	}

	active := s.ActiveSpecs()
	mode := s.GetMode()

	out := fullStatusOutput{
		Phase:         s.Phase,
		Mode:          string(mode),
		TestCmd:       s.TestCmd,
		CurrentSpecID: s.CurrentSpecID,
		Iteration:     s.Iteration,
		TotalSpecs:    len(s.Specs),
		ActiveSpecs:   len(active),
		DoneSpecs:     len(s.Specs) - len(active),
		Specs:         s.Specs,
		History:       s.History,
		NextAction:    nextAction(s),
	}

	switch f {
	case FormatJSON:
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	case FormatText:
		var b strings.Builder
		fmt.Fprintf(&b, "Phase: %s\n", strings.ToUpper(string(s.Phase)))
		fmt.Fprintf(&b, "Mode: %s\n", mode)
		if s.TestCmd != "" {
			fmt.Fprintf(&b, "Test Command: %s\n", s.TestCmd)
		}
		if cs := s.CurrentSpec(); cs != nil {
			fmt.Fprintf(&b, "Current Spec: [%d] %s\n", cs.ID, cs.Description)
		}
		if s.Iteration > 0 {
			fmt.Fprintf(&b, "Iteration: %d\n", s.Iteration)
		}
		fmt.Fprintf(&b, "Specs: %d total, %d active, %d done\n\n", out.TotalSpecs, out.ActiveSpecs, out.DoneSpecs)
		for _, spec := range sortSpecsByID(s.Specs) {
			status := "active"
			if spec.Status == types.SpecStatusCompleted {
				status = "done"
			}
			fmt.Fprintf(&b, "  [%d] (%s) %s\n", spec.ID, status, spec.Description)
		}
		if len(s.Specs) > 0 {
			b.WriteString("\n")
		}
		if len(s.History) > 0 {
			b.WriteString("History:\n")
			for _, ev := range s.History {
				line := fmt.Sprintf("  %s: %s", ev.Timestamp, ev.Action)
				if ev.From != "" && ev.To != "" {
					line += fmt.Sprintf(" (%s -> %s)", ev.From, ev.To)
				}
				if ev.Result != "" {
					line += fmt.Sprintf(" [%s]", ev.Result)
				}
				if ev.SpecCount > 0 {
					line += fmt.Sprintf(" (%d specs)", ev.SpecCount)
				}
				fmt.Fprintln(&b, line)
			}
			b.WriteString("\n")
		}
		fmt.Fprintln(&b, out.NextAction)
		return b.String(), nil
	default:
		return "", fmt.Errorf("unknown format: %q", f)
	}
}

// FormatStatus renders a simple session status.
func FormatStatus(s *types.Session, f Format) (string, error) {
	type statusOutput struct {
		Phase       types.Phase  `json:"phase"`
		TotalSpecs  int          `json:"total_specs"`
		ActiveSpecs int          `json:"active_specs"`
		DoneSpecs   int          `json:"done_specs"`
		Specs       []types.Spec `json:"specs"`
	}

	active := s.ActiveSpecs()
	out := statusOutput{
		Phase:       s.Phase,
		TotalSpecs:  len(s.Specs),
		ActiveSpecs: len(active),
		DoneSpecs:   len(s.Specs) - len(active),
		Specs:       s.Specs,
	}

	switch f {
	case FormatJSON:
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	case FormatText:
		var b strings.Builder
		fmt.Fprintf(&b, "Phase: %s\n", strings.ToUpper(string(s.Phase)))
		fmt.Fprintf(&b, "Specs: %d total, %d active, %d done\n\n", out.TotalSpecs, out.ActiveSpecs, out.DoneSpecs)
		for _, spec := range sortSpecsByID(s.Specs) {
			status := "active"
			if spec.Status == types.SpecStatusCompleted {
				status = "done"
			}
			isCurrent := s.CurrentSpecID != nil && spec.ID == *s.CurrentSpecID
			if isCurrent {
				fmt.Fprintf(&b, "→ [%d] (%s) %s (current)\n", spec.ID, status, spec.Description)
			} else {
				fmt.Fprintf(&b, "  [%d] (%s) %s\n", spec.ID, status, spec.Description)
			}
		}
		if len(s.Specs) > 0 {
			b.WriteString("\n")
		}
		return b.String(), nil
	default:
		return "", fmt.Errorf("unknown format: %q", f)
	}
}
