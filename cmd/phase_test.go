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

func executePhaseCmd(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), errBuf.String(), err
}

func TestPhaseNextLoadsReflectionsOnRefactorEntry(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseGreen
	s.AddSpec("feature")
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, _, err := executePhaseCmd(t, "phase", "next", "--test-result", "pass", "--format", "text")
	if err != nil {
		t.Fatalf("phase next failed: %v", err)
	}

	// Verify reflections were loaded
	loaded, err := session.Load(dir)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if len(loaded.Reflections) != 7 {
		t.Errorf("should load 7 reflections on refactor entry, got %d", len(loaded.Reflections))
	}
	if loaded.Phase != types.PhaseRefactor {
		t.Errorf("phase should be refactor, got %s", loaded.Phase)
	}
}

func TestPhaseNextBlockedByUnansweredReflections(t *testing.T) {
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

	_, _, err := executePhaseCmd(t, "phase", "next", "--test-result", "pass", "--format", "text")
	if err == nil {
		t.Fatal("phase next should be blocked when reflections are unanswered")
	}
	if !strings.Contains(err.Error(), "reflection question(s) unanswered") {
		t.Errorf("error should mention unanswered reflections, got: %v", err)
	}
}

func TestPhaseNextFromRefactorWorksWhenAllAnswered(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.AddSpec("feature")
	_ = s.SetCurrentSpec(1) // Set current spec so it gets completed on refactor exit
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

	out, _, err := executePhaseCmd(t, "phase", "next", "--test-result", "pass", "--format", "text")
	if err != nil {
		t.Fatalf("phase next should work when all reflections answered: %v", err)
	}
	if !strings.Contains(out, "done") {
		t.Errorf("should advance to done, got:\n%s", out)
	}
}

func TestPhaseNextFromRefactorWorksWithEmptyReflections(t *testing.T) {
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

	_, _, err := executePhaseCmd(t, "phase", "next", "--test-result", "pass", "--format", "text")
	if err != nil {
		t.Fatalf("phase next should work with empty reflections (backward compat): %v", err)
	}
}

func TestPhaseNextFromRefactorLoopsBackToRedWhenSpecsRemain(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.AddSpec("first")
	s.AddSpec("second")
	_ = s.SetCurrentSpec(1) // Working on first spec
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

	out, _, err := executePhaseCmd(t, "phase", "next", "--test-result", "pass", "--format", "text")
	if err != nil {
		t.Fatalf("phase next from refactor failed: %v", err)
	}

	if !strings.Contains(out, "refactor -> red") {
		t.Errorf("should loop back to red, got:\n%s", out)
	}
	if !strings.Contains(out, "1 spec(s) remaining") {
		t.Errorf("should show remaining specs, got:\n%s", out)
	}

	loaded, _ := session.Load(dir)
	if loaded.Phase != types.PhaseRed {
		t.Errorf("phase = %s, want red", loaded.Phase)
	}
	if loaded.CurrentSpecID != nil {
		t.Error("CurrentSpecID should be nil after looping back to red")
	}
	if loaded.Iteration != 1 {
		t.Errorf("Iteration = %d, want 1", loaded.Iteration)
	}
	// Spec 1 should be completed
	for _, spec := range loaded.Specs {
		if spec.ID == 1 && spec.Status != types.SpecStatusCompleted {
			t.Error("spec 1 should be completed")
		}
	}
}

func TestPhaseNextFromRefactorGoesToDoneWhenNoSpecsRemain(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.AddSpec("only spec")
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

	out, _, err := executePhaseCmd(t, "phase", "next", "--test-result", "pass", "--format", "text")
	if err != nil {
		t.Fatalf("phase next from refactor failed: %v", err)
	}

	if !strings.Contains(out, "refactor -> done") {
		t.Errorf("should advance to done, got:\n%s", out)
	}
}

func TestPhaseNextAutoCompletesCurrentSpec(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.AddSpec("to complete")
	s.AddSpec("still active")
	_ = s.SetCurrentSpec(1)
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	out, _, err := executePhaseCmd(t, "phase", "next", "--test-result", "pass", "--format", "text")
	if err != nil {
		t.Fatalf("phase next from refactor failed: %v", err)
	}

	if !strings.Contains(out, "Completed spec [1]") {
		t.Errorf("should show completed spec, got:\n%s", out)
	}

	loaded, _ := session.Load(dir)
	for _, spec := range loaded.Specs {
		if spec.ID == 1 && spec.Status != types.SpecStatusCompleted {
			t.Error("spec 1 should be auto-completed")
		}
		if spec.ID == 2 && spec.Status != types.SpecStatusActive {
			t.Error("spec 2 should still be active")
		}
	}
}

func TestPhaseNextFromRedRequiresCurrentSpec(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRed
	s.AddSpec("feature")
	// No current spec set
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, _, err := executePhaseCmd(t, "phase", "next", "--test-result", "fail", "--format", "text")
	if err == nil {
		t.Fatal("phase next from red should require CurrentSpecID")
	}
	if !strings.Contains(err.Error(), "no spec selected") {
		t.Errorf("error should mention no spec selected, got: %v", err)
	}
}

func TestPhaseSetToRefactorLoadsReflections(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRed
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, _, err := executePhaseCmd(t, "phase", "set", "refactor", "--format", "text")
	if err != nil {
		t.Fatalf("phase set refactor failed: %v", err)
	}

	loaded, err := session.Load(dir)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if len(loaded.Reflections) != 7 {
		t.Errorf("should load 7 reflections when setting to refactor, got %d", len(loaded.Reflections))
	}
}

func TestPhaseSetToRefactorDoesNotOverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.Reflections = reflection.DefaultQuestions()
	s.Reflections[0].Answer = "Already answered this with enough words"
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, _, err := executePhaseCmd(t, "phase", "set", "refactor", "--format", "text")
	if err != nil {
		t.Fatalf("phase set refactor failed: %v", err)
	}

	loaded, err := session.Load(dir)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if loaded.Reflections[0].Answer != "Already answered this with enough words" {
		t.Error("should not overwrite existing reflections")
	}
}
