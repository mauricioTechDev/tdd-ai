package verify

import (
	"testing"

	"github.com/macosta/tdd-ai/internal/types"
)

// buildCompliantSession creates a session with a fully compliant TDD history
// for one completed spec.
func buildCompliantSession() *types.Session {
	s := types.NewSession()
	s.Phase = types.PhaseDone
	s.AddSpec("feature A")
	_ = s.CompleteSpec(1)

	s.AddEvent("init", func(e *types.Event) { e.Result = "greenfield" })
	s.AddEvent("spec_add", func(e *types.Event) { e.SpecCount = 1 })
	s.AddEvent("spec_picked", func(e *types.Event) { e.SpecID = 1 })
	s.AddEvent("test_run", func(e *types.Event) { e.Result = "fail" })
	s.AddEvent("phase_next", func(e *types.Event) {
		e.From = "red"
		e.To = "green"
		e.Result = "fail"
	})
	s.AddEvent("test_run", func(e *types.Event) { e.Result = "pass" })
	s.AddEvent("phase_next", func(e *types.Event) {
		e.From = "green"
		e.To = "refactor"
		e.Result = "pass"
	})
	s.AddEvent("phase_next", func(e *types.Event) {
		e.From = "refactor"
		e.To = "done"
		e.Result = "pass"
	})
	return s
}

func TestAnalyzeDetectsMissingSpecPicked(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseDone
	s.AddSpec("feature A")
	_ = s.CompleteSpec(1)

	// Simulate a completed spec without spec_picked event
	s.AddEvent("init", func(e *types.Event) { e.Result = "greenfield" })
	s.AddEvent("spec_add", func(e *types.Event) { e.SpecCount = 1 })
	// Missing: spec_picked for spec 1
	s.AddEvent("phase_next", func(e *types.Event) {
		e.From = "red"
		e.To = "green"
		e.Result = "fail"
	})
	s.AddEvent("phase_next", func(e *types.Event) {
		e.From = "green"
		e.To = "refactor"
		e.Result = "pass"
	})
	s.AddEvent("phase_next", func(e *types.Event) {
		e.From = "refactor"
		e.To = "done"
		e.Result = "pass"
	})

	result := Analyze(s)

	if len(result.Violations) == 0 {
		t.Fatal("should detect missing spec_picked violation")
	}
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec_picked_required" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("should have spec_picked_required violation, got: %+v", result.Violations)
	}
}

func TestAnalyzeDetectsMissingRedFail(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseDone
	s.AddSpec("feature A")
	_ = s.CompleteSpec(1)

	s.AddEvent("init", func(e *types.Event) { e.Result = "greenfield" })
	s.AddEvent("spec_picked", func(e *types.Event) { e.SpecID = 1 })
	// Missing: test_run with result=fail after spec_picked
	s.AddEvent("phase_next", func(e *types.Event) {
		e.From = "red"
		e.To = "green"
		e.Result = "fail"
	})
	s.AddEvent("phase_next", func(e *types.Event) {
		e.From = "green"
		e.To = "refactor"
		e.Result = "pass"
	})
	s.AddEvent("phase_next", func(e *types.Event) {
		e.From = "refactor"
		e.To = "done"
		e.Result = "pass"
	})

	result := Analyze(s)

	found := false
	for _, v := range result.Violations {
		if v.Rule == "red_test_fail_required" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("should have red_test_fail_required violation, got: %+v", result.Violations)
	}
}

func TestAnalyzeDetectsPhaseSetUsage(t *testing.T) {
	s := buildCompliantSession()

	// Add a phase_set event
	s.AddEvent("phase_set", func(e *types.Event) {
		e.From = "red"
		e.To = "green"
		e.Result = "forced_override"
	})

	result := Analyze(s)

	found := false
	for _, v := range result.Violations {
		if v.Rule == "no_phase_set" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("should have no_phase_set violation, got: %+v", result.Violations)
	}
	if result.Compliant {
		t.Error("result should not be compliant when phase_set is used")
	}
}

func TestAnalyzeReturnsComplianceScore(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseDone
	s.AddSpec("spec A")
	s.AddSpec("spec B")
	_ = s.CompleteSpec(1)
	_ = s.CompleteSpec(2)

	s.AddEvent("init", func(e *types.Event) { e.Result = "greenfield" })

	// Spec 1: fully compliant
	s.AddEvent("spec_picked", func(e *types.Event) { e.SpecID = 1 })
	s.AddEvent("test_run", func(e *types.Event) { e.Result = "fail" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "red"; e.To = "green" })
	s.AddEvent("test_run", func(e *types.Event) { e.Result = "pass" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "green"; e.To = "refactor" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "refactor"; e.To = "red" })

	// Spec 2: missing spec_picked (non-compliant)
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "red"; e.To = "green" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "green"; e.To = "refactor" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "refactor"; e.To = "done" })

	result := Analyze(s)

	if result.SpecsVerified != 2 {
		t.Errorf("SpecsVerified = %d, want 2", result.SpecsVerified)
	}
	if result.SpecsCompliant != 1 {
		t.Errorf("SpecsCompliant = %d, want 1", result.SpecsCompliant)
	}
	if result.Score != 50 {
		t.Errorf("Score = %v, want 50", result.Score)
	}
	if result.Compliant {
		t.Error("result should not be compliant")
	}
}

func TestAnalyzeFullyCompliantSession(t *testing.T) {
	s := buildCompliantSession()

	result := Analyze(s)

	if !result.Compliant {
		t.Errorf("should be compliant, got violations: %+v", result.Violations)
	}
	if result.Score != 100 {
		t.Errorf("Score = %v, want 100", result.Score)
	}
	if result.SpecsVerified != 1 {
		t.Errorf("SpecsVerified = %d, want 1", result.SpecsVerified)
	}
	if result.SpecsCompliant != 1 {
		t.Errorf("SpecsCompliant = %d, want 1", result.SpecsCompliant)
	}
}

func TestAnalyzeEmptyHistoryGraceful(t *testing.T) {
	s := types.NewSession()
	// No specs, no history

	result := Analyze(s)

	if len(result.Violations) != 0 {
		t.Errorf("should have no violations for empty session, got: %+v", result.Violations)
	}
	if !result.Compliant {
		t.Error("empty session should be compliant")
	}
	if result.Score != 100 {
		t.Errorf("Score = %v, want 100 for empty session", result.Score)
	}
	if result.SpecsVerified != 0 {
		t.Errorf("SpecsVerified = %d, want 0", result.SpecsVerified)
	}
}
