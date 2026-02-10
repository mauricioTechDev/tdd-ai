package phase

import (
	"testing"

	"github.com/macosta/tdd-ai/internal/types"
)

func TestNextPhaseValidTransitions(t *testing.T) {
	tests := []struct {
		name    string
		current types.Phase
		want    types.Phase
	}{
		{"red to green", types.PhaseRed, types.PhaseGreen},
		{"green to refactor", types.PhaseGreen, types.PhaseRefactor},
		{"refactor to done", types.PhaseRefactor, types.PhaseDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Next(tt.current)
			if err != nil {
				t.Fatalf("Next(%q) returned unexpected error: %v", tt.current, err)
			}
			if got != tt.want {
				t.Errorf("Next(%q) = %q, want %q", tt.current, got, tt.want)
			}
		})
	}
}

func TestNextPhaseFromDoneErrors(t *testing.T) {
	_, err := Next(types.PhaseDone)
	if err == nil {
		t.Error("Next(done) should return an error")
	}
}

func TestNextPhaseFromInvalidErrors(t *testing.T) {
	_, err := Next(types.Phase("invalid"))
	if err == nil {
		t.Error("Next(invalid) should return an error")
	}
}

func TestNextWithModeGreenfield(t *testing.T) {
	// Greenfield mode should behave like normal: red->green->refactor->done
	tests := []struct {
		name    string
		current types.Phase
		want    types.Phase
	}{
		{"red to green", types.PhaseRed, types.PhaseGreen},
		{"green to refactor", types.PhaseGreen, types.PhaseRefactor},
		{"refactor to done", types.PhaseRefactor, types.PhaseDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextWithMode(tt.current, types.ModeGreenfield)
			if err != nil {
				t.Fatalf("NextWithMode(%q, greenfield) error: %v", tt.current, err)
			}
			if got != tt.want {
				t.Errorf("NextWithMode(%q, greenfield) = %q, want %q", tt.current, got, tt.want)
			}
		})
	}
}

func TestNextWithModeRetrofit(t *testing.T) {
	// Retrofit mode: red->refactor (skips green), refactor->done
	tests := []struct {
		name    string
		current types.Phase
		want    types.Phase
	}{
		{"red to refactor", types.PhaseRed, types.PhaseRefactor},
		{"refactor to done", types.PhaseRefactor, types.PhaseDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextWithMode(tt.current, types.ModeRetrofit)
			if err != nil {
				t.Fatalf("NextWithMode(%q, retrofit) error: %v", tt.current, err)
			}
			if got != tt.want {
				t.Errorf("NextWithMode(%q, retrofit) = %q, want %q", tt.current, got, tt.want)
			}
		})
	}
}

func TestNextWithModeRetrofitFromDoneErrors(t *testing.T) {
	_, err := NextWithMode(types.PhaseDone, types.ModeRetrofit)
	if err == nil {
		t.Error("NextWithMode(done, retrofit) should return an error")
	}
}

func TestExpectedTestResult(t *testing.T) {
	tests := []struct {
		name  string
		phase types.Phase
		mode  types.Mode
		want  string
	}{
		{"greenfield red expects fail", types.PhaseRed, types.ModeGreenfield, "fail"},
		{"greenfield green expects pass", types.PhaseGreen, types.ModeGreenfield, "pass"},
		{"greenfield refactor expects pass", types.PhaseRefactor, types.ModeGreenfield, "pass"},
		{"retrofit red expects pass", types.PhaseRed, types.ModeRetrofit, "pass"},
		{"retrofit refactor expects pass", types.PhaseRefactor, types.ModeRetrofit, "pass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpectedTestResult(tt.phase, tt.mode)
			if got != tt.want {
				t.Errorf("ExpectedTestResult(%q, %q) = %q, want %q", tt.phase, tt.mode, got, tt.want)
			}
		})
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from types.Phase
		to   types.Phase
		want bool
	}{
		{"red to green", types.PhaseRed, types.PhaseGreen, true},
		{"green to refactor", types.PhaseGreen, types.PhaseRefactor, true},
		{"refactor to done", types.PhaseRefactor, types.PhaseDone, true},
		{"red to refactor not allowed", types.PhaseRed, types.PhaseRefactor, false},
		{"green to red not allowed", types.PhaseGreen, types.PhaseRed, false},
		{"done to red not allowed", types.PhaseDone, types.PhaseRed, false},
		{"same phase not allowed", types.PhaseRed, types.PhaseRed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to); got != tt.want {
				t.Errorf("CanTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
