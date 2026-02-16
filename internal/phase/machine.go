package phase

import (
	"fmt"

	"github.com/macosta/tdd-ai/internal/types"
)

// transitions defines the valid TDD phase progression for greenfield mode.
var transitions = map[types.Phase]types.Phase{
	types.PhaseRed:      types.PhaseGreen,
	types.PhaseGreen:    types.PhaseRefactor,
	types.PhaseRefactor: types.PhaseDone,
}

// retrofitTransitions defines the phase progression for retrofit mode.
// Skips green since the implementation already exists.
var retrofitTransitions = map[types.Phase]types.Phase{
	types.PhaseRed:      types.PhaseRefactor,
	types.PhaseRefactor: types.PhaseDone,
}

// Next returns the next phase in the TDD cycle (greenfield mode).
// Returns an error if the current phase has no valid next phase.
func Next(current types.Phase) (types.Phase, error) {
	return NextWithMode(current, types.ModeGreenfield)
}

// NextWithMode returns the next phase based on the session mode.
func NextWithMode(current types.Phase, mode types.Mode) (types.Phase, error) {
	t := transitions
	if mode == types.ModeRetrofit {
		t = retrofitTransitions
	}

	next, ok := t[current]
	if !ok {
		if current == types.PhaseDone {
			return "", fmt.Errorf("TDD cycle is complete; start a new session or add more specs")
		}
		return "", fmt.Errorf("unknown phase: %q", current)
	}
	return next, nil
}

// ExpectedTestResult returns what test outcome is expected before leaving the given phase.
// Returns "fail" or "pass".
func ExpectedTestResult(p types.Phase, mode types.Mode) string {
	if p == types.PhaseRed && mode != types.ModeRetrofit {
		return "fail"
	}
	return "pass"
}

// NextInLoop returns the next phase considering the per-spec loop.
// At REFACTOR: returns RED if specs remain, DONE if empty.
// For all other phases, behaves like NextWithMode.
func NextInLoop(current types.Phase, mode types.Mode, hasRemainingSpecs bool) (types.Phase, error) {
	if current == types.PhaseRefactor && hasRemainingSpecs {
		return types.PhaseRed, nil
	}
	return NextWithMode(current, mode)
}

// CanTransition checks whether moving from one phase to another is valid.
func CanTransition(from, to types.Phase) bool {
	// Standard greenfield transitions
	if next, ok := transitions[from]; ok && next == to {
		return true
	}
	// Loop transition: refactor -> red
	if from == types.PhaseRefactor && to == types.PhaseRed {
		return true
	}
	return false
}
