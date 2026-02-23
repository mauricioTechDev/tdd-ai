package hooks

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/macosta/tdd-ai/internal/session"
	"github.com/macosta/tdd-ai/internal/types"
)

func commitCheckHookPath() string {
	return filepath.Join(repoRoot(), ".claude", "hooks", "tdd-commit-check.sh")
}

func runCommitCheckHook(t *testing.T, dir string, bashCmd string) (int, string) {
	t.Helper()
	// Simulate Claude Code Bash hook input
	input := `{"tool_name":"Bash","tool_input":{"command":"` + bashCmd + `"}}`

	cmd := exec.Command("bash", commitCheckHookPath())
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(input)
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

func TestCommitCheckBlocksWhenPhaseNotDone(t *testing.T) {
	phases := []types.Phase{types.PhaseRed, types.PhaseGreen, types.PhaseRefactor}
	for _, phase := range phases {
		t.Run(string(phase), func(t *testing.T) {
			dir := t.TempDir()
			s := types.NewSession()
			s.Phase = phase
			if err := session.Save(dir, s); err != nil {
				t.Fatalf("failed to save session: %v", err)
			}

			code, output := runCommitCheckHook(t, dir, "git commit -m 'test'")
			if code != 2 {
				t.Errorf("should block with exit 2 during %s phase, got exit %d, output: %s", phase, code, output)
			}
		})
	}
}

func TestCommitCheckAllowsWhenPhaseDone(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseDone
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	code, output := runCommitCheckHook(t, dir, "git commit -m 'test'")
	if code != 0 {
		t.Errorf("should allow when phase is done, got exit %d, output: %s", code, output)
	}
}

func TestCommitCheckAllowsWhenNoSession(t *testing.T) {
	dir := t.TempDir()
	// No .tdd-ai.json file

	code, output := runCommitCheckHook(t, dir, "git commit -m 'test'")
	if code != 0 {
		t.Errorf("should allow when no session file, got exit %d, output: %s", code, output)
	}
}

func TestCommitCheckIgnoresNonCommitCommands(t *testing.T) {
	dir := t.TempDir()
	s := types.NewSession()
	s.Phase = types.PhaseRed
	if err := session.Save(dir, s); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	code, output := runCommitCheckHook(t, dir, "git status")
	if code != 0 {
		t.Errorf("should allow non-commit commands, got exit %d, output: %s", code, output)
	}
}
