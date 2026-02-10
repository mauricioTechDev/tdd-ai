package formatter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/macosta/tdd-ai/internal/types"
)

func TestFormatGuidanceJSON(t *testing.T) {
	g := types.Guidance{
		Phase: types.PhaseRed,
		Mode:  types.ModeGreenfield,
		Specs: []types.Spec{
			{ID: 1, Description: "test spec", Status: types.SpecStatusActive},
		},
		Instructions: []string{"write tests"},
		Rules:        []string{"no implementation"},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	// Verify it is valid JSON
	var parsed types.Guidance
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed.Phase != types.PhaseRed {
		t.Errorf("parsed phase = %q, want %q", parsed.Phase, types.PhaseRed)
	}
	if parsed.Mode != types.ModeGreenfield {
		t.Errorf("parsed mode = %q, want %q", parsed.Mode, types.ModeGreenfield)
	}
	if len(parsed.Specs) != 1 {
		t.Errorf("parsed specs length = %d, want 1", len(parsed.Specs))
	}
}

func TestFormatGuidanceJSONIncludesMode(t *testing.T) {
	g := types.Guidance{
		Phase:        types.PhaseRed,
		Mode:         types.ModeRetrofit,
		Specs:        []types.Spec{},
		Instructions: []string{"verify existing behavior"},
		Rules:        []string{"do not modify implementation"},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["mode"] != "retrofit" {
		t.Errorf("mode = %v, want retrofit", parsed["mode"])
	}
}

func TestFormatGuidanceText(t *testing.T) {
	g := types.Guidance{
		Phase: types.PhaseGreen,
		Mode:  types.ModeGreenfield,
		Specs: []types.Spec{
			{ID: 1, Description: "my feature", Status: types.SpecStatusActive},
		},
		Instructions: []string{"write minimal code", "run tests"},
		Rules:        []string{"do not modify tests"},
	}

	out, err := FormatGuidance(g, FormatText)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	if !strings.Contains(out, "Phase: GREEN") {
		t.Error("text output should contain uppercase phase")
	}
	if !strings.Contains(out, "Mode: greenfield") {
		t.Error("text output should contain mode")
	}
	if !strings.Contains(out, "[1] my feature") {
		t.Error("text output should contain spec")
	}
	if !strings.Contains(out, "write minimal code") {
		t.Error("text output should contain instructions")
	}
	if !strings.Contains(out, "do not modify tests") {
		t.Error("text output should contain rules")
	}
}

func TestFormatGuidanceUnknownFormat(t *testing.T) {
	g := types.Guidance{Phase: types.PhaseRed}
	_, err := FormatGuidance(g, Format("xml"))
	if err == nil {
		t.Error("should return error for unknown format")
	}
}

func TestFormatGuidanceJSONIncludesNextPhase(t *testing.T) {
	g := types.Guidance{
		Phase:        types.PhaseRed,
		Mode:         types.ModeGreenfield,
		NextPhase:    types.PhaseGreen,
		Specs:        []types.Spec{},
		Instructions: []string{"write tests"},
		Rules:        []string{"no impl"},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["next_phase"] != "green" {
		t.Errorf("next_phase = %v, want green", parsed["next_phase"])
	}
}

func TestFormatGuidanceJSONIncludesTestCmd(t *testing.T) {
	g := types.Guidance{
		Phase:        types.PhaseRed,
		Mode:         types.ModeGreenfield,
		TestCmd:      "npm test",
		Specs:        []types.Spec{},
		Instructions: []string{"write tests"},
		Rules:        []string{"no impl"},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["test_cmd"] != "npm test" {
		t.Errorf("test_cmd = %v, want 'npm test'", parsed["test_cmd"])
	}
}

func TestFormatGuidanceJSONOmitsEmptyTestCmd(t *testing.T) {
	g := types.Guidance{
		Phase:        types.PhaseRed,
		Mode:         types.ModeGreenfield,
		Specs:        []types.Spec{},
		Instructions: []string{"write tests"},
		Rules:        []string{"no impl"},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, exists := parsed["test_cmd"]; exists {
		t.Error("test_cmd should be omitted from JSON when empty")
	}
}

func TestFormatGuidanceTextIncludesNextPhase(t *testing.T) {
	g := types.Guidance{
		Phase:        types.PhaseRed,
		Mode:         types.ModeGreenfield,
		NextPhase:    types.PhaseGreen,
		Instructions: []string{"write tests"},
		Rules:        []string{"no impl"},
	}

	out, err := FormatGuidance(g, FormatText)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	if !strings.Contains(out, "Next Phase: GREEN") {
		t.Errorf("text output should contain next phase, got:\n%s", out)
	}
}

func TestFormatGuidanceTextIncludesTestCmd(t *testing.T) {
	g := types.Guidance{
		Phase:        types.PhaseRed,
		Mode:         types.ModeGreenfield,
		TestCmd:      "dotnet test",
		Instructions: []string{"write tests"},
		Rules:        []string{"no impl"},
	}

	out, err := FormatGuidance(g, FormatText)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	if !strings.Contains(out, "Test Command: dotnet test") {
		t.Errorf("text output should contain test command, got:\n%s", out)
	}
}

func TestFormatStatusJSON(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("active spec")
	s.AddSpec("done spec")
	_ = s.CompleteSpec(2)

	out, err := FormatStatus(s, FormatJSON)
	if err != nil {
		t.Fatalf("FormatStatus() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["phase"] != "red" {
		t.Errorf("phase = %v, want red", parsed["phase"])
	}
	if parsed["total_specs"] != float64(2) {
		t.Errorf("total_specs = %v, want 2", parsed["total_specs"])
	}
	if parsed["active_specs"] != float64(1) {
		t.Errorf("active_specs = %v, want 1", parsed["active_specs"])
	}
	if parsed["done_specs"] != float64(1) {
		t.Errorf("done_specs = %v, want 1", parsed["done_specs"])
	}
}

func TestFormatStatusText(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("my feature")

	out, err := FormatStatus(s, FormatText)
	if err != nil {
		t.Fatalf("FormatStatus() error: %v", err)
	}

	if !strings.Contains(out, "Phase: RED") {
		t.Error("text output should contain phase")
	}
	if !strings.Contains(out, "1 total") {
		t.Error("text output should contain spec count")
	}
	if !strings.Contains(out, "(active) my feature") {
		t.Error("text output should contain spec with status")
	}
}

func TestFormatGuidanceTextSortsByID(t *testing.T) {
	g := types.Guidance{
		Phase: types.PhaseRed,
		Mode:  types.ModeGreenfield,
		Specs: []types.Spec{
			{ID: 3, Description: "third", Status: types.SpecStatusActive},
			{ID: 1, Description: "first", Status: types.SpecStatusActive},
			{ID: 2, Description: "second", Status: types.SpecStatusActive},
		},
		Instructions: []string{"write tests"},
		Rules:        []string{"no impl"},
	}

	out, err := FormatGuidance(g, FormatText)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	idx1 := strings.Index(out, "[1] first")
	idx2 := strings.Index(out, "[2] second")
	idx3 := strings.Index(out, "[3] third")
	if idx1 == -1 || idx2 == -1 || idx3 == -1 {
		t.Fatalf("output missing specs, got:\n%s", out)
	}
	if !(idx1 < idx2 && idx2 < idx3) {
		t.Errorf("specs should be sorted by ID, got:\n%s", out)
	}
}

func TestFormatFullStatusText(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature A")
	s.AddSpec("feature B")
	_ = s.CompleteSpec(2)

	out, err := FormatFullStatus(s, FormatText)
	if err != nil {
		t.Fatalf("FormatFullStatus() error: %v", err)
	}

	if !strings.Contains(out, "Phase: RED") {
		t.Error("should contain phase")
	}
	if !strings.Contains(out, "Mode: greenfield") {
		t.Error("should contain mode")
	}
	if !strings.Contains(out, "2 total") {
		t.Error("should contain total count")
	}
	if !strings.Contains(out, "1 active") {
		t.Error("should contain active count")
	}
	if !strings.Contains(out, "1 done") {
		t.Error("should contain done count")
	}
	if !strings.Contains(out, "(active) feature A") {
		t.Error("should contain active spec")
	}
	if !strings.Contains(out, "(done) feature B") {
		t.Error("should contain done spec")
	}
	if !strings.Contains(out, "Next:") {
		t.Error("should contain next action hint")
	}
}

func TestFormatFullStatusJSON(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature A")

	out, err := FormatFullStatus(s, FormatJSON)
	if err != nil {
		t.Fatalf("FormatFullStatus() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["phase"] != "red" {
		t.Errorf("phase = %v, want red", parsed["phase"])
	}
	if parsed["mode"] != "greenfield" {
		t.Errorf("mode = %v, want greenfield", parsed["mode"])
	}
	if parsed["next_action"] == nil {
		t.Error("should contain next_action field")
	}
}

func TestFormatFullStatusShowsTestCmd(t *testing.T) {
	s := types.NewSession()
	s.TestCmd = "go test ./..."
	s.AddSpec("feature A")

	textOut, err := FormatFullStatus(s, FormatText)
	if err != nil {
		t.Fatalf("FormatFullStatus(text) error: %v", err)
	}
	if !strings.Contains(textOut, "Test Command: go test ./...") {
		t.Errorf("text output should contain test command, got:\n%s", textOut)
	}

	jsonOut, err := FormatFullStatus(s, FormatJSON)
	if err != nil {
		t.Fatalf("FormatFullStatus(json) error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["test_cmd"] != "go test ./..." {
		t.Errorf("test_cmd = %v, want 'go test ./...'", parsed["test_cmd"])
	}
}

func TestFormatFullStatusOmitsTestCmdWhenEmpty(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature")

	textOut, err := FormatFullStatus(s, FormatText)
	if err != nil {
		t.Fatalf("FormatFullStatus(text) error: %v", err)
	}
	if strings.Contains(textOut, "Test Command:") {
		t.Error("text output should not contain Test Command when not configured")
	}

	jsonOut, err := FormatFullStatus(s, FormatJSON)
	if err != nil {
		t.Fatalf("FormatFullStatus(json) error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, exists := parsed["test_cmd"]; exists {
		t.Error("test_cmd should be omitted from JSON when empty")
	}
}

func TestFormatFullStatusNextActionNoSpecs(t *testing.T) {
	s := types.NewSession()

	out, err := FormatFullStatus(s, FormatText)
	if err != nil {
		t.Fatalf("FormatFullStatus() error: %v", err)
	}

	if !strings.Contains(out, "add specs") {
		t.Errorf("next action should suggest adding specs when none exist, got:\n%s", out)
	}
}

func TestNextActionBatchSyntax(t *testing.T) {
	s := types.NewSession()

	out, err := FormatFullStatus(s, FormatText)
	if err != nil {
		t.Fatalf("FormatFullStatus() error: %v", err)
	}

	if !strings.Contains(out, `"desc1" "desc2"`) {
		t.Errorf("next action should show batch spec syntax, got:\n%s", out)
	}
}

func TestNextActionDonePhaseMentionsAll(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseDone
	s.AddSpec("feature")

	out, err := FormatFullStatus(s, FormatText)
	if err != nil {
		t.Fatalf("FormatFullStatus() error: %v", err)
	}

	if !strings.Contains(out, "spec done --all") {
		t.Errorf("done phase next action should mention --all flag, got:\n%s", out)
	}
}

func TestNextActionDonePhaseAllSpecsCompleted(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseDone
	s.AddSpec("feature A")
	s.AddSpec("feature B")
	_ = s.CompleteSpec(1)
	_ = s.CompleteSpec(2)

	out, err := FormatFullStatus(s, FormatText)
	if err != nil {
		t.Fatalf("FormatFullStatus() error: %v", err)
	}

	if !strings.Contains(out, "all specs complete") {
		t.Errorf("done phase with all specs completed should say 'all specs complete', got:\n%s", out)
	}
	if strings.Contains(out, "spec done") {
		t.Errorf("done phase with all specs completed should NOT suggest 'spec done', got:\n%s", out)
	}
}

func TestFormatStatusTextSortsByID(t *testing.T) {
	s := types.NewSession()
	// Add specs in a way that results in non-sequential order
	s.Specs = []types.Spec{
		{ID: 5, Description: "fifth", Status: types.SpecStatusActive},
		{ID: 2, Description: "second", Status: types.SpecStatusCompleted},
		{ID: 7, Description: "seventh", Status: types.SpecStatusActive},
		{ID: 1, Description: "first", Status: types.SpecStatusActive},
	}

	out, err := FormatStatus(s, FormatText)
	if err != nil {
		t.Fatalf("FormatStatus() error: %v", err)
	}

	idx1 := strings.Index(out, "[1]")
	idx2 := strings.Index(out, "[2]")
	idx5 := strings.Index(out, "[5]")
	idx7 := strings.Index(out, "[7]")
	if idx1 == -1 || idx2 == -1 || idx5 == -1 || idx7 == -1 {
		t.Fatalf("output missing specs, got:\n%s", out)
	}
	if !(idx1 < idx2 && idx2 < idx5 && idx5 < idx7) {
		t.Errorf("specs should be sorted by ID, got:\n%s", out)
	}
}
