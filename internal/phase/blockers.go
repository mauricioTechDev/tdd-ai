package phase

import (
	"fmt"

	"github.com/macosta/tdd-ai/internal/types"
)

// checkTestResult returns a blocker if the test result is missing or doesn't match
// the expected result for the given phase and mode.
func checkTestResult(s *types.Session, phase types.Phase, mode types.Mode) []string {
	if s.LastTestResult == "" {
		return []string{"No test result recorded"}
	}
	expected := ExpectedTestResult(phase, mode)
	if s.LastTestResult != expected {
		return []string{
			fmt.Sprintf("Test result '%s' does not match expected '%s'", s.LastTestResult, expected),
		}
	}
	return nil
}

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
		blockers = append(blockers, checkTestResult(s, s.Phase, mode)...)
	case types.PhaseGreen:
		blockers = append(blockers, checkTestResult(s, s.Phase, mode)...)
	case types.PhaseRefactor:
		blockers = append(blockers, checkTestResult(s, s.Phase, mode)...)
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
