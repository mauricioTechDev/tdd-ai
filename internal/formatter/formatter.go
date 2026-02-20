package formatter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/macosta/tdd-ai/internal/phase"
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
	if g.ExpectedTestResult != "" {
		fmt.Fprintf(&b, "Expected Test Result: %s\n", g.ExpectedTestResult)
	}
	b.WriteString("\n")

	if len(g.Specs) > 0 {
		b.WriteString("Active Specs:\n")
		for _, s := range sortSpecsByID(g.Specs) {
			fmt.Fprintf(&b, "  [%d] %s\n", s.ID, s.Description)
		}
		b.WriteString("\n")
	}

	if len(g.Blockers) > 0 {
		b.WriteString("Blockers:\n")
		for _, bl := range g.Blockers {
			fmt.Fprintf(&b, "  - %s\n", bl)
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
		return b.String(), nil
	default:
		return "", fmt.Errorf("unknown format: %q", f)
	}
}

// resumeNextAction returns the single most important next action for context recovery.
func resumeNextAction(s *types.Session) string {
	if len(s.Specs) == 0 {
		return `tdd-ai spec add "desc1" "desc2" ...`
	}
	switch s.Phase {
	case types.PhaseDone:
		if len(s.ActiveSpecs()) > 0 {
			return "tdd-ai spec done --all"
		}
		return `All specs complete. Add more specs: tdd-ai spec add "desc1" ...`
	case types.PhaseRed:
		if s.CurrentSpecID == nil {
			active := s.ActiveSpecs()
			if len(active) > 0 {
				return fmt.Sprintf("tdd-ai spec pick %d", active[0].ID)
			}
		}
		return "tdd-ai test && tdd-ai phase next"
	case types.PhaseGreen:
		return "tdd-ai test && tdd-ai phase next"
	case types.PhaseRefactor:
		pending := s.PendingReflections()
		if len(pending) > 0 {
			return fmt.Sprintf(`tdd-ai refactor reflect %d --answer "your answer here"`, pending[0].ID)
		}
		return "tdd-ai test && tdd-ai phase next"
	}
	return "tdd-ai guide"
}

// recentHistory returns the last n events from the session history.
func recentHistory(s *types.Session, n int) []types.Event {
	if len(s.History) <= n {
		return s.History
	}
	return s.History[len(s.History)-n:]
}

// FormatResume renders a compact session checkpoint for agent context recovery.
// Designed to be run after context compression or by a new sub-agent to quickly
// re-orient to the current TDD session state without reading the full history.
func FormatResume(s *types.Session, f Format) (string, error) {
	type resumeOutput struct {
		Phase          types.Phase   `json:"phase"`
		Mode           types.Mode    `json:"mode"`
		TestCmd        string        `json:"test_cmd,omitempty"`
		Iteration      int           `json:"iteration,omitempty"`
		CurrentSpec    *types.Spec   `json:"current_spec,omitempty"`
		RemainingSpecs int           `json:"remaining_specs"`
		Blockers       []string      `json:"blockers,omitempty"`
		NextAction     string        `json:"next_action"`
		RecentEvents   []types.Event `json:"recent_events,omitempty"`
	}

	remaining := len(s.RemainingSpecs())
	blockers := phase.GetBlockers(s)
	recent := recentHistory(s, 5)

	out := resumeOutput{
		Phase:          s.Phase,
		Mode:           s.GetMode(),
		TestCmd:        s.TestCmd,
		Iteration:      s.Iteration,
		RemainingSpecs: remaining,
		Blockers:       blockers,
		NextAction:     resumeNextAction(s),
		RecentEvents:   recent,
	}
	if cs := s.CurrentSpec(); cs != nil {
		out.CurrentSpec = cs
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
		b.WriteString("=== TDD Session Checkpoint ===\n")
		fmt.Fprintf(&b, "Phase: %s | Mode: %s", strings.ToUpper(string(s.Phase)), s.GetMode())
		if s.Iteration > 0 {
			fmt.Fprintf(&b, " | Iteration: %d", s.Iteration)
		}
		b.WriteString("\n")
		if cs := s.CurrentSpec(); cs != nil {
			fmt.Fprintf(&b, "Working on: [%d] %s\n", cs.ID, cs.Description)
		} else if s.Phase == types.PhaseRed && len(s.ActiveSpecs()) > 0 {
			b.WriteString("Working on: (no spec selected)\n")
		}
		if remaining > 0 {
			fmt.Fprintf(&b, "Remaining specs: %d\n", remaining)
		}
		b.WriteString("\n")
		if len(blockers) > 0 {
			b.WriteString("BLOCKERS:\n")
			for _, bl := range blockers {
				fmt.Fprintf(&b, "  - %s\n", bl)
			}
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "NEXT ACTION:\n  %s\n", out.NextAction)
		if len(recent) > 0 {
			b.WriteString("\nRecent events:\n")
			for _, ev := range recent {
				line := "  " + ev.Action
				if ev.From != "" && ev.To != "" {
					line += fmt.Sprintf(" (%s -> %s)", ev.From, ev.To)
				}
				if ev.Result != "" {
					line += fmt.Sprintf(" [%s]", ev.Result)
				}
				fmt.Fprintln(&b, line)
			}
		}
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
