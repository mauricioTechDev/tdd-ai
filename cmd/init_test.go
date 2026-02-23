package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/macosta/tdd-ai/internal/session"
)

func executeInitCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), err
}

func TestInitAgentMode(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	out, err := executeInitCmd(t, "init", "--agent", "--format", "text")
	if err != nil {
		t.Fatalf("init --agent failed: %v", err)
	}
	if !strings.Contains(out, "agent") {
		t.Errorf("output should mention agent mode, got:\n%s", out)
	}

	loaded, err := session.Load(dir)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if !loaded.AgentMode {
		t.Error("AgentMode should be true when --agent is used")
	}
}

func TestInitWithoutAgentModeIsDefault(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Reset flags from previous test runs
	agentFlag = false
	retrofitFlag = false
	testCmdFlag = ""

	_, err := executeInitCmd(t, "init", "--format", "text")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	loaded, err := session.Load(dir)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if loaded.AgentMode {
		t.Error("AgentMode should be false when --agent is not used")
	}
}
