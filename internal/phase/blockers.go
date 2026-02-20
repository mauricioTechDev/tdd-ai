package phase

import (
	"fmt"

	"github.com/macosta/tdd-ai/internal/types"
)

// GetBlockers returns conditions preventing advancement from the current phase.
func GetBlockers(s *types.Session) []string {
	var blockers []string
	mode := s.GetMode()

	switch s.Phase {
	case types.PhaseRed:
		if len(s.ActiveSpecs()) == 0 {
			blockers = append(blockers, "No active specs")
		}
		if s.CurrentSpecID == nil && len(s.ActiveSpecs()) > 0 {
			blockers = append(blockers, "No spec selected")
		}
		if s.LastTestResult == "" {
			blockers = append(blockers, "No test result recorded")
		} else {
			expected := ExpectedTestResult(s.Phase, mode)
			if s.LastTestResult != expected {
				blockers = append(blockers,
					fmt.Sprintf("Test result '%s' does not match expected '%s'", s.LastTestResult, expected),
				)
			}
		}
	case types.PhaseGreen:
		if s.LastTestResult == "" {
			blockers = append(blockers, "No test result recorded")
		} else {
			expected := ExpectedTestResult(s.Phase, mode)
			if s.LastTestResult != expected {
				blockers = append(blockers,
					fmt.Sprintf("Test result '%s' does not match expected '%s'", s.LastTestResult, expected),
				)
			}
		}
	case types.PhaseRefactor:
		if s.LastTestResult == "" {
			blockers = append(blockers, "No test result recorded")
		} else {
			expected := ExpectedTestResult(s.Phase, mode)
			if s.LastTestResult != expected {
				blockers = append(blockers,
					fmt.Sprintf("Test result '%s' does not match expected '%s'", s.LastTestResult, expected),
				)
			}
		}
		pending := s.PendingReflections()
		if len(pending) > 0 {
			blockers = append(blockers,
				fmt.Sprintf("%d reflection questions unanswered", len(pending)),
			)
		}
	case types.PhaseDone:
		blockers = append(blockers, "Cannot advance past done")
	}

	return blockers
}
