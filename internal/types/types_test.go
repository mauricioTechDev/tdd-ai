package types

import (
	"testing"
)

func TestPhaseIsValid(t *testing.T) {
	tests := []struct {
		name  string
		phase Phase
		want  bool
	}{
		{"red is valid", PhaseRed, true},
		{"green is valid", PhaseGreen, true},
		{"refactor is valid", PhaseRefactor, true},
		{"done is valid", PhaseDone, true},
		{"empty is invalid", Phase(""), false},
		{"unknown is invalid", Phase("blue"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.phase.IsValid(); got != tt.want {
				t.Errorf("Phase(%q).IsValid() = %v, want %v", tt.phase, got, tt.want)
			}
		})
	}
}

func TestPhaseString(t *testing.T) {
	if PhaseRed.String() != "red" {
		t.Errorf("PhaseRed.String() = %q, want %q", PhaseRed.String(), "red")
	}
}

func TestNewSession(t *testing.T) {
	s := NewSession()

	if s.Phase != PhaseRed {
		t.Errorf("NewSession().Phase = %q, want %q", s.Phase, PhaseRed)
	}
	if len(s.Specs) != 0 {
		t.Errorf("NewSession().Specs length = %d, want 0", len(s.Specs))
	}
	if s.NextID != 1 {
		t.Errorf("NewSession().NextID = %d, want 1", s.NextID)
	}
}

func TestAddSpec(t *testing.T) {
	s := NewSession()

	id1 := s.AddSpec("first feature")
	id2 := s.AddSpec("second feature")

	if id1 != 1 {
		t.Errorf("first AddSpec returned id %d, want 1", id1)
	}
	if id2 != 2 {
		t.Errorf("second AddSpec returned id %d, want 2", id2)
	}
	if len(s.Specs) != 2 {
		t.Fatalf("Specs length = %d, want 2", len(s.Specs))
	}
	if s.Specs[0].Description != "first feature" {
		t.Errorf("Specs[0].Description = %q, want %q", s.Specs[0].Description, "first feature")
	}
	if s.Specs[0].Status != SpecStatusActive {
		t.Errorf("Specs[0].Status = %q, want %q", s.Specs[0].Status, SpecStatusActive)
	}
}

func TestCompleteSpec(t *testing.T) {
	s := NewSession()
	s.AddSpec("feature to complete")

	err := s.CompleteSpec(1)
	if err != nil {
		t.Fatalf("CompleteSpec(1) returned unexpected error: %v", err)
	}
	if s.Specs[0].Status != SpecStatusCompleted {
		t.Errorf("Specs[0].Status = %q, want %q", s.Specs[0].Status, SpecStatusCompleted)
	}
}

func TestCompleteSpecNotFound(t *testing.T) {
	s := NewSession()

	err := s.CompleteSpec(99)
	if err == nil {
		t.Error("CompleteSpec(99) should return error for nonexistent spec")
	}
}

func TestCompleteSpecAlreadyDone(t *testing.T) {
	s := NewSession()
	s.AddSpec("already done")
	_ = s.CompleteSpec(1)

	err := s.CompleteSpec(1)
	if err == nil {
		t.Error("CompleteSpec(1) should return error when already completed")
	}
}

func TestCompleteAllSpecs(t *testing.T) {
	s := NewSession()
	s.AddSpec("first")
	s.AddSpec("second")
	s.AddSpec("third")
	_ = s.CompleteSpec(2) // already done

	count := s.CompleteAllSpecs()
	if count != 2 {
		t.Errorf("CompleteAllSpecs() = %d, want 2 (only active specs)", count)
	}

	active := s.ActiveSpecs()
	if len(active) != 0 {
		t.Errorf("ActiveSpecs() after CompleteAllSpecs() = %d, want 0", len(active))
	}
}

func TestCompleteAllSpecsNoneActive(t *testing.T) {
	s := NewSession()
	s.AddSpec("only")
	_ = s.CompleteSpec(1)

	count := s.CompleteAllSpecs()
	if count != 0 {
		t.Errorf("CompleteAllSpecs() = %d, want 0 (no active specs)", count)
	}
}

func TestGetModeDefaultsToGreenfield(t *testing.T) {
	s := NewSession()
	if s.GetMode() != ModeGreenfield {
		t.Errorf("GetMode() = %q, want %q", s.GetMode(), ModeGreenfield)
	}
}

func TestGetModeReturnsRetrofit(t *testing.T) {
	s := NewSession()
	s.Mode = ModeRetrofit
	if s.GetMode() != ModeRetrofit {
		t.Errorf("GetMode() = %q, want %q", s.GetMode(), ModeRetrofit)
	}
}

func TestGetModeBackwardCompatEmptyString(t *testing.T) {
	// Simulates loading an old session file that has no mode field
	s := &Session{Phase: PhaseRed, Mode: ""}
	if s.GetMode() != ModeGreenfield {
		t.Errorf("GetMode() for empty mode = %q, want %q", s.GetMode(), ModeGreenfield)
	}
}

func TestSessionTestCmdField(t *testing.T) {
	s := NewSession()
	if s.TestCmd != "" {
		t.Errorf("new session TestCmd should be empty, got %q", s.TestCmd)
	}
	s.TestCmd = "go test ./..."
	if s.TestCmd != "go test ./..." {
		t.Errorf("TestCmd = %q, want %q", s.TestCmd, "go test ./...")
	}
}

func TestSessionLastTestResultField(t *testing.T) {
	s := NewSession()
	if s.LastTestResult != "" {
		t.Errorf("new session LastTestResult should be empty, got %q", s.LastTestResult)
	}
	s.LastTestResult = "pass"
	if s.LastTestResult != "pass" {
		t.Errorf("LastTestResult = %q, want %q", s.LastTestResult, "pass")
	}
}

func TestActiveSpecs(t *testing.T) {
	s := NewSession()
	s.AddSpec("active one")
	s.AddSpec("will complete")
	s.AddSpec("active two")
	_ = s.CompleteSpec(2)

	active := s.ActiveSpecs()
	if len(active) != 2 {
		t.Fatalf("ActiveSpecs() length = %d, want 2", len(active))
	}
	if active[0].ID != 1 {
		t.Errorf("ActiveSpecs()[0].ID = %d, want 1", active[0].ID)
	}
	if active[1].ID != 3 {
		t.Errorf("ActiveSpecs()[1].ID = %d, want 3", active[1].ID)
	}
}

func TestNewSessionHistoryIsEmpty(t *testing.T) {
	s := NewSession()
	if len(s.History) != 0 {
		t.Errorf("new session History should be nil/empty, got %d events", len(s.History))
	}
}

func TestAddEvent(t *testing.T) {
	s := NewSession()
	s.AddEvent("test_run", func(e *Event) {
		e.Result = "pass"
	})

	if len(s.History) != 1 {
		t.Fatalf("History length = %d, want 1", len(s.History))
	}
	ev := s.History[0]
	if ev.Action != "test_run" {
		t.Errorf("event Action = %q, want %q", ev.Action, "test_run")
	}
	if ev.Result != "pass" {
		t.Errorf("event Result = %q, want %q", ev.Result, "pass")
	}
	if ev.Timestamp == "" {
		t.Error("event Timestamp should not be empty")
	}
}

func TestAddEventPhaseTransition(t *testing.T) {
	s := NewSession()
	s.AddEvent("phase_next", func(e *Event) {
		e.From = "red"
		e.To = "green"
		e.Result = "fail"
	})

	if len(s.History) != 1 {
		t.Fatalf("History length = %d, want 1", len(s.History))
	}
	ev := s.History[0]
	if ev.From != "red" {
		t.Errorf("event From = %q, want %q", ev.From, "red")
	}
	if ev.To != "green" {
		t.Errorf("event To = %q, want %q", ev.To, "green")
	}
}

func TestAddEventMultiple(t *testing.T) {
	s := NewSession()
	s.AddEvent("init", func(_ *Event) {})
	s.AddEvent("spec_add", func(e *Event) { e.SpecCount = 3 })
	s.AddEvent("test_run", func(e *Event) { e.Result = "pass" })

	if len(s.History) != 3 {
		t.Fatalf("History length = %d, want 3", len(s.History))
	}
	if s.History[1].SpecCount != 3 {
		t.Errorf("History[1].SpecCount = %d, want 3", s.History[1].SpecCount)
	}
}

func TestPendingReflections(t *testing.T) {
	s := NewSession()
	s.Reflections = []ReflectionQuestion{
		{ID: 1, Question: "Q1", Answer: "answered with enough words here"},
		{ID: 2, Question: "Q2", Answer: ""},
		{ID: 3, Question: "Q3", Answer: "also answered with enough words"},
	}

	pending := s.PendingReflections()
	if len(pending) != 1 {
		t.Fatalf("PendingReflections() = %d, want 1", len(pending))
	}
	if pending[0].ID != 2 {
		t.Errorf("PendingReflections()[0].ID = %d, want 2", pending[0].ID)
	}
}

func TestPendingReflectionsAllAnswered(t *testing.T) {
	s := NewSession()
	s.Reflections = []ReflectionQuestion{
		{ID: 1, Question: "Q1", Answer: "done"},
	}

	pending := s.PendingReflections()
	if len(pending) != 0 {
		t.Errorf("PendingReflections() = %d, want 0", len(pending))
	}
}

func TestPendingReflectionsEmpty(t *testing.T) {
	s := NewSession()

	pending := s.PendingReflections()
	if len(pending) != 0 {
		t.Errorf("PendingReflections() on empty = %d, want 0", len(pending))
	}
}

func TestAllReflectionsAnsweredTrue(t *testing.T) {
	s := NewSession()
	s.Reflections = []ReflectionQuestion{
		{ID: 1, Question: "Q1", Answer: "yes"},
		{ID: 2, Question: "Q2", Answer: "yes"},
	}

	if !s.AllReflectionsAnswered() {
		t.Error("AllReflectionsAnswered() = false, want true")
	}
}

func TestAllReflectionsAnsweredFalse(t *testing.T) {
	s := NewSession()
	s.Reflections = []ReflectionQuestion{
		{ID: 1, Question: "Q1", Answer: "yes"},
		{ID: 2, Question: "Q2", Answer: ""},
	}

	if s.AllReflectionsAnswered() {
		t.Error("AllReflectionsAnswered() = true, want false")
	}
}

func TestAllReflectionsAnsweredEmptySlice(t *testing.T) {
	s := NewSession()

	if !s.AllReflectionsAnswered() {
		t.Error("AllReflectionsAnswered() on empty = false, want true (backward compat)")
	}
}

func TestAnswerReflection(t *testing.T) {
	s := NewSession()
	s.Reflections = []ReflectionQuestion{
		{ID: 1, Question: "Q1"},
		{ID: 2, Question: "Q2"},
	}

	err := s.AnswerReflection(1, "my detailed answer here")
	if err != nil {
		t.Fatalf("AnswerReflection(1) unexpected error: %v", err)
	}
	if s.Reflections[0].Answer != "my detailed answer here" {
		t.Errorf("Reflections[0].Answer = %q, want %q", s.Reflections[0].Answer, "my detailed answer here")
	}
}

func TestAnswerReflectionNotFound(t *testing.T) {
	s := NewSession()
	s.Reflections = []ReflectionQuestion{
		{ID: 1, Question: "Q1"},
	}

	err := s.AnswerReflection(99, "some answer")
	if err == nil {
		t.Error("AnswerReflection(99) should return error for nonexistent ID")
	}
}
