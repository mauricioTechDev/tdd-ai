package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func executeCommands(t *testing.T, args ...string) string {
	t.Helper()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs(append([]string{"commands"}, args...))

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("commands command failed: %v", err)
	}
	return buf.String()
}

func TestCommandsOutputJSON(t *testing.T) {
	out := executeCommands(t, "--format", "json")

	var parsed commandsOutput
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw output:\n%s", err, out)
	}

	if parsed.Version == "" {
		t.Error("version should not be empty")
	}
	if len(parsed.Workflow) == 0 {
		t.Error("workflow should not be empty")
	}
	if len(parsed.Commands) == 0 {
		t.Error("commands should not be empty")
	}
	if len(parsed.GlobalFlags) == 0 {
		t.Error("global_flags should not be empty")
	}
}

func TestCommandsOutputText(t *testing.T) {
	out := executeCommands(t, "--format", "text")

	if !strings.Contains(out, "tdd-ai") {
		t.Error("text output should contain 'tdd-ai'")
	}
	if !strings.Contains(out, "Workflow:") {
		t.Error("text output should contain 'Workflow:'")
	}
	if !strings.Contains(out, "Commands:") {
		t.Error("text output should contain 'Commands:'")
	}
	if !strings.Contains(out, "Global flags:") {
		t.Error("text output should contain 'Global flags:'")
	}
	if !strings.Contains(out, "Session:") {
		t.Error("text output should contain 'Session:'")
	}
}

func TestCommandsIncludesAllCommands(t *testing.T) {
	output := buildCommandsOutput()

	expectedCommands := []string{
		"init", "spec add", "spec list", "spec done", "spec pick",
		"phase", "phase next", "phase set",
		"guide", "test", "complete", "status", "reset", "version",
		"refactor", "refactor reflect", "refactor status",
	}

	commandNames := make(map[string]bool)
	for _, cmd := range output.Commands {
		commandNames[cmd.Name] = true
	}

	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("commands output missing %q", expected)
		}
	}
}

func TestCommandsExcludesMetaCommands(t *testing.T) {
	output := buildCommandsOutput()

	for _, cmd := range output.Commands {
		if cmd.Name == "help" || cmd.Name == "commands" || cmd.Name == "completion" {
			t.Errorf("commands output should not include meta-command %q", cmd.Name)
		}
	}
}

func TestCommandsFlagsPresent(t *testing.T) {
	output := buildCommandsOutput()

	for _, cmd := range output.Commands {
		if cmd.Name == "init" {
			flagNames := make(map[string]bool)
			for _, f := range cmd.Flags {
				flagNames[f.Name] = true
			}
			if !flagNames["--retrofit"] {
				t.Error("init command should have --retrofit flag")
			}
			if !flagNames["--test-cmd"] {
				t.Error("init command should have --test-cmd flag")
			}
			return
		}
	}
	t.Error("init command not found")
}

func TestCommandsWorkflowSteps(t *testing.T) {
	steps := workflowSteps()

	if len(steps) != 10 {
		t.Errorf("workflow should have 10 steps, got %d", len(steps))
	}

	if !strings.Contains(steps[0], "init") {
		t.Error("first workflow step should mention init")
	}

	if !strings.Contains(steps[len(steps)-1], "complete") {
		t.Error("last workflow step should mention complete")
	}

	foundReflect := false
	for _, step := range steps {
		if strings.Contains(step, "refactor reflect") {
			foundReflect = true
			break
		}
	}
	if !foundReflect {
		t.Error("workflow should include refactor reflect step")
	}
}

func TestCommandsGlobalFormatFlag(t *testing.T) {
	output := buildCommandsOutput()

	found := false
	for _, f := range output.GlobalFlags {
		if f.Name == "--format" {
			found = true
			break
		}
	}
	if !found {
		t.Error("global flags should include --format")
	}
}

// resetFormatFlag clears the Cobra "changed" state for --format so
// TTY auto-detection in PersistentPreRun works correctly between tests.
func resetFormatFlag() {
	formatFlag = "text"
	if f := rootCmd.PersistentFlags().Lookup("format"); f != nil {
		f.Changed = false
	}
}

func TestCommandsJSONDefaultsNonTTY(t *testing.T) {
	resetFormatFlag()

	// Override isTerminal to simulate non-TTY
	origIsTerminal := isTerminal
	isTerminal = func() bool { return false }
	defer func() { isTerminal = origIsTerminal }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	// Don't pass --format, let TTY detection kick in
	rootCmd.SetArgs([]string{"commands"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("commands command failed: %v", err)
	}

	// Output should be valid JSON since non-TTY defaults to json
	var parsed commandsOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("non-TTY output should be valid JSON: %v\nraw:\n%s", err, buf.String())
	}
}

func TestCommandsTextDefaultsTTY(t *testing.T) {
	resetFormatFlag()

	// Override isTerminal to simulate TTY
	origIsTerminal := isTerminal
	isTerminal = func() bool { return true }
	defer func() { isTerminal = origIsTerminal }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	// Don't pass --format, let TTY detection kick in
	rootCmd.SetArgs([]string{"commands"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("commands command failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Workflow:") {
		t.Errorf("TTY output should be text format, got:\n%s", out)
	}
}

func TestFormatFlagOverridesTTYDetection(t *testing.T) {
	resetFormatFlag()

	// Simulate non-TTY but explicitly pass --format text
	origIsTerminal := isTerminal
	isTerminal = func() bool { return false }
	defer func() { isTerminal = origIsTerminal }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"commands", "--format", "text"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("commands command failed: %v", err)
	}

	out := buf.String()
	// Even though non-TTY, explicit --format text should produce text
	if !strings.Contains(out, "Workflow:") {
		t.Errorf("explicit --format text should produce text even in non-TTY, got:\n%s", out)
	}
}
