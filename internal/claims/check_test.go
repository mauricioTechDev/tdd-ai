package claims

import (
	"strings"
	"testing"
)

const expectedOut = "{\"total\":100}\n"

var invoiceRun = []string{"node", "skills/invoice/reconcile.mjs", "invoices/march.csv"}

func verifiedChecks() *Checks {
	return &Checks{
		Run:          invoiceRun,
		Pins:         map[string]string{"skills/invoice/SKILL.md": "h1", "skills/invoice/reconcile.mjs": "h2"},
		Reads:        []string{"invoices/**", "skills/invoice/**"},
		Writes:       []string{},
		External:     []string{},
		Network:      []string{},
		Execs:        []string{"node"},
		ExitCode:     0,
		Stdout:       ".tdd-ai/expected/1.out",
		StdoutSHA256: HashBytes([]byte(expectedOut)),
	}
}

func cleanObservation() Observation {
	return Observation{
		Run: invoiceRun,
		Evidence: &Evidence{
			Reads:          []string{"invoices/march.csv", "skills/invoice/reconcile.mjs"},
			Writes:         []string{},
			ExternalReads:  []string{},
			ExternalWrites: []string{},
			Network:        []string{},
			Execs:          []string{"node"},
		},
		Traced:   true,
		PinsNow:  map[string]string{"skills/invoice/SKILL.md": "h1", "skills/invoice/reconcile.mjs": "h2"},
		Stdout:   []byte(expectedOut),
		Expected: []byte(expectedOut),
	}
}

func failureChecks(out Outcome) []string {
	var got []string
	for _, f := range out.Failures {
		got = append(got, f.Check+": "+f.Detail)
	}
	return got
}

func TestEvaluatePassesACleanRun(t *testing.T) {
	out := Evaluate(verifiedChecks(), cleanObservation())
	if !out.Pass || !out.PolicyChecked || len(out.Failures) != 0 {
		t.Errorf("Evaluate = %+v, want a pass with the policy checked", out)
	}
}

func TestEvaluateCatchesARugPull(t *testing.T) {
	obs := cleanObservation()
	obs.PinsNow["skills/invoice/reconcile.mjs"] = "edited"
	obs.Evidence.Network = []string{"socket:inet", "203.0.113.7:443"}
	obs.Evidence.ExternalReads = []string{"~/.ssh/id_rsa"}

	out := Evaluate(verifiedChecks(), obs)
	got := failureChecks(out)
	want := []string{
		"pin: skills/invoice/reconcile.mjs changed since the claim was verified",
		"outside: read ~/.ssh/id_rsa outside the project",
		"network: socket:inet (allowed: none)",
		"network: 203.0.113.7:443 (allowed: none)",
	}
	if out.Pass || strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("failures =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestEvaluateCatchesReadsWritesProgramsAndExit(t *testing.T) {
	obs := cleanObservation()
	obs.Evidence.Reads = append(obs.Evidence.Reads, ".env")
	obs.Evidence.Writes = []string{"invoices/march.csv"}
	obs.Evidence.ExternalWrites = []string{"~/.bashrc"}
	obs.Evidence.Execs = []string{"node", "curl"}
	obs.Evidence.ExitCode = 1

	got := failureChecks(Evaluate(verifiedChecks(), obs))
	want := []string{
		"exit: exited with 1, want 0",
		"read: .env is outside the allowed reads (invoices/**, skills/invoice/**)",
		"write: invoices/march.csv is outside the allowed writes (none)",
		"outside: wrote ~/.bashrc outside the project",
		"program: ran curl (allowed: node)",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("failures =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestEvaluateComparesOutputLineByLine(t *testing.T) {
	obs := cleanObservation()
	obs.Stdout = []byte("{\"total\":99}\n")
	got := failureChecks(Evaluate(verifiedChecks(), obs))
	if len(got) != 1 || got[0] != `stdout: line 1 differs: got "{\"total\":99}", want "{\"total\":100}"` {
		t.Errorf("failures = %v", got)
	}
}

func TestEvaluateReportsMissingOrEditedExpectations(t *testing.T) {
	obs := cleanObservation()
	obs.Expected = nil
	delete(obs.PinsNow, "skills/invoice/SKILL.md")
	got := failureChecks(Evaluate(verifiedChecks(), obs))
	want := []string{
		"pin: skills/invoice/SKILL.md is missing",
		"stdout: expected output .tdd-ai/expected/1.out is missing",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("failures = %v", got)
	}

	obs = cleanObservation()
	obs.Expected = []byte("edited\n")
	obs.Stdout = []byte("edited\n")
	got = failureChecks(Evaluate(verifiedChecks(), obs))
	if len(got) != 1 || !strings.Contains(got[0], "was edited after the claim was verified") {
		t.Errorf("failures = %v", got)
	}
}

func TestEvaluateWithoutTraceSkipsThePolicy(t *testing.T) {
	obs := cleanObservation()
	obs.Traced = false
	obs.Evidence.Network = []string{"203.0.113.7:443"}
	out := Evaluate(verifiedChecks(), obs)
	if !out.Pass || out.PolicyChecked {
		t.Errorf("Evaluate = %+v, want a pass with the policy not checked", out)
	}
}

func TestFirstDifferenceReportsLengthChanges(t *testing.T) {
	cases := map[string][2]string{
		"output has 2 lines, want 1":                                       {"a\nb\n", "a\n"},
		"output has 0 lines, want 1":                                       {"", "a\n"},
		"output differs only in its final newline":                         {"a", "a\n"},
		`line 2 differs: got "c", want "b"`:                                {"a\nc\n", "a\nb\n"},
		`line 1 differs: got "` + strings.Repeat("x", 60) + `…", want "y"`: {strings.Repeat("x", 70), "y"},
	}
	for want, in := range cases {
		if got := firstDifference([]byte(in[0]), []byte(in[1])); got != want {
			t.Errorf("firstDifference(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}

func TestEvaluateCatchesAChangedCommand(t *testing.T) {
	obs := cleanObservation()
	obs.Run = []string{"cat", ".tdd-ai/expected/1.out"}
	got := failureChecks(Evaluate(verifiedChecks(), obs))
	want := `command: changed since the claim was verified: now "cat .tdd-ai/expected/1.out", verified "node skills/invoice/reconcile.mjs invoices/march.csv"`
	if len(got) != 1 || got[0] != want {
		t.Errorf("failures = %v", got)
	}
}
