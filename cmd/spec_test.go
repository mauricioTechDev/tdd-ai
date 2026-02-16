package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
)

func executeSpecCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), err
}

func TestSpecPickSetsCurrentSpecID(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.AddSpec("first spec")
	s.AddSpec("second spec")
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	out, err := executeSpecCmd(t, "spec", "pick", "2", "--format", "text")
	if err != nil {
		t.Fatalf("spec pick failed: %v", err)
	}

	if !strings.Contains(out, "Picked spec [2]") {
		t.Errorf("should confirm picked spec, got:\n%s", out)
	}
	if !strings.Contains(out, "1 spec(s) remaining") {
		t.Errorf("should show remaining count, got:\n%s", out)
	}

	// Verify session was saved
	loaded, err := session.Load(dir)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if loaded.CurrentSpecID == nil || *loaded.CurrentSpecID != 2 {
		t.Errorf("CurrentSpecID = %v, want 2", loaded.CurrentSpecID)
	}

	// Verify event was recorded
	found := false
	for _, ev := range loaded.History {
		if ev.Action == "spec_picked" && ev.SpecID == 2 {
			found = true
		}
	}
	if !found {
		t.Error("should record spec_picked event with spec ID")
	}
}

func TestSpecPickRejectsInvalidID(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.AddSpec("first spec")
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := executeSpecCmd(t, "spec", "pick", "99", "--format", "text")
	if err == nil {
		t.Error("spec pick should reject nonexistent spec ID")
	}
}

func TestSpecPickRejectsCompletedSpec(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.AddSpec("done spec")
	_ = s.CompleteSpec(1)
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := executeSpecCmd(t, "spec", "pick", "1", "--format", "text")
	if err == nil {
		t.Error("spec pick should reject completed spec")
	}
}

func TestSpecPickOnlyWorksInRedPhase(t *testing.T) {
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

	_, err := executeSpecCmd(t, "spec", "pick", "1", "--format", "text")
	if err == nil {
		t.Error("spec pick should only work in RED phase")
	}
	if err != nil && !strings.Contains(err.Error(), "RED phase") {
		t.Errorf("error should mention RED phase, got: %v", err)
	}
}

func TestSpecListMarksCurrentSpec(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.AddSpec("first spec")
	s.AddSpec("second spec")
	_ = s.SetCurrentSpec(1)
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	out, err := executeSpecCmd(t, "spec", "list", "--format", "text")
	if err != nil {
		t.Fatalf("spec list failed: %v", err)
	}

	if !strings.Contains(out, "(current)") {
		t.Errorf("spec list should mark current spec, got:\n%s", out)
	}
}
