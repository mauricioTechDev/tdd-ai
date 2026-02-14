package types

import (
	"fmt"
	"time"
)

// Phase represents a stage in the TDD cycle.
type Phase string

const (
	PhaseRed      Phase = "red"
	PhaseGreen    Phase = "green"
	PhaseRefactor Phase = "refactor"
	PhaseDone     Phase = "done"
)

// ValidPhases returns all valid phase values.
func ValidPhases() []Phase {
	return []Phase{PhaseRed, PhaseGreen, PhaseRefactor, PhaseDone}
}

// IsValid checks whether the phase is a recognized value.
func (p Phase) IsValid() bool {
	for _, v := range ValidPhases() {
		if p == v {
			return true
		}
	}
	return false
}

// String returns the string representation of a Phase.
func (p Phase) String() string {
	return string(p)
}

// SpecStatus represents the state of a spec.
type SpecStatus string

const (
	SpecStatusActive    SpecStatus = "active"
	SpecStatusCompleted SpecStatus = "completed"
)

// Mode represents the TDD workflow mode.
type Mode string

const (
	ModeGreenfield Mode = "greenfield"
	ModeRetrofit   Mode = "retrofit"
)

// Spec is a single requirement to be implemented via TDD.
type Spec struct {
	ID          int        `json:"id"`
	Description string     `json:"description"`
	Status      SpecStatus `json:"status"`
}

// ReflectionQuestion is a structured prompt the agent must answer during the refactor phase.
type ReflectionQuestion struct {
	ID       int    `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer,omitempty"`
}

// Session holds the full state of a TDD session.
type Session struct {
	Phase          Phase                `json:"phase"`
	Mode           Mode                 `json:"mode,omitempty"`
	TestCmd        string               `json:"test_cmd,omitempty"`
	LastTestResult string               `json:"last_test_result,omitempty"`
	Specs          []Spec               `json:"specs"`
	NextID         int                  `json:"next_id"`
	Reflections    []ReflectionQuestion `json:"reflections,omitempty"`
	History        []Event              `json:"history,omitempty"`
}

// GetMode returns the session mode, defaulting to greenfield if unset.
func (s *Session) GetMode() Mode {
	if s.Mode == "" {
		return ModeGreenfield
	}
	return s.Mode
}

// NewSession creates a fresh TDD session starting in the red phase.
func NewSession() *Session {
	return &Session{
		Phase:  PhaseRed,
		Specs:  []Spec{},
		NextID: 1,
	}
}

// AddSpec adds a new spec to the session and returns the assigned ID.
func (s *Session) AddSpec(description string) int {
	id := s.NextID
	s.Specs = append(s.Specs, Spec{
		ID:          id,
		Description: description,
		Status:      SpecStatusActive,
	})
	s.NextID++
	return id
}

// CompleteSpec marks a spec as completed by ID. Returns an error if not found.
func (s *Session) CompleteSpec(id int) error {
	for i, spec := range s.Specs {
		if spec.ID == id {
			if spec.Status == SpecStatusCompleted {
				return fmt.Errorf("spec %d is already completed", id)
			}
			s.Specs[i].Status = SpecStatusCompleted
			return nil
		}
	}
	return fmt.Errorf("spec %d not found", id)
}

// CompleteAllSpecs marks all active specs as completed and returns the count.
func (s *Session) CompleteAllSpecs() int {
	count := 0
	for i, spec := range s.Specs {
		if spec.Status == SpecStatusActive {
			s.Specs[i].Status = SpecStatusCompleted
			count++
		}
	}
	return count
}

// ActiveSpecs returns only specs that are not yet completed.
func (s *Session) ActiveSpecs() []Spec {
	var active []Spec
	for _, spec := range s.Specs {
		if spec.Status == SpecStatusActive {
			active = append(active, spec)
		}
	}
	return active
}

// PendingReflections returns reflection questions that have not been answered.
func (s *Session) PendingReflections() []ReflectionQuestion {
	var pending []ReflectionQuestion
	for _, r := range s.Reflections {
		if r.Answer == "" {
			pending = append(pending, r)
		}
	}
	return pending
}

// AllReflectionsAnswered returns true when all reflection questions have answers,
// or when the reflections slice is empty (backward compatibility).
func (s *Session) AllReflectionsAnswered() bool {
	for _, r := range s.Reflections {
		if r.Answer == "" {
			return false
		}
	}
	return true
}

// AnswerReflection sets the answer for a reflection question by ID.
// Returns an error if the ID is not found.
func (s *Session) AnswerReflection(id int, answer string) error {
	for i, r := range s.Reflections {
		if r.ID == id {
			s.Reflections[i].Answer = answer
			return nil
		}
	}
	return fmt.Errorf("reflection question %d not found", id)
}

// Event records a notable action during the TDD session for audit trail.
type Event struct {
	Action    string `json:"action"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Result    string `json:"result,omitempty"`
	SpecCount int    `json:"spec_count,omitempty"`
	Timestamp string `json:"at"`
}

// AddEvent appends an event to the session history.
func (s *Session) AddEvent(action string, opts ...func(*Event)) {
	e := Event{
		Action:    action,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	for _, opt := range opts {
		opt(&e)
	}
	s.History = append(s.History, e)
}

// Guidance is the structured output of the guide command.
type Guidance struct {
	Phase        Phase                `json:"phase"`
	Mode         Mode                 `json:"mode"`
	NextPhase    Phase                `json:"next_phase,omitempty"`
	TestCmd      string               `json:"test_cmd,omitempty"`
	Specs        []Spec               `json:"specs"`
	Instructions []string             `json:"instructions"`
	Rules        []string             `json:"rules"`
	Reflections  []ReflectionQuestion `json:"reflections,omitempty"`
}
