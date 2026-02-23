package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
)

func executeVerifyCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), err
}

func TestVerifyCompliantSessionExitsZero(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseDone
	s.AddSpec("feature A")
	_ = s.CompleteSpec(1)

	s.AddEvent("init", func(e *types.Event) { e.Result = "greenfield" })
	s.AddEvent("spec_picked", func(e *types.Event) { e.SpecID = 1 })
	s.AddEvent("test_run", func(e *types.Event) { e.Result = "fail" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "red"; e.To = "green"; e.Result = "fail" })
	s.AddEvent("test_run", func(e *types.Event) { e.Result = "pass" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "green"; e.To = "refactor"; e.Result = "pass" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "refactor"; e.To = "done"; e.Result = "pass" })

	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	out, err := executeVerifyCmd(t, "verify", "--format", "text")
	if err != nil {
		t.Fatalf("verify should succeed for compliant session: %v", err)
	}
	if !strings.Contains(out, "100") {
		t.Errorf("should show 100%% compliance, got:\n%s", out)
	}
}

func TestVerifyNonCompliantSessionExitsOne(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseDone
	s.AddSpec("feature A")
	_ = s.CompleteSpec(1)

	// Missing spec_picked — violation
	s.AddEvent("init", func(e *types.Event) { e.Result = "greenfield" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "red"; e.To = "green" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "green"; e.To = "refactor" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "refactor"; e.To = "done" })

	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := executeVerifyCmd(t, "verify", "--format", "text")
	if err == nil {
		t.Fatal("verify should fail for non-compliant session")
	}
}

func TestVerifyJSONOutput(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseDone
	s.AddSpec("feature A")
	_ = s.CompleteSpec(1)

	s.AddEvent("init", func(e *types.Event) { e.Result = "greenfield" })
	s.AddEvent("spec_picked", func(e *types.Event) { e.SpecID = 1 })
	s.AddEvent("test_run", func(e *types.Event) { e.Result = "fail" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "red"; e.To = "green"; e.Result = "fail" })
	s.AddEvent("test_run", func(e *types.Event) { e.Result = "pass" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "green"; e.To = "refactor"; e.Result = "pass" })
	s.AddEvent("phase_next", func(e *types.Event) { e.From = "refactor"; e.To = "done"; e.Result = "pass" })

	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	out, err := executeVerifyCmd(t, "verify", "--format", "json")
	if err != nil {
		t.Fatalf("verify should succeed for compliant session: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["compliant"] != true {
		t.Errorf("compliant = %v, want true", parsed["compliant"])
	}
	if parsed["score"] != float64(100) {
		t.Errorf("score = %v, want 100", parsed["score"])
	}
}
