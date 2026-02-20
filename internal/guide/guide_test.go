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
	if g.ExpectedTestResult != "fail" {
		t.Errorf("expected_test_result = %q, want %q", g.ExpectedTestResult, "fail")
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
	if g.ExpectedTestResult != "pass" {
		t.Errorf("expected_test_result = %q, want %q", g.ExpectedTestResult, "pass")
	}
}

func TestGenerateRefactorPhase(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor

	g := Generate(s)

	if g.Phase != types.PhaseRefactor {
		t.Errorf("guidance phase = %q, want %q", g.Phase, types.PhaseRefactor)
	}
	if g.ExpectedTestResult != "pass" {
		t.Errorf("expected_test_result = %q, want %q", g.ExpectedTestResult, "pass")
	}
}

func TestGenerateDonePhase(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseDone

	g := Generate(s)

	if g.Phase != types.PhaseDone {
		t.Errorf("guidance phase = %q, want %q", g.Phase, types.PhaseDone)
	}
	if g.ExpectedTestResult != "" {
		t.Errorf("expected_test_result should be empty for done phase, got %q", g.ExpectedTestResult)
	}
}

func TestGenerateRetrofitRedPhase(t *testing.T) {
	s := types.NewSession()
	s.Mode = types.ModeRetrofit
	s.AddSpec("GET /users returns 200")
	_ = s.SetCurrentSpec(1)

	g := Generate(s)

	if g.Phase != types.PhaseRed {
		t.Errorf("guidance phase = %q, want %q", g.Phase, types.PhaseRed)
	}
	if g.Mode != types.ModeRetrofit {
		t.Errorf("guidance mode = %q, want %q", g.Mode, types.ModeRetrofit)
	}
	if g.ExpectedTestResult != "pass" {
		t.Errorf("retrofit red expected_test_result = %q, want %q", g.ExpectedTestResult, "pass")
	}
}

func TestGenerateIncludesNextPhase(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature")

	g := Generate(s)

	if g.NextPhase != types.PhaseGreen {
		t.Errorf("next_phase = %q, want %q", g.NextPhase, types.PhaseGreen)
	}
}

func TestGenerateRetrofitNextPhaseSkipsGreen(t *testing.T) {
	s := types.NewSession()
	s.Mode = types.ModeRetrofit
	s.AddSpec("existing feature")

	g := Generate(s)

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
}

func TestGenerateRefactorPhaseAllReflectionsAnswered(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.LastTestResult = "pass"
	s.Reflections = []types.ReflectionQuestion{
		{ID: 1, Question: "Q1", Answer: "answered with enough words here"},
		{ID: 2, Question: "Q2", Answer: "also answered with enough words here"},
	}

	g := Generate(s)

	if len(g.Blockers) != 0 {
		t.Errorf("expected no blockers when all reflections answered, got %v", g.Blockers)
	}
}

func TestGenerateRefactorPhaseNoReflections(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor

	g := Generate(s)

	if len(g.Reflections) != 0 {
		t.Errorf("guidance reflections should be empty when not loaded, got %d", len(g.Reflections))
	}
}

func TestGenerateBlockersRedNoSpecSelected(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("first")
	s.AddSpec("second")

	g := Generate(s)

	found := false
	for _, b := range g.Blockers {
		if b == "No spec selected" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("RED guidance blockers should include 'No spec selected', got %v", g.Blockers)
	}
}

func TestGeneratePopulatesCurrentSpec(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("my feature")
	_ = s.SetCurrentSpec(1)

	g := Generate(s)

	if g.CurrentSpec == nil {
		t.Fatal("CurrentSpec should be populated when a spec is selected")
	}
	if g.CurrentSpec.ID != 1 {
		t.Errorf("CurrentSpec.ID = %d, want 1", g.CurrentSpec.ID)
	}
}

func TestGeneratePopulatesIterationAndTotalSpecs(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("a")
	s.AddSpec("b")
	s.Iteration = 3

	g := Generate(s)

	if g.Iteration != 3 {
		t.Errorf("Iteration = %d, want 3", g.Iteration)
	}
	if g.TotalSpecs != 2 {
		t.Errorf("TotalSpecs = %d, want 2", g.TotalSpecs)
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

func TestGenerateDonePhaseHasBlocker(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseDone

	g := Generate(s)

	found := false
	for _, b := range g.Blockers {
		if b == "Cannot advance past done" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DONE guidance should have 'Cannot advance past done' blocker, got %v", g.Blockers)
	}
}
