package guide

import (
	"testing"

	"github.com/macosta/tdd-ai/internal/types"
)

func TestGenerateRedPhase(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("calculate shipping cost")

	g := Generate(s)

	if g.Phase != types.PhaseRed {
		t.Errorf("guidance phase = %q, want %q", g.Phase, types.PhaseRed)
	}
	if g.Mode != types.ModeGreenfield {
		t.Errorf("guidance mode = %q, want %q", g.Mode, types.ModeGreenfield)
	}
	if len(g.Instructions) == 0 {
		t.Error("red phase should have instructions")
	}
	if len(g.Rules) == 0 {
		t.Error("red phase should have rules")
	}
	if len(g.Specs) != 1 {
		t.Errorf("guidance specs length = %d, want 1", len(g.Specs))
	}
}

func TestGenerateGreenPhase(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("calculate shipping cost")
	s.Phase = types.PhaseGreen

	g := Generate(s)

	if g.Phase != types.PhaseGreen {
		t.Errorf("guidance phase = %q, want %q", g.Phase, types.PhaseGreen)
	}
	if len(g.Instructions) == 0 {
		t.Error("green phase should have instructions")
	}
	if len(g.Rules) == 0 {
		t.Error("green phase should have rules")
	}
}

func TestGenerateRefactorPhase(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor

	g := Generate(s)

	if g.Phase != types.PhaseRefactor {
		t.Errorf("guidance phase = %q, want %q", g.Phase, types.PhaseRefactor)
	}
	if len(g.Instructions) == 0 {
		t.Error("refactor phase should have instructions")
	}
	if len(g.Rules) == 0 {
		t.Error("refactor phase should have rules")
	}
}

func TestGenerateDonePhase(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseDone

	g := Generate(s)

	if g.Phase != types.PhaseDone {
		t.Errorf("guidance phase = %q, want %q", g.Phase, types.PhaseDone)
	}
	if len(g.Instructions) == 0 {
		t.Error("done phase should have instructions")
	}
	if g.Rules != nil {
		t.Error("done phase should have no rules")
	}
}

func TestGenerateRetrofitRedPhase(t *testing.T) {
	s := types.NewSession()
	s.Mode = types.ModeRetrofit
	s.AddSpec("GET /users returns 200")

	g := Generate(s)

	if g.Phase != types.PhaseRed {
		t.Errorf("guidance phase = %q, want %q", g.Phase, types.PhaseRed)
	}
	if g.Mode != types.ModeRetrofit {
		t.Errorf("guidance mode = %q, want %q", g.Mode, types.ModeRetrofit)
	}
	if len(g.Instructions) == 0 {
		t.Fatal("retrofit red phase should have instructions")
	}

	// Retrofit red instructions should mention writing NEW tests per spec
	foundNewTest := false
	foundPass := false
	foundGreenSkipped := false
	foundEachSpec := false
	for _, inst := range g.Instructions {
		if contains(inst, "NEW test") {
			foundNewTest = true
		}
		if contains(inst, "EACH") && contains(inst, "spec") {
			foundEachSpec = true
		}
		if contains(inst, "PASS") {
			foundPass = true
		}
		if contains(inst, "GREEN is skipped") {
			foundGreenSkipped = true
		}
	}
	if !foundNewTest {
		t.Error("retrofit red instructions should mention writing 'NEW test'")
	}
	if !foundEachSpec {
		t.Error("retrofit red instructions should mention 'EACH' spec")
	}
	if !foundPass {
		t.Error("retrofit red instructions should mention tests PASS (not fail)")
	}
	if !foundGreenSkipped {
		t.Error("retrofit red instructions should mention that GREEN is skipped")
	}
}

func TestGenerateRetrofitGreenPhase(t *testing.T) {
	s := types.NewSession()
	s.Mode = types.ModeRetrofit
	s.Phase = types.PhaseGreen

	g := Generate(s)

	// In retrofit, green should indicate implementation exists
	if len(g.Instructions) == 0 {
		t.Error("retrofit green phase should have instructions")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGenerateIncludesNextPhase(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature")

	g := Generate(s)

	// Red phase in greenfield -> next is green
	if g.NextPhase != types.PhaseGreen {
		t.Errorf("next_phase = %q, want %q", g.NextPhase, types.PhaseGreen)
	}
}

func TestGenerateRetrofitNextPhaseSkipsGreen(t *testing.T) {
	s := types.NewSession()
	s.Mode = types.ModeRetrofit
	s.AddSpec("existing feature")

	g := Generate(s)

	// Red phase in retrofit -> next is refactor (skips green)
	if g.NextPhase != types.PhaseRefactor {
		t.Errorf("next_phase = %q, want %q", g.NextPhase, types.PhaseRefactor)
	}
}

func TestGenerateDonePhaseHasNoNextPhase(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseDone

	g := Generate(s)

	if g.NextPhase != "" {
		t.Errorf("next_phase should be empty for done phase, got %q", g.NextPhase)
	}
}

func TestGenerateIncludesTestCmd(t *testing.T) {
	s := types.NewSession()
	s.TestCmd = "go test ./..."
	s.AddSpec("feature")

	g := Generate(s)

	if g.TestCmd != "go test ./..." {
		t.Errorf("test_cmd = %q, want %q", g.TestCmd, "go test ./...")
	}
}

func TestGenerateOmitsTestCmdWhenEmpty(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature")

	g := Generate(s)

	if g.TestCmd != "" {
		t.Errorf("test_cmd should be empty when not configured, got %q", g.TestCmd)
	}
}

func TestRedInstructionsMentionAutoConsume(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature")

	g := Generate(s)

	found := false
	for _, inst := range g.Instructions {
		if contains(inst, "tdd-ai test") && contains(inst, "auto-used") {
			found = true
			break
		}
	}
	if !found {
		t.Error("red phase instructions should mention 'tdd-ai test' with auto-used result")
	}
}

func TestGreenInstructionsMentionAutoConsume(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseGreen
	s.AddSpec("feature")

	g := Generate(s)

	found := false
	for _, inst := range g.Instructions {
		if contains(inst, "tdd-ai test") && contains(inst, "auto-used") {
			found = true
			break
		}
	}
	if !found {
		t.Error("green phase instructions should mention 'tdd-ai test' with auto-used result")
	}
}

func TestRefactorInstructionsMentionComplete(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor

	g := Generate(s)

	foundComplete := false
	foundAutoConsume := false
	foundTestResultShortcut := false
	for _, inst := range g.Instructions {
		if contains(inst, "tdd-ai complete") {
			foundComplete = true
		}
		if contains(inst, "tdd-ai test") && contains(inst, "auto-used") {
			foundAutoConsume = true
		}
		if contains(inst, "--test-result pass") {
			foundTestResultShortcut = true
		}
	}
	if !foundComplete {
		t.Error("refactor phase instructions should mention 'tdd-ai complete'")
	}
	if !foundAutoConsume {
		t.Error("refactor phase instructions should mention 'tdd-ai test' with auto-used result")
	}
	if !foundTestResultShortcut {
		t.Error("refactor phase instructions should mention '--test-result pass' shortcut")
	}
}

func TestDoneInstructionsMentionComplete(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseDone

	g := Generate(s)

	foundComplete := false
	foundDoneAll := false
	for _, inst := range g.Instructions {
		if contains(inst, "tdd-ai complete") {
			foundComplete = true
		}
		if contains(inst, "spec done --all") {
			foundDoneAll = true
		}
	}
	if !foundComplete {
		t.Error("done phase instructions should mention 'tdd-ai complete'")
	}
	if !foundDoneAll {
		t.Error("done phase instructions should mention 'tdd-ai spec done --all'")
	}
}

func TestRetrofitRedInstructionsMentionAutoConsume(t *testing.T) {
	s := types.NewSession()
	s.Mode = types.ModeRetrofit
	s.AddSpec("existing feature")

	g := Generate(s)

	found := false
	for _, inst := range g.Instructions {
		if contains(inst, "tdd-ai test") && contains(inst, "auto-used") {
			found = true
			break
		}
	}
	if !found {
		t.Error("retrofit red phase instructions should mention 'tdd-ai test' with auto-used result")
	}
}

func TestGenerateRefactorPhaseWithPendingReflections(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.Reflections = []types.ReflectionQuestion{
		{ID: 1, Question: "Q1", Answer: ""},
		{ID: 2, Question: "Q2", Answer: ""},
	}

	g := Generate(s)

	if len(g.Reflections) != 2 {
		t.Errorf("guidance reflections length = %d, want 2", len(g.Reflections))
	}

	foundRequired := false
	foundViewCmd := false
	foundAnswerCmd := false
	for _, inst := range g.Instructions {
		if contains(inst, "REQUIRED") && contains(inst, "reflection") {
			foundRequired = true
		}
		if contains(inst, "tdd-ai refactor status") {
			foundViewCmd = true
		}
		if contains(inst, "tdd-ai refactor reflect") {
			foundAnswerCmd = true
		}
	}
	if !foundRequired {
		t.Error("refactor instructions should mention REQUIRED reflection questions when pending")
	}
	if !foundViewCmd {
		t.Error("refactor instructions should mention 'tdd-ai refactor status' when pending")
	}
	if !foundAnswerCmd {
		t.Error("refactor instructions should mention 'tdd-ai refactor reflect' when pending")
	}
}

func TestGenerateRefactorPhaseAllReflectionsAnswered(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.Reflections = []types.ReflectionQuestion{
		{ID: 1, Question: "Q1", Answer: "answered with enough words here"},
		{ID: 2, Question: "Q2", Answer: "also answered with enough words here"},
	}

	g := Generate(s)

	foundReady := false
	for _, inst := range g.Instructions {
		if contains(inst, "All reflection questions answered") {
			foundReady = true
		}
	}
	if !foundReady {
		t.Error("refactor instructions should say 'All reflection questions answered' when all answered")
	}
}

func TestGenerateRefactorPhaseNoReflections(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	// No reflections loaded (backward compat)

	g := Generate(s)

	if len(g.Reflections) != 0 {
		t.Errorf("guidance reflections should be empty when not loaded, got %d", len(g.Reflections))
	}

	for _, inst := range g.Instructions {
		if contains(inst, "REQUIRED") && contains(inst, "reflection") {
			t.Error("should not mention required reflections when none loaded")
		}
	}
}

func TestGenerateOnlyShowsActiveSpecs(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("active spec")
	s.AddSpec("completed spec")
	_ = s.CompleteSpec(2)

	g := Generate(s)

	if len(g.Specs) != 1 {
		t.Fatalf("guidance specs length = %d, want 1", len(g.Specs))
	}
	if g.Specs[0].ID != 1 {
		t.Errorf("guidance specs[0].ID = %d, want 1", g.Specs[0].ID)
	}
}
