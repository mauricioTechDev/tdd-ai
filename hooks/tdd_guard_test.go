package hooks

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
)

func hookPath() string {
	// Find the hook script relative to the test file
	dir, _ := os.Getwd()
	return filepath.Join(dir, "tdd-guard.sh")
}

type hookInput struct {
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

func runHook(t *testing.T, dir string, input hookInput) (int, string) {
	t.Helper()
	data, _ := json.Marshal(input)

	cmd := exec.Command("bash", hookPath())
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(string(data))
	out, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("unexpected error running hook: %v", err)
		}
	}
	return exitCode, string(out)
}

func TestHookBlocksNonTestFileDuringRed(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRed
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	input := hookInput{
		ToolName:  "Write",
		ToolInput: map[string]interface{}{"file_path": filepath.Join(dir, "main.go")},
	}

	code, output := runHook(t, dir, input)
	if code != 2 {
		t.Errorf("should block with exit 2, got exit %d, output: %s", code, output)
	}
}

func TestHookAllowsTestFileDuringRed(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRed
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	testFiles := []string{
		"main_test.go",
		"app.test.js",
		"app.spec.ts",
		"test/helper.go",
		"tests/integration.py",
	}

	for _, f := range testFiles {
		input := hookInput{
			ToolName:  "Write",
			ToolInput: map[string]interface{}{"file_path": filepath.Join(dir, f)},
		}
		code, output := runHook(t, dir, input)
		if code != 0 {
			t.Errorf("should allow test file %q during RED, got exit %d, output: %s", f, code, output)
		}
	}
}

func TestHookAllowsAllWritesOutsideRed(t *testing.T) {
	phases := []types.Phase{types.PhaseGreen, types.PhaseRefactor, types.PhaseDone}

	for _, phase := range phases {
		t.Run(string(phase), func(t *testing.T) {
			dir := t.TempDir()
			s := types.NewSession()
			s.Phase = phase
			if err := session.Save(dir, s); err != nil {
				t.Fatalf("failed to save session: %v", err)
			}

			input := hookInput{
				ToolName:  "Write",
				ToolInput: map[string]interface{}{"file_path": filepath.Join(dir, "main.go")},
			}
			code, output := runHook(t, dir, input)
			if code != 0 {
				t.Errorf("should allow writes during %s phase, got exit %d, output: %s", phase, code, output)
			}
		})
	}
}

func TestHookExitsZeroWhenNoSession(t *testing.T) {
	dir := t.TempDir()
	// No .tdd-ai.json file

	input := hookInput{
		ToolName:  "Write",
		ToolInput: map[string]interface{}{"file_path": filepath.Join(dir, "main.go")},
	}
	code, output := runHook(t, dir, input)
	if code != 0 {
		t.Errorf("should exit 0 when no session file, got exit %d, output: %s", code, output)
	}
}
