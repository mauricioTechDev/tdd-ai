package claims

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/macosta/tdd-ai/internal/observe"
)

var testLoc = Locations{Project: "/work/proj", Home: "/home/u", Temp: "/tmp"}

func testResult() *observe.Result {
	return &observe.Result{
		ExitCode: 0,
		Stdout:   []byte("ok\n"),
		Trace: &observe.Trace{
			Opens: []observe.Open{
				{Path: "/work/proj/invoices/march.csv"},
				{Path: "/work/proj/invoices/march.csv"},
				{Path: "/work/proj/skills/invoice/reconcile.mjs"},
				{Path: "/work/proj/out/report.json", Write: true},
				{Path: "/work/proj/invoices", Dir: true},
				{Path: "/etc/ld.so.cache"},
				{Path: "/etc/locale.alias"},
				{Path: "/usr/lib/x86_64-linux-gnu/libc.so.6"},
				{Path: "/opt/node22/bin/node"},
				{Path: "/tmp/scratch-123", Write: true},
				{Path: "/home/u/.ssh/id_rsa"},
				{Path: "/home/u/.ssh", Dir: true},
				{Path: "/home/u/.bashrc", Write: true},
				{Path: "/etc/passwd"},
			},
			Network: []string{"socket:inet", "93.184.216.34:443"},
			Execs:   []string{"/opt/node22/bin/node", "/usr/bin/curl", "/opt/node22/bin/node"},
		},
	}
}

func TestFromResultClassifiesWhatTheCommandTouched(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	ev := FromResult(testLoc, testResult(), map[string]string{"SKILL.md": "abc"}, "strace", now)

	if ev.ObservedAt != "2026-10-19T12:00:00Z" || ev.Observer != "strace" {
		t.Errorf("ObservedAt/Observer = %q/%q", ev.ObservedAt, ev.Observer)
	}
	if ev.StdoutSHA256 != HashBytes([]byte("ok\n")) || ev.StdoutBytes != 3 {
		t.Errorf("stdout digest = %q (%d bytes)", ev.StdoutSHA256, ev.StdoutBytes)
	}
	check := func(name string, got, want []string) {
		t.Helper()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	check("Reads", ev.Reads, []string{"invoices/march.csv", "skills/invoice/reconcile.mjs"})
	check("Writes", ev.Writes, []string{"out/report.json"})
	check("ExternalReads", ev.ExternalReads, []string{"/etc/passwd", "~/.ssh/id_rsa"})
	check("ExternalWrites", ev.ExternalWrites, []string{"~/.bashrc"})
	check("Listed", ev.Listed, []string{"invoices", "~/.ssh"})
	check("Network", ev.Network, []string{"93.184.216.34:443", "socket:inet"})
	check("Execs", ev.Execs, []string{"node", "curl"})
	if ev.Pins["SKILL.md"] != "abc" {
		t.Errorf("Pins = %v", ev.Pins)
	}
}

func TestFromResultIgnoresTheRealPathOfASymlinkedProgram(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "opt", "node22", "bin", "node")
	link := filepath.Join(dir, "bin", "node")
	for _, d := range []string{filepath.Dir(target), filepath.Dir(link)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	res := &observe.Result{Trace: &observe.Trace{
		Opens: []observe.Open{{Path: target}},
		Execs: []string{link},
	}}
	// Temp stays unset so the test directory is not ignored as scratch space.
	loc := Locations{Project: "/work/proj", Home: "/home/u"}
	ev := FromResult(loc, res, nil, "strace", time.Now())
	if len(ev.ExternalReads) != 0 {
		t.Errorf("ExternalReads = %v, want none", ev.ExternalReads)
	}
	if !reflect.DeepEqual(ev.Execs, []string{"node"}) {
		t.Errorf("Execs = %v, want [node]", ev.Execs)
	}
}

func TestFromResultWithoutTraceKeepsOnlyExitOutputAndPins(t *testing.T) {
	res := &observe.Result{ExitCode: 2, Stdout: []byte("x")}
	ev := FromResult(testLoc, res, nil, "none", time.Now())
	if ev.ExitCode != 2 || ev.StdoutBytes != 1 {
		t.Errorf("exit/stdout = %d/%d", ev.ExitCode, ev.StdoutBytes)
	}
	if len(ev.Reads)+len(ev.Writes)+len(ev.Network)+len(ev.Execs) != 0 {
		t.Errorf("untraced evidence should have no policy data: %+v", ev)
	}
}

func TestNotableFlagsWhatNeedsAVerdict(t *testing.T) {
	ev := FromResult(testLoc, testResult(), nil, "strace", time.Now())
	ev.ExitCode = 1
	got := Notable(ev)
	want := []string{
		"network activity: 93.184.216.34:443, socket:inet",
		"read outside the project: /etc/passwd, ~/.ssh/id_rsa",
		"wrote outside the project: ~/.bashrc",
		"listed directories outside the project: ~/.ssh",
		"wrote in the project: out/report.json",
		"ran other programs: curl",
		"exited with code 1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Notable =\n%v\nwant\n%v", got, want)
	}
}

func TestNotableIsEmptyForAQuietRun(t *testing.T) {
	ev := &Evidence{Execs: []string{"node"}, Reads: []string{"invoices/march.csv"}}
	if got := Notable(ev); len(got) != 0 {
		t.Errorf("Notable = %v, want nothing", got)
	}
}
