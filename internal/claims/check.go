package claims

import (
	"bytes"
	"fmt"
	"strings"
)

// Failure is one check that did not hold.
type Failure struct {
	Check  string `json:"check"`
	Detail string `json:"detail"`
}

// Observation is a fresh run of a claim's command, ready to compare.
type Observation struct {
	// Run is the claim's command as it stands now.
	Run      []string
	Evidence *Evidence
	// Traced is false when the run skipped the tracer; the read, write,
	// network and program checks are then skipped too.
	Traced  bool
	PinsNow map[string]string
	Stdout  []byte
	// Expected holds the expected output file; nil when it is missing.
	Expected []byte
}

// Outcome is the result of checking one claim.
type Outcome struct {
	Pass          bool      `json:"pass"`
	PolicyChecked bool      `json:"policy_checked"`
	Failures      []Failure `json:"failures"`
}

// Evaluate compares a fresh run against a claim's checks. It needs no model:
// every comparison is a hash, a glob or an exact match.
func Evaluate(ch *Checks, obs Observation) Outcome {
	var fails []Failure
	add := func(check, format string, args ...any) {
		fails = append(fails, Failure{Check: check, Detail: fmt.Sprintf(format, args...)})
	}

	if len(ch.Run) > 0 && strings.Join(obs.Run, "\x00") != strings.Join(ch.Run, "\x00") {
		add("command", "changed since the claim was verified: now %q, verified %q", strings.Join(obs.Run, " "), strings.Join(ch.Run, " "))
	}
	for _, p := range sortedKeys(ch.Pins) {
		switch got := obs.PinsNow[p]; {
		case got == "":
			add("pin", "%s is missing", p)
		case got != ch.Pins[p]:
			add("pin", "%s changed since the claim was verified", p)
		}
	}

	ev := obs.Evidence
	if ev.ExitCode != ch.ExitCode {
		add("exit", "exited with %d, want %d", ev.ExitCode, ch.ExitCode)
	}

	if ch.Stdout != "" {
		switch {
		case obs.Expected == nil:
			add("stdout", "expected output %s is missing", ch.Stdout)
		case HashBytes(obs.Expected) != ch.StdoutSHA256:
			add("stdout", "expected output %s was edited after the claim was verified", ch.Stdout)
		case !bytes.Equal(obs.Stdout, obs.Expected):
			add("stdout", "%s", firstDifference(obs.Stdout, obs.Expected))
		}
	}

	if obs.Traced {
		allowedReads := append(append([]string{}, ch.Reads...), ch.Writes...)
		for _, r := range ev.Reads {
			if !MatchAny(allowedReads, r) {
				add("read", "%s is outside the allowed reads (%s)", r, listOrNone(allowedReads))
			}
		}
		for _, w := range ev.Writes {
			if !MatchAny(ch.Writes, w) {
				add("write", "%s is outside the allowed writes (%s)", w, listOrNone(ch.Writes))
			}
		}
		for _, e := range ev.ExternalReads {
			if !MatchAny(ch.External, e) {
				add("outside", "read %s outside the project", e)
			}
		}
		for _, e := range ev.ExternalWrites {
			if !MatchAny(ch.External, e) {
				add("outside", "wrote %s outside the project", e)
			}
		}
		for _, n := range ev.Network {
			if !NetworkAllowed(n, ch.Network) {
				add("network", "%s (allowed: %s)", n, listOrNone(ch.Network))
			}
		}
		for _, x := range ev.Execs {
			if !contains(ch.Execs, x) {
				add("program", "ran %s (allowed: %s)", x, listOrNone(ch.Execs))
			}
		}
	}

	if fails == nil {
		fails = []Failure{}
	}
	return Outcome{Pass: len(fails) == 0, PolicyChecked: obs.Traced, Failures: fails}
}

func firstDifference(got, want []byte) string {
	g, w := lines(got), lines(want)
	for i := 0; i < len(g) && i < len(w); i++ {
		if g[i] != w[i] {
			return fmt.Sprintf("line %d differs: got %q, want %q", i+1, clip(g[i]), clip(w[i]))
		}
	}
	if len(g) != len(w) {
		return fmt.Sprintf("output has %d lines, want %d", len(g), len(w))
	}
	return "output differs only in its final newline"
}

func lines(b []byte) []string {
	s := strings.TrimSuffix(string(b), "\n")
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}

func clip(s string) string {
	const max = 60
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func listOrNone(list []string) string {
	if len(list) == 0 {
		return "none"
	}
	return strings.Join(list, ", ")
}
