package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/macosta/tdd-ai/internal/reflection"
	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
)

// setupRefactorSession creates a temp directory with a session in refactor phase
// and reflections loaded. It overrides getWorkDir for the test.
func setupRefactorSession(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.Reflections = reflection.DefaultQuestions()
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	cleanup := func() {
		os.Chdir(origDir)
	}
	return dir, cleanup
}

func executeRefactorCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), err
}

func TestRefactorShowsStatus(t *testing.T) {
	_, cleanup := setupRefactorSession(t)
	defer cleanup()

	out, err := executeRefactorCmd(t, "refactor", "--format", "text")
	if err != nil {
		t.Fatalf("refactor command failed: %v", err)
	}

	if !strings.Contains(out, "0/6 answered") {
		t.Errorf("should show 0/6 answered, got:\n%s", out)
	}
}

func TestRefactorRejectsWhenNotInRefactorPhase(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRed
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := executeRefactorCmd(t, "refactor", "--format", "text")
	if err == nil {
		t.Error("refactor should error when not in refactor phase")
	}
}

func TestRefactorReflectSetsAnswer(t *testing.T) {
	dir, cleanup := setupRefactorSession(t)
	defer cleanup()

	out, err := executeRefactorCmd(t, "refactor", "reflect", "1", "--answer", "Tests are already descriptive and clear enough", "--format", "text")
	if err != nil {
		t.Fatalf("refactor reflect failed: %v", err)
	}

	if !strings.Contains(out, "Answered question 1") {
		t.Errorf("should confirm answer, got:\n%s", out)
	}
	if !strings.Contains(out, "5 remaining") {
		t.Errorf("should show remaining count, got:\n%s", out)
	}

	// Verify session was saved
	s, err := session.Load(dir)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if s.Reflections[0].Answer != "Tests are already descriptive and clear enough" {
		t.Errorf("answer not saved, got %q", s.Reflections[0].Answer)
	}
}

func TestRefactorReflectRecordsEvent(t *testing.T) {
	dir, cleanup := setupRefactorSession(t)
	defer cleanup()

	_, err := executeRefactorCmd(t, "refactor", "reflect", "1", "--answer", "Tests are already descriptive and clear enough", "--format", "text")
	if err != nil {
		t.Fatalf("refactor reflect failed: %v", err)
	}

	s, _ := session.Load(dir)
	found := false
	for _, ev := range s.History {
		if ev.Action == "reflection_answer" && ev.Result == "q1" {
			found = true
		}
	}
	if !found {
		t.Error("should record reflection_answer event")
	}
}

func TestRefactorReflectRejectsTooFewWords(t *testing.T) {
	_, cleanup := setupRefactorSession(t)
	defer cleanup()

	_, err := executeRefactorCmd(t, "refactor", "reflect", "1", "--answer", "no", "--format", "text")
	if err == nil {
		t.Error("should reject answer with too few words")
	}
	if err != nil && !strings.Contains(err.Error(), "at least 5 words") {
		t.Errorf("error should mention word count, got: %v", err)
	}
}

func TestRefactorReflectErrorsWhenNotInRefactor(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseGreen
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := executeRefactorCmd(t, "refactor", "reflect", "1", "--answer", "this has more than five words here", "--format", "text")
	if err == nil {
		t.Error("should error when not in refactor phase")
	}
}

func TestRefactorStatusText(t *testing.T) {
	dir, cleanup := setupRefactorSession(t)
	defer cleanup()

	// Answer one question first
	s, _ := session.Load(dir)
	s.Reflections[0].Answer = "Tests are already descriptive and clear enough"
	session.Save(dir, s)

	out, err := executeRefactorCmd(t, "refactor", "status", "--format", "text")
	if err != nil {
		t.Fatalf("refactor status failed: %v", err)
	}

	if !strings.Contains(out, "1/6 answered") {
		t.Errorf("should show 1/6 answered, got:\n%s", out)
	}
	if !strings.Contains(out, "(answered)") {
		t.Errorf("should show answered label, got:\n%s", out)
	}
	if !strings.Contains(out, "(pending)") {
		t.Errorf("should show pending label, got:\n%s", out)
	}
}

func TestRefactorStatusJSON(t *testing.T) {
	_, cleanup := setupRefactorSession(t)
	defer cleanup()

	out, err := executeRefactorCmd(t, "refactor", "status", "--format", "json")
	if err != nil {
		t.Fatalf("refactor status failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw:\n%s", err, out)
	}

	if parsed["total"] != float64(6) {
		t.Errorf("total = %v, want 6", parsed["total"])
	}
	if parsed["answered"] != float64(0) {
		t.Errorf("answered = %v, want 0", parsed["answered"])
	}
	if parsed["pending"] != float64(6) {
		t.Errorf("pending = %v, want 6", parsed["pending"])
	}
	if parsed["all_answered"] != false {
		t.Errorf("all_answered = %v, want false", parsed["all_answered"])
	}
}
