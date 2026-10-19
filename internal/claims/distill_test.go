package claims

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func quietEvidence() *Evidence {
	return &Evidence{
		ExitCode:       0,
		StdoutSHA256:   "out-hash",
		Reads:          []string{"invoices/march.csv", "skills/invoice/reconcile.mjs"},
		Writes:         []string{},
		ExternalReads:  []string{},
		ExternalWrites: []string{},
		Network:        []string{},
		Execs:          []string{"node"},
		Pins:           map[string]string{"skills/invoice/SKILL.md": "h1", "skills/invoice/reconcile.mjs": "h2"},
	}
}

func TestDistillProposesChecksFromQuietEvidence(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	run := []string{"node", "skills/invoice/reconcile.mjs", "invoices/march.csv"}
	ch, refusals := Distill(quietEvidence(), Options{Run: run, StdoutPath: ".tdd-ai/expected/1.out", Now: now})
	if len(refusals) != 0 {
		t.Fatalf("unexpected refusals: %v", refusals)
	}
	want := &Checks{
		DistilledAt:  "2026-10-19T12:00:00Z",
		Run:          run,
		Pins:         map[string]string{"skills/invoice/SKILL.md": "h1", "skills/invoice/reconcile.mjs": "h2"},
		Reads:        []string{"invoices/**", "skills/invoice/**"},
		Writes:       []string{},
		External:     []string{},
		Network:      []string{},
		Execs:        []string{"node"},
		ExitCode:     0,
		Stdout:       ".tdd-ai/expected/1.out",
		StdoutSHA256: "out-hash",
	}
	if !reflect.DeepEqual(ch, want) {
		t.Errorf("Distill =\n%+v\nwant\n%+v", ch, want)
	}
}

func TestDistillRefusesNetworkActivityUnlessAllowed(t *testing.T) {
	ev := quietEvidence()
	ev.Network = []string{"93.184.216.34:443", "socket:inet"}

	_, refusals := Distill(ev, Options{})
	if len(refusals) != 1 || !strings.Contains(refusals[0], "93.184.216.34:443, socket:inet") {
		t.Fatalf("refusals = %v, want one naming both entries", refusals)
	}

	ch, refusals := Distill(ev, Options{Network: []string{"93.184.216.34:443"}})
	if len(refusals) != 0 {
		t.Fatalf("allowing the address should also allow its socket: %v", refusals)
	}
	if !reflect.DeepEqual(ch.Network, []string{"93.184.216.34:443"}) {
		t.Errorf("Network = %v", ch.Network)
	}
}

func TestDistillRefusesPathsOutsideTheProjectUnlessAllowed(t *testing.T) {
	ev := quietEvidence()
	ev.ExternalReads = []string{"~/.ssh/id_rsa"}
	ev.ExternalWrites = []string{"~/.bashrc"}

	_, refusals := Distill(ev, Options{})
	if len(refusals) != 2 {
		t.Fatalf("refusals = %v, want one for the read and one for the write", refusals)
	}
	if _, refusals := Distill(ev, Options{External: []string{"~/.ssh/**", "~/.bashrc"}}); len(refusals) != 0 {
		t.Errorf("explicitly allowed paths should be accepted: %v", refusals)
	}
}

func TestDistillRefusesOtherProgramsUnlessAllowed(t *testing.T) {
	ev := quietEvidence()
	ev.Execs = []string{"sh", "curl"}

	_, refusals := Distill(ev, Options{})
	if len(refusals) != 1 || !strings.Contains(refusals[0], "curl") {
		t.Fatalf("refusals = %v, want one naming curl", refusals)
	}
	ch, refusals := Distill(ev, Options{Execs: []string{"curl"}})
	if len(refusals) != 0 || !reflect.DeepEqual(ch.Execs, []string{"sh", "curl"}) {
		t.Errorf("Execs = %v, refusals = %v", ch.Execs, refusals)
	}
}

func TestDistillRejectsOverridesThatContradictTheEvidence(t *testing.T) {
	ev := quietEvidence()
	ev.Writes = []string{"out/report.json"}

	_, refusals := Distill(ev, Options{Reads: []string{"invoices/**"}, Writes: []string{"tmp/**"}})
	if len(refusals) != 2 {
		t.Fatalf("refusals = %v, want the uncovered read and write", refusals)
	}
	if !strings.Contains(refusals[0], "out/report.json") || !strings.Contains(refusals[1], "skills/invoice/reconcile.mjs") {
		t.Errorf("refusals = %v", refusals)
	}

	ch, refusals := Distill(ev, Options{Reads: []string{"invoices/**", "skills/**"}, Writes: []string{"out/*.json"}})
	if len(refusals) != 0 {
		t.Fatalf("covering overrides should be accepted: %v", refusals)
	}
	if !reflect.DeepEqual(ch.Reads, []string{"invoices/**", "skills/**"}) || !reflect.DeepEqual(ch.Writes, []string{"out/*.json"}) {
		t.Errorf("Reads/Writes = %v / %v", ch.Reads, ch.Writes)
	}
}

func TestDistillRefusesWhenAPinnedFileChangedSinceVerify(t *testing.T) {
	ev := quietEvidence()
	_, refusals := Distill(ev, Options{PinsNow: map[string]string{
		"skills/invoice/SKILL.md":      "h1",
		"skills/invoice/reconcile.mjs": "edited",
	}})
	if len(refusals) != 1 || !strings.Contains(refusals[0], "skills/invoice/reconcile.mjs") {
		t.Errorf("refusals = %v", refusals)
	}
}

func TestDistillCanSkipTheOutputCheck(t *testing.T) {
	ch, _ := Distill(quietEvidence(), Options{StdoutPath: "x", SkipStdout: true})
	if ch.Stdout != "" || ch.StdoutSHA256 != "" {
		t.Errorf("Stdout = %q, %q; want both empty", ch.Stdout, ch.StdoutSHA256)
	}
}

func TestNetworkAllowed(t *testing.T) {
	cases := []struct {
		entry   string
		allowed []string
		want    bool
	}{
		{"127.0.0.1:5432", []string{"127.0.0.1:5432"}, true},
		{"127.0.0.1:5433", []string{"127.0.0.1:5432"}, false},
		{"socket:inet", []string{"127.0.0.1:5432"}, true},
		{"socket:inet", []string{"[::1]:5432"}, false},
		{"socket:inet6", []string{"[::1]:5432"}, true},
		{"socket:inet", []string{"bind:0.0.0.0:8080"}, true},
		{"socket:inet", []string{"unix:/run/x.sock"}, false},
		{"socket:inet", nil, false},
	}
	for _, c := range cases {
		if got := NetworkAllowed(c.entry, c.allowed); got != c.want {
			t.Errorf("NetworkAllowed(%q, %v) = %v, want %v", c.entry, c.allowed, got, c.want)
		}
	}
}
