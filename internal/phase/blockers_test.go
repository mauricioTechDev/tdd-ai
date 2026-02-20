package phase

import (
	"strings"
	"testing"

	"github.com/macosta/tdd-ai/internal/types"
)

func TestGetBlockersRedNoSpecs(t *testing.T) {
	s := types.NewSession()

	blockers := GetBlockers(s)

	assertContains(t, blockers, "No active specs")
	assertContains(t, blockers, "No test result recorded")
}

func TestGetBlockersRedNoSpecSelected(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature")

	blockers := GetBlockers(s)

	assertContains(t, blockers, "No spec selected")
	assertContains(t, blockers, "No test result recorded")
	assertNotContains(t, blockers, "No active specs")
}

func TestGetBlockersRedNoTestResult(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature")
	_ = s.SetCurrentSpec(1)

	blockers := GetBlockers(s)

	assertContains(t, blockers, "No test result recorded")
	assertNotContains(t, blockers, "No spec selected")
}

func TestGetBlockersRedWrongTestResult(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature")
	_ = s.SetCurrentSpec(1)
	s.LastTestResult = "pass"

	blockers := GetBlockers(s)

	assertContains(t, blockers, "does not match expected 'fail'")
}

func TestGetBlockersRedCorrectTestResult(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature")
	_ = s.SetCurrentSpec(1)
	s.LastTestResult = "fail"

	blockers := GetBlockers(s)

	if len(blockers) != 0 {
		t.Errorf("expected no blockers, got %v", blockers)
	}
}

func TestGetBlockersRedRetrofitExpectsPass(t *testing.T) {
	s := types.NewSession()
	s.Mode = types.ModeRetrofit
	s.AddSpec("feature")
	_ = s.SetCurrentSpec(1)
	s.LastTestResult = "fail"

	blockers := GetBlockers(s)

	assertContains(t, blockers, "does not match expected 'pass'")
}

func TestGetBlockersGreenNoTestResult(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseGreen

	blockers := GetBlockers(s)

	assertContains(t, blockers, "No test result recorded")
}

func TestGetBlockersGreenWrongTestResult(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseGreen
	s.LastTestResult = "fail"

	blockers := GetBlockers(s)

	assertContains(t, blockers, "does not match expected 'pass'")
}

func TestGetBlockersGreenCorrectTestResult(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseGreen
	s.LastTestResult = "pass"

	blockers := GetBlockers(s)

	if len(blockers) != 0 {
		t.Errorf("expected no blockers, got %v", blockers)
	}
}

func TestGetBlockersRefactorNoTestResult(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor

	blockers := GetBlockers(s)

	assertContains(t, blockers, "No test result recorded")
}

func TestGetBlockersRefactorWrongTestResult(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.LastTestResult = "fail"

	blockers := GetBlockers(s)

	assertContains(t, blockers, "does not match expected 'pass'")
}

func TestGetBlockersRefactorUnansweredReflections(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.LastTestResult = "pass"
	s.Reflections = []types.ReflectionQuestion{
		{ID: 1, Question: "Q1", Answer: ""},
		{ID: 2, Question: "Q2", Answer: ""},
	}

	blockers := GetBlockers(s)

	assertContains(t, blockers, "2 reflection questions unanswered")
}

func TestGetBlockersRefactorAllAnswered(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.LastTestResult = "pass"
	s.Reflections = []types.ReflectionQuestion{
		{ID: 1, Question: "Q1", Answer: "answered"},
		{ID: 2, Question: "Q2", Answer: "answered"},
	}

	blockers := GetBlockers(s)

	if len(blockers) != 0 {
		t.Errorf("expected no blockers, got %v", blockers)
	}
}

func TestGetBlockersRefactorNoReflections(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.LastTestResult = "pass"

	blockers := GetBlockers(s)

	if len(blockers) != 0 {
		t.Errorf("expected no blockers with no reflections, got %v", blockers)
	}
}

func TestGetBlockersDone(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseDone

	blockers := GetBlockers(s)

	assertContains(t, blockers, "Cannot advance past done")
}

func assertContains(t *testing.T, blockers []string, substr string) {
	t.Helper()
	for _, b := range blockers {
		if strings.Contains(b, substr) {
			return
		}
	}
	t.Errorf("blockers %v should contain %q", blockers, substr)
}

func assertNotContains(t *testing.T, blockers []string, substr string) {
	t.Helper()
	for _, b := range blockers {
		if strings.Contains(b, substr) {
			t.Errorf("blockers %v should not contain %q", blockers, substr)
			return
		}
	}
}
