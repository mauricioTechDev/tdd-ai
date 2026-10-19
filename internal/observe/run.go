package observe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Result is one run of a command: what it printed, how it exited and, when it
// ran under the tracer, what it touched.
type Result struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Trace    *Trace // nil when the command ran without tracing
}

// Available reports whether commands can be traced on this machine. Tracing
// uses strace, so it needs Linux with strace on the PATH.
func Available() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	_, err := exec.LookPath("strace")
	return err == nil
}

// Run executes argv in dir under strace and returns what it observed.
func Run(ctx context.Context, dir string, argv []string) (*Result, error) {
	if len(argv) == 0 {
		return nil, errors.New("no command to run")
	}
	logFile, err := os.CreateTemp("", "tdd-ai-trace-*.log")
	if err != nil {
		return nil, fmt.Errorf("creating trace file: %w", err)
	}
	logPath := logFile.Name()
	_ = logFile.Close()
	defer func() { _ = os.Remove(logPath) }()

	args := []string{
		"-f", "-qq", "-y", "-s", "65535",
		"-e", "signal=none",
		"-e", "trace=" + strings.Join(Syscalls, ","),
		"-o", logPath,
		"--",
	}
	res, err := run(ctx, dir, "strace", append(args, argv...))
	if err != nil {
		return nil, err
	}

	f, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("reading trace: %w", err)
	}
	defer func() { _ = f.Close() }()
	trace, err := ParseTrace(f, dir)
	if err != nil {
		return nil, fmt.Errorf("parsing trace: %w", err)
	}
	if len(trace.Execs) == 0 {
		return nil, fmt.Errorf("strace did not record %q starting (is ptrace allowed here?): %s",
			argv[0], lastLine(res.Stderr))
	}
	res.Trace = trace
	return res, nil
}

// RunPlain executes argv in dir without tracing.
func RunPlain(ctx context.Context, dir string, argv []string) (*Result, error) {
	if len(argv) == 0 {
		return nil, errors.New("no command to run")
	}
	return run(ctx, dir, argv[0], argv[1:])
}

func run(ctx context.Context, dir, name string, args []string) (*Result, error) {
	var stdout, stderr bytes.Buffer
	c := exec.CommandContext(ctx, name, args...)
	c.Dir = dir
	c.Stdout = &stdout
	c.Stderr = &stderr
	c.WaitDelay = 5 * time.Second

	err := c.Run()
	res := &Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, fmt.Errorf("running %s: %w", name, ctxErr)
	}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		res.ExitCode = exitErr.ExitCode()
	default:
		return nil, fmt.Errorf("running %s: %w", name, err)
	}
	return res, nil
}

func lastLine(b []byte) string {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	return lines[len(lines)-1]
}
