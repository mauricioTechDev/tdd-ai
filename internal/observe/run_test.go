package observe

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const helperEnv = "TDD_AI_OBSERVE_HELPER"

// TestHelperProcess is not a real test: Run and RunPlain re-execute the test
// binary with helperEnv set, so it acts as a small program with known behavior.
func TestHelperProcess(t *testing.T) {
	if os.Getenv(helperEnv) != "1" {
		return
	}
	if _, err := os.ReadFile("in.txt"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := os.WriteFile("out.txt", []byte("written"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if conn, err := net.DialTimeout("tcp", "127.0.0.1:9", 200*time.Millisecond); err == nil {
		conn.Close()
	}
	fmt.Println("hello")
	os.Exit(3)
}

func helperCommand(t *testing.T) (dir string, argv []string) {
	t.Helper()
	dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "in.txt"), []byte("input"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(helperEnv, "1")
	return dir, []string{os.Args[0], "-test.run=^TestHelperProcess$"}
}

func TestRunObservesReadsWritesNetworkAndExit(t *testing.T) {
	if !Available() {
		t.Skip("strace is not available")
	}
	dir, argv := helperCommand(t)
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Run(context.Background(), dir, argv)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3 (stderr: %s)", res.ExitCode, res.Stderr)
	}
	if string(res.Stdout) != "hello\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "hello\n")
	}
	if res.Trace == nil {
		t.Fatal("Trace is nil")
	}

	var sawRead, sawWrite bool
	for _, o := range res.Trace.Opens {
		switch o.Path {
		case filepath.Join(dir, "in.txt"):
			sawRead = sawRead || !o.Write
		case filepath.Join(dir, "out.txt"):
			sawWrite = sawWrite || o.Write
		}
	}
	if !sawRead {
		t.Errorf("read of in.txt not observed in %+v", res.Trace.Opens)
	}
	if !sawWrite {
		t.Errorf("write of out.txt not observed in %+v", res.Trace.Opens)
	}
	if !contains(res.Trace.Network, "127.0.0.1:9") {
		t.Errorf("connect to 127.0.0.1:9 not observed in %v", res.Trace.Network)
	}
	if len(res.Trace.Execs) == 0 || filepath.Base(res.Trace.Execs[0]) != filepath.Base(argv[0]) {
		t.Errorf("Execs = %v, want the helper binary first", res.Trace.Execs)
	}
}

func TestRunPlainReportsOutputAndExitWithoutTrace(t *testing.T) {
	dir, argv := helperCommand(t)
	res, err := RunPlain(context.Background(), dir, argv)
	if err != nil {
		t.Fatalf("RunPlain: %v", err)
	}
	if res.ExitCode != 3 || string(res.Stdout) != "hello\n" || res.Trace != nil {
		t.Errorf("got exit %d, stdout %q, trace %v", res.ExitCode, res.Stdout, res.Trace)
	}
}

func TestRunRejectsEmptyCommand(t *testing.T) {
	if _, err := Run(context.Background(), t.TempDir(), nil); err == nil {
		t.Error("Run with no command should fail")
	}
	if _, err := RunPlain(context.Background(), t.TempDir(), nil); err == nil {
		t.Error("RunPlain with no command should fail")
	}
}

func TestRunPlainFailsForMissingProgram(t *testing.T) {
	if _, err := RunPlain(context.Background(), t.TempDir(), []string{"tdd-ai-no-such-program"}); err == nil {
		t.Error("expected an error for a program that does not exist")
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
