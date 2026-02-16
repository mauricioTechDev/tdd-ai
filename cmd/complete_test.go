package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/macosta/tdd-ai/internal/reflection"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
)

func executeCompleteCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), err
}

func TestCompleteBlockedByUnansweredReflections(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.AddSpec("feature")
	s.Reflections = reflection.DefaultQuestions()
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := executeCompleteCmd(t, "complete", "--test-result", "pass", "--format", "text")
	if err == nil {
		t.Fatal("complete should be blocked when reflections are unanswered")
	}
	if !strings.Contains(err.Error(), "reflection question(s) unanswered") {
		t.Errorf("error should mention unanswered reflections, got: %v", err)
	}
}

func TestCompleteWorksWhenAllReflectionsAnswered(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.AddSpec("feature")
	s.Reflections = reflection.DefaultQuestions()
	for i := range s.Reflections {
		s.Reflections[i].Answer = "This reflection is answered with enough words"
	}
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := executeCompleteCmd(t, "complete", "--test-result", "pass", "--format", "text")
	if err != nil {
		t.Fatalf("complete should work when all reflections answered: %v", err)
	}
}

func TestCompleteWorksWithEmptyReflections(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.AddSpec("feature")
	// No reflections (backward compat)
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := executeCompleteCmd(t, "complete", "--test-result", "pass", "--format", "text")
	if err != nil {
		t.Fatalf("complete should work with empty reflections (backward compat): %v", err)
	}
}

func TestCompleteFromMidLoopBatchFinishesAllSpecs(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.AddSpec("first")
	s.AddSpec("second")
	s.AddSpec("third")
	_ = s.SetCurrentSpec(1)
	s.Reflections = reflection.DefaultQuestions()
	for i := range s.Reflections {
		s.Reflections[i].Answer = "This reflection is answered with enough words"
	}
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	out, err := executeCompleteCmd(t, "complete", "--test-result", "pass", "--format", "text")
	if err != nil {
		t.Fatalf("complete from mid-loop failed: %v", err)
	}

	if !strings.Contains(out, "3 spec(s) as done") {
		t.Errorf("should batch-complete all specs, got:\n%s", out)
	}

	loaded, _ := session.Load(dir)
	if loaded.Phase != types.PhaseDone {
		t.Errorf("phase = %s, want done", loaded.Phase)
	}
	if loaded.CurrentSpecID != nil {
		t.Error("CurrentSpecID should be nil after complete")
	}
	for _, spec := range loaded.Specs {
		if spec.Status != types.SpecStatusCompleted {
			t.Errorf("spec %d should be completed, got %s", spec.ID, spec.Status)
		}
	}
}

func TestCompleteClearsCurrentSpecID(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseGreen
	s.AddSpec("feature")
	_ = s.SetCurrentSpec(1)
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := executeCompleteCmd(t, "complete", "--test-result", "pass", "--format", "text")
	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	loaded, _ := session.Load(dir)
	if loaded.CurrentSpecID != nil {
		t.Error("CurrentSpecID should be cleared after complete")
	}
}

func TestCompleteFromRedPhaseNotBlockedByReflections(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRed
	s.AddSpec("feature")
	// No reflections loaded yet (they load at refactor entry)
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// complete from red should advance through all phases without reflection block
	_, err := executeCompleteCmd(t, "complete", "--test-result", "pass", "--format", "text")
	if err != nil {
		t.Fatalf("complete from red should not be blocked by reflections: %v", err)
	}
}
