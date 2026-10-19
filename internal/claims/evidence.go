package claims

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/macosta/tdd-ai/internal/observe"
)

// Locations decide where an observed path belongs.
type Locations struct {
	Project string // absolute project root
	Home    string // absolute home directory, "" if unknown
	Temp    string // absolute temp directory, "" if unknown
}

// DefaultLocations resolves the project, home and temp directories on this machine.
func DefaultLocations(project string) Locations {
	l := Locations{Project: realPath(project), Temp: realPath(os.TempDir())}
	if home, err := os.UserHomeDir(); err == nil {
		l.Home = realPath(home)
	}
	return l
}

func realPath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}

// systemPrefixes are read by the dynamic loader and language runtimes on every
// start. They say nothing about a claim, so evidence leaves them out.
// /etc/locale.alias is glibc's locale alias table: with LANG set, every
// program reads /usr/share/locale/locale.alias, which Debian and Ubuntu link
// there, and the trace records the link's target.
var systemPrefixes = []string{
	"/usr/", "/lib/", "/lib32/", "/lib64/", "/libx32/", "/bin/", "/sbin/",
	"/proc/", "/sys/", "/dev/",
	"/etc/ld.so.", "/etc/localtime", "/etc/locale.alias", "/etc/ssl/openssl.cnf",
}

type place int

const (
	placeIgnored place = iota
	placeProject
	placeOutside
)

func (l Locations) classify(abs string, execs map[string]bool) (place, string) {
	abs = filepath.Clean(abs)
	if rel, ok := within(l.Project, abs); ok {
		return placeProject, rel
	}
	if execs[abs] {
		return placeIgnored, ""
	}
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(abs, prefix) {
			return placeIgnored, ""
		}
	}
	if _, ok := within(l.Temp, abs); ok {
		return placeIgnored, ""
	}
	if rel, ok := within(l.Home, abs); ok {
		if rel == "." {
			return placeOutside, "~"
		}
		return placeOutside, "~/" + rel
	}
	return placeOutside, filepath.ToSlash(abs)
}

func within(root, p string) (string, bool) {
	if root == "" {
		return "", false
	}
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// FromResult turns one run into evidence. Without a trace (an untraced run)
// only the exit code, output and pins are recorded.
func FromResult(loc Locations, res *observe.Result, pins map[string]string, observer string, now time.Time) *Evidence {
	ev := &Evidence{
		ObservedAt:     now.UTC().Format(time.RFC3339),
		Observer:       observer,
		ExitCode:       res.ExitCode,
		StdoutSHA256:   HashBytes(res.Stdout),
		StdoutBytes:    len(res.Stdout),
		Reads:          []string{},
		Writes:         []string{},
		ExternalReads:  []string{},
		ExternalWrites: []string{},
		Network:        []string{},
		Execs:          []string{},
		Pins:           pins,
	}
	if res.Trace == nil {
		return ev
	}

	execs := map[string]bool{}
	for _, e := range res.Trace.Execs {
		e = filepath.Clean(e)
		execs[e] = true
		// A program can read its own image by its real path, so a symlinked
		// install (/usr/local/bin/node -> /opt/node22/bin/node) counts too.
		if resolved, err := filepath.EvalSymlinks(e); err == nil {
			execs[resolved] = true
		}
	}
	for _, o := range res.Trace.Opens {
		where, p := loc.classify(o.Path, execs)
		switch {
		case where == placeIgnored:
		case o.Dir:
			ev.Listed = append(ev.Listed, p)
		case where == placeProject && o.Write:
			ev.Writes = append(ev.Writes, p)
		case where == placeProject:
			ev.Reads = append(ev.Reads, p)
		case o.Write:
			ev.ExternalWrites = append(ev.ExternalWrites, p)
		default:
			ev.ExternalReads = append(ev.ExternalReads, p)
		}
	}
	ev.Reads = unique(ev.Reads)
	ev.Writes = unique(ev.Writes)
	ev.ExternalReads = unique(ev.ExternalReads)
	ev.ExternalWrites = unique(ev.ExternalWrites)
	if len(ev.Listed) > 0 {
		ev.Listed = unique(ev.Listed)
	}
	ev.Network = unique(res.Trace.Network)

	seen := map[string]bool{}
	for _, e := range res.Trace.Execs {
		name := filepath.Base(e)
		if !seen[name] {
			seen[name] = true
			ev.Execs = append(ev.Execs, name)
		}
	}
	return ev
}

// Notable lists what a verifier should look at before accepting the evidence.
func Notable(ev *Evidence) []string {
	var out []string
	if len(ev.Network) > 0 {
		out = append(out, "network activity: "+strings.Join(ev.Network, ", "))
	}
	if len(ev.ExternalReads) > 0 {
		out = append(out, "read outside the project: "+strings.Join(ev.ExternalReads, ", "))
	}
	if len(ev.ExternalWrites) > 0 {
		out = append(out, "wrote outside the project: "+strings.Join(ev.ExternalWrites, ", "))
	}
	var listedOutside []string
	for _, l := range ev.Listed {
		if strings.HasPrefix(l, "~") || strings.HasPrefix(l, "/") {
			listedOutside = append(listedOutside, l)
		}
	}
	if len(listedOutside) > 0 {
		out = append(out, "listed directories outside the project: "+strings.Join(listedOutside, ", "))
	}
	if len(ev.Writes) > 0 {
		out = append(out, "wrote in the project: "+strings.Join(ev.Writes, ", "))
	}
	if len(ev.Execs) > 1 {
		out = append(out, "ran other programs: "+strings.Join(ev.Execs[1:], ", "))
	}
	if ev.ExitCode != 0 {
		out = append(out, "exited with code "+itoa(ev.ExitCode))
	}
	return out
}

func unique(in []string) []string {
	set := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !set[s] {
			set[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
