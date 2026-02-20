package guide

import (
	"github.com/macosta/tdd-ai/internal/phase"
	"github.com/macosta/tdd-ai/internal/types"
)

// Generate produces state-only guidance for the current TDD session.
func Generate(s *types.Session) types.Guidance {
	mode := s.GetMode()

	g := types.Guidance{
		Phase:      s.Phase,
		Mode:       mode,
		TestCmd:    s.TestCmd,
		Specs:      s.ActiveSpecs(),
		Iteration:  s.Iteration,
		TotalSpecs: len(s.Specs),
	}

	// Populate current spec if one is selected
	if cs := s.CurrentSpec(); cs != nil {
		g.CurrentSpec = cs
	}

	// Compute next phase from the state machine (ignore error for done/invalid)
	if next, err := phase.NextWithMode(s.Phase, mode); err == nil {
		g.NextPhase = next
	}

	// Expected test result for the current phase
	if s.Phase != types.PhaseDone {
		g.ExpectedTestResult = phase.ExpectedTestResult(s.Phase, mode)
	}

	// Blockers preventing advancement
	g.Blockers = phase.GetBlockers(s)

	// Include reflections during refactor phase
	if s.Phase == types.PhaseRefactor {
		g.Reflections = s.Reflections
	}

	return g
}
