package verify

import (
	"fmt"

	"github.com/macosta/tdd-ai/internal/types"
)

// Violation represents a single TDD compliance violation.
type Violation struct {
	SpecID  int    `json:"spec_id,omitempty"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// Result holds the outcome of a TDD compliance analysis.
type Result struct {
	Violations     []Violation `json:"violations"`
	SpecsVerified  int         `json:"specs_verified"`
	SpecsCompliant int         `json:"specs_compliant"`
	Score          float64     `json:"score"`
	Compliant      bool        `json:"compliant"`
}

// Analyze checks a session's history for TDD compliance violations.
func Analyze(s *types.Session) Result {
	var violations []Violation

	// Check for phase_set usage (global violation)
	for _, ev := range s.History {
		if ev.Action == "phase_set" {
			violations = append(violations, Violation{
				Rule:    "no_phase_set",
				Message: fmt.Sprintf("phase_set used (%s -> %s) — bypasses TDD guardrails", ev.From, ev.To),
			})
		}
	}

	// Per-spec analysis for completed specs
	completedSpecs := completedSpecIDs(s)
	specsCompliant := 0

	for _, specID := range completedSpecs {
		specViolations := analyzeSpec(s, specID)
		if len(specViolations) == 0 {
			specsCompliant++
		}
		violations = append(violations, specViolations...)
	}

	score := float64(100)
	if len(completedSpecs) > 0 {
		score = float64(specsCompliant) / float64(len(completedSpecs)) * 100
	}

	if violations == nil {
		violations = []Violation{}
	}

	return Result{
		Violations:     violations,
		SpecsVerified:  len(completedSpecs),
		SpecsCompliant: specsCompliant,
		Score:          score,
		Compliant:      len(violations) == 0,
	}
}

// completedSpecIDs returns the IDs of all completed specs.
func completedSpecIDs(s *types.Session) []int {
	var ids []int
	for _, spec := range s.Specs {
		if spec.Status == types.SpecStatusCompleted {
			ids = append(ids, spec.ID)
		}
	}
	return ids
}

// analyzeSpec checks the history for a single completed spec's TDD compliance.
func analyzeSpec(s *types.Session, specID int) []Violation {
	var violations []Violation

	// Check for spec_picked event
	hasPicked := false
	for _, ev := range s.History {
		if ev.Action == "spec_picked" && ev.SpecID == specID {
			hasPicked = true
			break
		}
	}
	if !hasPicked {
		violations = append(violations, Violation{
			SpecID:  specID,
			Rule:    "spec_picked_required",
			Message: fmt.Sprintf("spec %d was completed without a spec_picked event", specID),
		})
	}

	// Check for RED phase: test_run with fail result after spec_picked (greenfield only)
	mode := s.GetMode()
	if mode == types.ModeGreenfield {
		hasRedFail := false
		afterPick := false
		for _, ev := range s.History {
			if ev.Action == "spec_picked" && ev.SpecID == specID {
				afterPick = true
				continue
			}
			if afterPick && ev.Action == "test_run" && ev.Result == "fail" {
				hasRedFail = true
				break
			}
			// Stop scanning if we hit the next spec_picked or phase transition past green
			if afterPick && ev.Action == "spec_picked" && ev.SpecID != specID {
				break
			}
		}
		if !hasRedFail && hasPicked {
			violations = append(violations, Violation{
				SpecID:  specID,
				Rule:    "red_test_fail_required",
				Message: fmt.Sprintf("spec %d: no failing test recorded during RED phase", specID),
			})
		}
	}

	return violations
}
