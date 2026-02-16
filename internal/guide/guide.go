package guide

import (
	"fmt"

	"github.com/macosta/tdd-ai/internal/phase"
	"github.com/macosta/tdd-ai/internal/types"
)

// Generate produces phase-appropriate guidance for the AI agent.
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

	switch s.Phase {
	case types.PhaseRed:
		if mode == types.ModeRetrofit {
			g.Instructions = retrofitRedInstructions(s)
			g.Rules = retrofitRedRules()
		} else {
			g.Instructions = redInstructions(s)
			g.Rules = redRules()
		}
	case types.PhaseGreen:
		if mode == types.ModeRetrofit {
			g.Instructions = retrofitGreenInstructions()
			g.Rules = retrofitGreenRules()
		} else {
			g.Instructions = greenInstructions(s)
			g.Rules = greenRules()
		}
	case types.PhaseRefactor:
		g.Instructions = refactorInstructions(s)
		g.Rules = refactorRules()
		g.Reflections = s.Reflections
	case types.PhaseDone:
		g.Instructions = doneInstructions()
		g.Rules = nil
	}

	return g
}

func redInstructions(s *types.Session) []string {
	if cs := s.CurrentSpec(); cs != nil {
		return []string{
			fmt.Sprintf("Write a failing test for spec [%d]: %s", cs.ID, cs.Description),
			"Cover happy path, edge cases, and error conditions for this spec.",
			"Run the project's test command to verify the new test FAILS.",
			"Do NOT write any implementation code yet.",
			"When the test is written and confirmed failing, run: tdd-ai test && tdd-ai phase next (test result is stored and auto-used)",
		}
	}
	return []string{
		"Pick a spec to work on: tdd-ai spec pick <id>",
		"Run 'tdd-ai spec list' to see available specs.",
		"After picking a spec, run 'tdd-ai guide' again for specific instructions.",
	}
}

func redRules() []string {
	return []string{
		"DO NOT create implementation files.",
		"DO NOT write skeleton or stub implementations.",
		"Tests must assert specific expected values, not just 'does not throw'.",
	}
}

func greenInstructions(s *types.Session) []string {
	instructions := []string{}
	if cs := s.CurrentSpec(); cs != nil {
		instructions = append(instructions,
			fmt.Sprintf("Write the MINIMAL code to make the test for spec [%d] pass: %s", cs.ID, cs.Description),
		)
	} else {
		instructions = append(instructions, "Write the MINIMAL code to make all failing tests pass.")
	}
	instructions = append(instructions,
		"Run tests after each change.",
		"Do NOT modify any test files.",
		"Do NOT add functionality beyond what the tests require.",
		"When all tests pass, run: tdd-ai test && tdd-ai phase next (test result is stored and auto-used)",
	)
	return instructions
}

func greenRules() []string {
	return []string{
		"DO NOT modify test files.",
		"DO NOT add features not covered by existing tests.",
		"Prefer the simplest implementation that passes.",
	}
}

func refactorInstructions(s *types.Session) []string {
	instructions := []string{
		"Improve code quality: naming, structure, DRY, performance.",
		"Run tests after EVERY change to ensure they still pass.",
		"Do NOT add new functionality.",
		"Do NOT modify test assertions.",
	}

	remaining := s.RemainingSpecs()
	if len(remaining) > 0 {
		instructions = append(instructions,
			fmt.Sprintf("%d spec(s) remaining after this one. Discover new scenarios? Add them: tdd-ai spec add \"new scenario\"", len(remaining)),
		)
	}

	if len(s.Reflections) > 0 && !s.AllReflectionsAnswered() {
		pending := s.PendingReflections()
		instructions = append(instructions,
			fmt.Sprintf("REQUIRED: Answer all %d reflection questions before advancing. %d remaining.", len(s.Reflections), len(pending)),
			"View questions: tdd-ai refactor status",
			"Answer a question: tdd-ai refactor reflect <number> --answer \"your response\"",
		)
	} else if len(s.Reflections) > 0 && s.AllReflectionsAnswered() {
		instructions = append(instructions, "All reflection questions answered. Ready to advance.")
	}

	if len(remaining) > 0 {
		instructions = append(instructions,
			"When satisfied with code quality, run: tdd-ai test && tdd-ai phase next (loops back to RED for the next spec)",
		)
	} else {
		instructions = append(instructions,
			"When satisfied with code quality, run: tdd-ai test && tdd-ai phase next (test result is stored and auto-used)",
		)
	}

	instructions = append(instructions,
		"Or finish the entire cycle in one step: tdd-ai complete (uses cached test result if available, or pass --test-result pass to skip re-running tests)",
	)

	return instructions
}

func refactorRules() []string {
	return []string{
		"Tests must pass after every refactor step.",
		"DO NOT change test expectations.",
		"DO NOT add new features during refactor.",
	}
}

func retrofitRedInstructions(s *types.Session) []string {
	if cs := s.CurrentSpec(); cs != nil {
		return []string{
			fmt.Sprintf("Write a NEW test for spec [%d]: %s", cs.ID, cs.Description),
			"Do NOT rely on pre-existing tests to cover this spec — write explicit new tests even if similar coverage exists.",
			"Run the project's test command to verify the new test PASSES against the existing implementation.",
			"If tests fail, determine whether the test is wrong or the implementation has a bug.",
			"When the test is written and confirmed passing, run: tdd-ai test && tdd-ai phase next (test result is stored and auto-used)",
			"Note: After tests pass, the next phase is REFACTOR (GREEN is skipped since implementation exists).",
		}
	}
	return []string{
		"Pick a spec to work on: tdd-ai spec pick <id>",
		"Run 'tdd-ai spec list' to see available specs.",
		"After picking a spec, run 'tdd-ai guide' again for specific instructions.",
	}
}

func retrofitRedRules() []string {
	return []string{
		"DO NOT modify the existing implementation.",
		"Tests must assert specific expected values, not just 'does not throw'.",
		"Tests should document existing behavior, not desired behavior.",
	}
}

func retrofitGreenInstructions() []string {
	return []string{
		"Implementation already exists — this phase is typically skipped in retrofit mode.",
		"If you reached this phase, consider running: tdd-ai test && tdd-ai phase next (test result is stored and auto-used)",
		"Only make changes if tests revealed bugs in the existing implementation.",
	}
}

func retrofitGreenRules() []string {
	return []string{
		"DO NOT modify test files.",
		"Only fix bugs discovered during the red phase.",
		"Prefer minimal changes to the existing implementation.",
	}
}

func doneInstructions() []string {
	return []string{
		"TDD cycle is complete.",
		"Mark completed specs with: tdd-ai spec done <id> or tdd-ai spec done --all",
		"Or finish all at once: tdd-ai complete",
		"To start a new cycle, add more specs and run: tdd-ai phase set red",
	}
}
