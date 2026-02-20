package formatter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/macosta/tdd-ai/internal/types"
)

func TestFormatGuidanceJSON(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "fail",
		Specs: []types.Spec{
			{ID: 1, Description: "test spec", Status: types.SpecStatusActive},
		},
		Blockers: []string{"No test result recorded"},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

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
	if parsed.ExpectedTestResult != "fail" {
		t.Errorf("parsed expected_test_result = %q, want %q", parsed.ExpectedTestResult, "fail")
	}
}

func TestFormatGuidanceJSONIncludesMode(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseRed,
		Mode:               types.ModeRetrofit,
		ExpectedTestResult: "pass",
		Specs:              []types.Spec{},
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
		Phase:              types.PhaseGreen,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "pass",
		Specs: []types.Spec{
			{ID: 1, Description: "my feature", Status: types.SpecStatusActive},
		},
		Blockers: []string{"No test result recorded"},
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
	if !strings.Contains(out, "Expected Test Result: pass") {
		t.Error("text output should contain expected test result")
	}
	if !strings.Contains(out, "Blockers:") {
		t.Error("text output should contain blockers section")
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
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		NextPhase:          types.PhaseGreen,
		ExpectedTestResult: "fail",
		Specs:              []types.Spec{},
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
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		TestCmd:            "npm test",
		ExpectedTestResult: "fail",
		Specs:              []types.Spec{},
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
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "fail",
		Specs:              []types.Spec{},
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
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		NextPhase:          types.PhaseGreen,
		ExpectedTestResult: "fail",
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
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		TestCmd:            "dotnet test",
		ExpectedTestResult: "fail",
	}

	out, err := FormatGuidance(g, FormatText)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	if !strings.Contains(out, "Test Command: dotnet test") {
		t.Errorf("text output should contain test command, got:\n%s", out)
	}
}

func TestFormatGuidanceTextIncludesExpectedTestResult(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "fail",
	}

	out, err := FormatGuidance(g, FormatText)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	if !strings.Contains(out, "Expected Test Result: fail") {
		t.Errorf("text output should contain expected test result, got:\n%s", out)
	}
}

func TestFormatGuidanceJSONIncludesExpectedTestResult(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "fail",
		Specs:              []types.Spec{},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["expected_test_result"] != "fail" {
		t.Errorf("expected_test_result = %v, want 'fail'", parsed["expected_test_result"])
	}
}

func TestFormatGuidanceJSONIncludesBlockers(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "fail",
		Specs:              []types.Spec{},
		Blockers:           []string{"No spec selected", "No test result recorded"},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	blockers, ok := parsed["blockers"].([]interface{})
	if !ok {
		t.Fatal("blockers field should be present in JSON output")
	}
	if len(blockers) != 2 {
		t.Errorf("blockers length = %d, want 2", len(blockers))
	}
}

func TestFormatGuidanceJSONOmitsEmptyBlockers(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseGreen,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "pass",
		Specs:              []types.Spec{},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, exists := parsed["blockers"]; exists {
		t.Error("blockers should be omitted from JSON when empty")
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
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "fail",
		Specs: []types.Spec{
			{ID: 3, Description: "third", Status: types.SpecStatusActive},
			{ID: 1, Description: "first", Status: types.SpecStatusActive},
			{ID: 2, Description: "second", Status: types.SpecStatusActive},
		},
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
	if idx1 >= idx2 || idx2 >= idx3 {
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

func TestFormatGuidanceTextIncludesReflections(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseRefactor,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "pass",
		Reflections: []types.ReflectionQuestion{
			{ID: 1, Question: "Can I improve tests?", Answer: "Tests are already descriptive and clear enough"},
			{ID: 2, Question: "Are tests isolated?", Answer: ""},
		},
	}

	out, err := FormatGuidance(g, FormatText)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	if !strings.Contains(out, "Reflections (1/2 answered)") {
		t.Errorf("text output should contain reflections header, got:\n%s", out)
	}
	if !strings.Contains(out, "(answered) Can I improve tests?") {
		t.Errorf("text output should show answered question, got:\n%s", out)
	}
	if !strings.Contains(out, "(pending) Are tests isolated?") {
		t.Errorf("text output should show pending question, got:\n%s", out)
	}
	if !strings.Contains(out, "Tests are already descriptive and clear enough") {
		t.Errorf("text output should show answer text, got:\n%s", out)
	}
}

func TestFormatGuidanceJSONIncludesReflections(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseRefactor,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "pass",
		Specs:              []types.Spec{},
		Reflections: []types.ReflectionQuestion{
			{ID: 1, Question: "Q1", Answer: "answered with enough words here"},
			{ID: 2, Question: "Q2", Answer: ""},
		},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	reflections, ok := parsed["reflections"].([]interface{})
	if !ok {
		t.Fatal("reflections field should be present in JSON output")
	}
	if len(reflections) != 2 {
		t.Errorf("reflections length = %d, want 2", len(reflections))
	}
}

func TestFormatGuidanceJSONOmitsEmptyReflections(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "fail",
		Specs:              []types.Spec{},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if _, exists := parsed["reflections"]; exists {
		t.Error("reflections should be omitted from JSON when empty")
	}
}

func TestFormatGuidanceTextNoReflectionsSection(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "fail",
	}

	out, err := FormatGuidance(g, FormatText)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	if strings.Contains(out, "Reflections") {
		t.Errorf("text output should not contain Reflections section when empty, got:\n%s", out)
	}
}

func TestFormatGuidanceTextShowsCurrentSpec(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "fail",
		CurrentSpec:        &types.Spec{ID: 2, Description: "my current spec", Status: types.SpecStatusActive},
		Iteration:          3,
		Specs: []types.Spec{
			{ID: 2, Description: "my current spec", Status: types.SpecStatusActive},
		},
	}

	out, err := FormatGuidance(g, FormatText)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	if !strings.Contains(out, "Current Spec: [2] my current spec") {
		t.Errorf("text output should show current spec, got:\n%s", out)
	}
	if !strings.Contains(out, "Iteration: 3") {
		t.Errorf("text output should show iteration, got:\n%s", out)
	}
}

func TestFormatGuidanceJSONIncludesCurrentSpec(t *testing.T) {
	g := types.Guidance{
		Phase:              types.PhaseRed,
		Mode:               types.ModeGreenfield,
		ExpectedTestResult: "fail",
		CurrentSpec:        &types.Spec{ID: 1, Description: "test spec", Status: types.SpecStatusActive},
		Iteration:          2,
		TotalSpecs:         3,
		Specs:              []types.Spec{},
	}

	out, err := FormatGuidance(g, FormatJSON)
	if err != nil {
		t.Fatalf("FormatGuidance() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["iteration"] != float64(2) {
		t.Errorf("iteration = %v, want 2", parsed["iteration"])
	}
	if parsed["total_specs"] != float64(3) {
		t.Errorf("total_specs = %v, want 3", parsed["total_specs"])
	}
	cs, ok := parsed["current_spec"].(map[string]interface{})
	if !ok {
		t.Fatal("current_spec should be present in JSON output")
	}
	if cs["id"] != float64(1) {
		t.Errorf("current_spec.id = %v, want 1", cs["id"])
	}
}

func TestFormatFullStatusIncludesCurrentSpecAndIteration(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature A")
	s.AddSpec("feature B")
	_ = s.SetCurrentSpec(1)
	s.Iteration = 2

	jsonOut, err := FormatFullStatus(s, FormatJSON)
	if err != nil {
		t.Fatalf("FormatFullStatus(json) error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["current_spec_id"] != float64(1) {
		t.Errorf("current_spec_id = %v, want 1", parsed["current_spec_id"])
	}
	if parsed["iteration"] != float64(2) {
		t.Errorf("iteration = %v, want 2", parsed["iteration"])
	}

	textOut, err := FormatFullStatus(s, FormatText)
	if err != nil {
		t.Fatalf("FormatFullStatus(text) error: %v", err)
	}
	if !strings.Contains(textOut, "Current Spec: [1] feature A") {
		t.Errorf("text output should show current spec, got:\n%s", textOut)
	}
	if !strings.Contains(textOut, "Iteration: 2") {
		t.Errorf("text output should show iteration, got:\n%s", textOut)
	}
}

func TestFormatStatusTextMarksCurrentSpec(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("first")
	s.AddSpec("second")
	s.AddSpec("third")
	_ = s.SetCurrentSpec(2)

	out, err := FormatStatus(s, FormatText)
	if err != nil {
		t.Fatalf("FormatStatus() error: %v", err)
	}

	if !strings.Contains(out, "→ [2]") {
		t.Errorf("should mark current spec with arrow, got:\n%s", out)
	}
	if !strings.Contains(out, "(current)") {
		t.Errorf("should mark current spec with (current), got:\n%s", out)
	}
	if strings.Contains(out, "→ [1]") {
		t.Errorf("non-current spec should not have arrow, got:\n%s", out)
	}
}

func TestFormatResumeJSONStructure(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature A")
	s.AddSpec("feature B")
	_ = s.SetCurrentSpec(1)

	out, err := FormatResume(s, FormatJSON)
	if err != nil {
		t.Fatalf("FormatResume() error: %v", err)
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
		t.Error("next_action field should be present")
	}
	if parsed["remaining_specs"] != float64(1) {
		t.Errorf("remaining_specs = %v, want 1", parsed["remaining_specs"])
	}
}

func TestFormatResumeTextOutput(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("user can login")
	_ = s.SetCurrentSpec(1)

	out, err := FormatResume(s, FormatText)
	if err != nil {
		t.Fatalf("FormatResume() error: %v", err)
	}

	if !strings.Contains(out, "=== TDD Session Checkpoint ===") {
		t.Error("text output should contain checkpoint header")
	}
	if !strings.Contains(out, "Phase: RED") {
		t.Error("text output should contain uppercase phase")
	}
	if !strings.Contains(out, "Working on: [1] user can login") {
		t.Error("text output should contain current spec")
	}
	if !strings.Contains(out, "NEXT ACTION:") {
		t.Error("text output should contain NEXT ACTION section")
	}
}

func TestFormatResumeTextRedNoSpecPicked(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature A")
	s.AddSpec("feature B")

	out, err := FormatResume(s, FormatText)
	if err != nil {
		t.Fatalf("FormatResume() error: %v", err)
	}

	if !strings.Contains(out, "Working on: (no spec selected)") {
		t.Errorf("text output should show no spec selected, got:\n%s", out)
	}
	if !strings.Contains(out, "BLOCKERS:") {
		t.Errorf("text output should show blockers when no spec picked, got:\n%s", out)
	}
	if !strings.Contains(out, "tdd-ai spec pick 1") {
		t.Errorf("next action should suggest spec pick, got:\n%s", out)
	}
}

func TestFormatResumeTextRefactorWithPendingReflections(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.AddSpec("feature A")
	_ = s.SetCurrentSpec(1)
	s.Reflections = []types.ReflectionQuestion{
		{ID: 1, Question: "Q1", Answer: "already answered this one"},
		{ID: 2, Question: "Q2", Answer: ""},
		{ID: 3, Question: "Q3", Answer: ""},
	}

	out, err := FormatResume(s, FormatText)
	if err != nil {
		t.Fatalf("FormatResume() error: %v", err)
	}

	if !strings.Contains(out, "BLOCKERS:") {
		t.Errorf("text output should show blockers with pending reflections, got:\n%s", out)
	}
	if !strings.Contains(out, "2 reflection questions unanswered") {
		t.Errorf("text output should report pending count, got:\n%s", out)
	}
	if !strings.Contains(out, `tdd-ai refactor reflect 2`) {
		t.Errorf("next action should target first pending reflection, got:\n%s", out)
	}
}

func TestFormatResumeTextGreenPhase(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseGreen
	s.AddSpec("feature A")
	_ = s.SetCurrentSpec(1)

	out, err := FormatResume(s, FormatText)
	if err != nil {
		t.Fatalf("FormatResume() error: %v", err)
	}

	// GREEN phase with no test result will have blockers now
	if !strings.Contains(out, "tdd-ai test && tdd-ai phase next") {
		t.Errorf("GREEN next action should be test && phase next, got:\n%s", out)
	}
}

func TestFormatResumeTextNoSpecs(t *testing.T) {
	s := types.NewSession()

	out, err := FormatResume(s, FormatText)
	if err != nil {
		t.Fatalf("FormatResume() error: %v", err)
	}

	if !strings.Contains(out, "tdd-ai spec add") {
		t.Errorf("no specs: next action should suggest spec add, got:\n%s", out)
	}
}

func TestFormatResumeUnknownFormat(t *testing.T) {
	s := types.NewSession()
	_, err := FormatResume(s, Format("xml"))
	if err == nil {
		t.Error("should return error for unknown format")
	}
}

func TestFormatResumeTextShowsIterationWhenNonZero(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseGreen
	s.Iteration = 3
	s.AddSpec("feature A")
	_ = s.SetCurrentSpec(1)

	out, err := FormatResume(s, FormatText)
	if err != nil {
		t.Fatalf("FormatResume() error: %v", err)
	}

	if !strings.Contains(out, "Iteration: 3") {
		t.Errorf("text output should show iteration when non-zero, got:\n%s", out)
	}
}

func TestFormatResumeTextShowsRecentEvents(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseGreen
	s.AddSpec("feature")
	_ = s.SetCurrentSpec(1)
	s.AddEvent("phase_next", func(e *types.Event) {
		e.From = "red"
		e.To = "green"
		e.Result = "fail"
	})

	out, err := FormatResume(s, FormatText)
	if err != nil {
		t.Fatalf("FormatResume() error: %v", err)
	}

	if !strings.Contains(out, "Recent events:") {
		t.Errorf("text output should show recent events section, got:\n%s", out)
	}
	if !strings.Contains(out, "phase_next (red -> green) [fail]") {
		t.Errorf("text output should show event details, got:\n%s", out)
	}
}

func TestFormatResumeJSONIncludesBlockers(t *testing.T) {
	s := types.NewSession()
	s.AddSpec("feature A")
	s.AddSpec("feature B")
	// No spec selected — should have blockers

	out, err := FormatResume(s, FormatJSON)
	if err != nil {
		t.Fatalf("FormatResume() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	blockers, ok := parsed["blockers"].([]interface{})
	if !ok {
		t.Fatal("blockers should be present when there are blockers")
	}
	if len(blockers) == 0 {
		t.Error("should have at least one blocker when no spec selected")
	}
}

func TestFormatResumeJSONIncludesCurrentSpec(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseGreen
	s.AddSpec("user can login")
	_ = s.SetCurrentSpec(1)

	out, err := FormatResume(s, FormatJSON)
	if err != nil {
		t.Fatalf("FormatResume() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	cs, ok := parsed["current_spec"].(map[string]interface{})
	if !ok {
		t.Fatal("current_spec should be present in JSON output")
	}
	if cs["description"] != "user can login" {
		t.Errorf("current_spec.description = %v, want 'user can login'", cs["description"])
	}
}

func TestFormatResumeTextRefactorAllAnswered(t *testing.T) {
	s := types.NewSession()
	s.Phase = types.PhaseRefactor
	s.AddSpec("feature A")
	_ = s.SetCurrentSpec(1)
	s.LastTestResult = "pass"
	s.Reflections = []types.ReflectionQuestion{
		{ID: 1, Question: "Q1", Answer: "answered with enough words here"},
		{ID: 2, Question: "Q2", Answer: "also answered with enough words"},
	}

	out, err := FormatResume(s, FormatText)
	if err != nil {
		t.Fatalf("FormatResume() error: %v", err)
	}

	if strings.Contains(out, "BLOCKERS:") {
		t.Errorf("REFACTOR with all reflections answered and pass result should have no blockers, got:\n%s", out)
	}
	if !strings.Contains(out, "tdd-ai test && tdd-ai phase next") {
		t.Errorf("all reflections answered: next action should be test && phase next, got:\n%s", out)
	}
}

func TestFormatStatusTextSortsByID(t *testing.T) {
	s := types.NewSession()
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
	if idx1 >= idx2 || idx2 >= idx5 || idx5 >= idx7 {
		t.Errorf("specs should be sorted by ID, got:\n%s", out)
	}
}
