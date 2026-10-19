package claims

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Options are the verifier's decisions about evidence that tdd-ai will not
// accept on its own: anything outside the project, network activity, and
// other programs. Reads and Writes, when set, replace the proposed globs.
type Options struct {
	Reads      []string
	Writes     []string
	External   []string
	Network    []string
	Execs      []string
	SkipStdout bool
	// Run is the claim's command, recorded so a later edit to it fails the check.
	Run []string
	// StdoutPath is the project-relative file that will hold the expected output.
	StdoutPath string
	// PinsNow holds the current hashes of the pinned files. When set, distill
	// refuses if a pinned file changed after it was observed.
	PinsNow map[string]string
	Now     time.Time
}

// Distill turns accepted evidence into checks. The second result lists why it
// refused: the evidence contradicts a decision in o, or needs one that wasn't
// made. Checks are only meaningful when that list is empty.
func Distill(ev *Evidence, o Options) (*Checks, []string) {
	ch := &Checks{
		DistilledAt:  o.Now.UTC().Format(time.RFC3339),
		Run:          o.Run,
		Pins:         ev.Pins,
		Reads:        Generalize(ev.Reads),
		Writes:       Generalize(ev.Writes),
		External:     orEmpty(o.External),
		Network:      orEmpty(o.Network),
		ExitCode:     ev.ExitCode,
		Stdout:       o.StdoutPath,
		StdoutSHA256: ev.StdoutSHA256,
	}
	var refusals []string

	if o.Writes != nil {
		ch.Writes = o.Writes
		for _, w := range ev.Writes {
			if !MatchAny(ch.Writes, w) {
				refusals = append(refusals, fmt.Sprintf("observed write %s is not covered by --allow-write", w))
			}
		}
	}
	if o.Reads != nil {
		ch.Reads = o.Reads
		for _, r := range ev.Reads {
			if !MatchAny(ch.Reads, r) && !MatchAny(ch.Writes, r) {
				refusals = append(refusals, fmt.Sprintf("observed read %s is not covered by --allow-read", r))
			}
		}
	}

	if missing := notMatched(ev.ExternalReads, ch.External); len(missing) > 0 {
		refusals = append(refusals, fmt.Sprintf("read outside the project (%s): pass --allow-external for each to accept it", strings.Join(missing, ", ")))
	}
	if missing := notMatched(ev.ExternalWrites, ch.External); len(missing) > 0 {
		refusals = append(refusals, fmt.Sprintf("wrote outside the project (%s): pass --allow-external for each to accept it", strings.Join(missing, ", ")))
	}

	var net []string
	for _, n := range ev.Network {
		if !NetworkAllowed(n, ch.Network) {
			net = append(net, n)
		}
	}
	if len(net) > 0 {
		refusals = append(refusals, fmt.Sprintf("network activity observed (%s): pass --allow-net for each to accept it, or treat the claim as false", strings.Join(net, ", ")))
	}

	ch.Execs = []string{}
	if len(ev.Execs) > 0 {
		ch.Execs = append(ch.Execs, ev.Execs[0])
	}
	for _, e := range o.Execs {
		if !contains(ch.Execs, e) {
			ch.Execs = append(ch.Execs, e)
		}
	}
	if extra := notIn(ev.Execs, ch.Execs); len(extra) > 0 {
		refusals = append(refusals, fmt.Sprintf("ran other programs (%s): pass --allow-exec for each to accept them", strings.Join(extra, ", ")))
	}

	if o.PinsNow != nil {
		for _, p := range sortedKeys(ev.Pins) {
			if o.PinsNow[p] != ev.Pins[p] {
				refusals = append(refusals, fmt.Sprintf("pinned file %s changed after it was observed: verify the claim again", p))
			}
		}
	}

	if o.SkipStdout {
		ch.Stdout = ""
		ch.StdoutSHA256 = ""
	}
	return ch, refusals
}

// NetworkAllowed reports whether an observed network entry is allowed. Opening
// an internet socket ("socket:inet") is implied by allowing any address of
// that family.
func NetworkAllowed(entry string, allowed []string) bool {
	if contains(allowed, entry) {
		return true
	}
	switch entry {
	case "socket:inet":
		for _, a := range allowed {
			if isInet4(a) {
				return true
			}
		}
	case "socket:inet6":
		for _, a := range allowed {
			if strings.HasPrefix(strings.TrimPrefix(a, "bind:"), "[") {
				return true
			}
		}
	}
	return false
}

func isInet4(entry string) bool {
	entry = strings.TrimPrefix(entry, "bind:")
	host, _, ok := strings.Cut(entry, ":")
	return ok && host != "" && host[0] >= '0' && host[0] <= '9'
}

func notMatched(paths, patterns []string) []string {
	var out []string
	for _, p := range paths {
		if !MatchAny(patterns, p) {
			out = append(out, p)
		}
	}
	return out
}

func notIn(list, allowed []string) []string {
	var out []string
	for _, v := range list {
		if !contains(allowed, v) {
			out = append(out, v)
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
