// Package observe runs a command under a system-call tracer and reports what
// it actually touched: files it read and wrote, network destinations, and the
// programs it executed. Claims use it so tdd-ai records what it saw instead of
// trusting what an agent says happened.
package observe

import (
	"bufio"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Open is a file the traced process read, wrote or changed.
type Open struct {
	Path  string `json:"path"`  // absolute path
	Write bool   `json:"write"` // written, created, truncated, renamed or removed
	Dir   bool   `json:"dir"`   // opened as a directory: a listing, not a read of contents
}

// Trace is what a strace log says the process tree did.
type Trace struct {
	Opens []Open
	// Network holds destinations ("1.2.3.4:443", "[::1]:9", "unix:/run/x.sock"),
	// listening addresses ("bind:0.0.0.0:8080") and internet sockets opened
	// ("socket:inet", "socket:inet6"), in first-seen order.
	Network []string
	// Execs holds the absolute paths of programs that were executed successfully.
	Execs []string
}

// Syscalls is the strace filter used to observe a command.
var Syscalls = []string{
	"open", "openat", "creat",
	"unlink", "unlinkat", "rmdir", "rename", "renameat", "renameat2", "mkdir", "mkdirat",
	"execve", "execveat", "chdir", "fchdir", "clone", "clone3", "fork", "vfork",
	"socket", "connect", "bind", "sendto", "sendmsg",
}

var (
	lineRE    = regexp.MustCompile(`^(\d+)\s+(.*)$`)
	resumedRE = regexp.MustCompile(`^<\.\.\. ([a-z0-9_]+) resumed>(.*)$`)
	nameRE    = regexp.MustCompile(`^[a-z0-9_]+$`)
	inetRE    = regexp.MustCompile(`sa_family=AF_INET, sin_port=htons\((\d+)\), sin_addr=inet_addr\("([^"]+)"\)`)
	inet6RE   = regexp.MustCompile(`sa_family=AF_INET6, sin6_port=htons\((\d+)\),.*?inet_pton\(AF_INET6, "([^"]+)"`)
	unixRE    = regexp.MustCompile(`sa_family=AF_UNIX, sun_path=(@?)"((?:[^"\\]|\\.)*)"`)
)

// ParseTrace reads a log written by `strace -f -y -o FILE` and returns what the
// traced processes did. cwd is the directory the first process started in; it
// resolves relative paths that strace does not annotate.
func ParseTrace(r io.Reader, cwd string) (*Trace, error) {
	p := &parser{
		root:    cwd,
		cwd:     map[string]string{},
		pending: map[string]string{},
		seenNet: map[string]bool{},
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		p.line(sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return &p.trace, nil
}

type parser struct {
	root    string
	cwd     map[string]string // pid -> working directory
	pending map[string]string // pid -> call text cut off by "<unfinished ...>"
	seenNet map[string]bool
	trace   Trace
}

func (p *parser) line(s string) {
	m := lineRE.FindStringSubmatch(s)
	if m == nil {
		return
	}
	pid, rest := m[1], m[2]
	if strings.HasSuffix(rest, "<unfinished ...>") {
		p.pending[pid] = strings.TrimRight(strings.TrimSuffix(rest, "<unfinished ...>"), " ")
		return
	}
	if rm := resumedRE.FindStringSubmatch(rest); rm != nil {
		prefix, ok := p.pending[pid]
		if !ok {
			return
		}
		delete(p.pending, pid)
		rest = prefix + strings.TrimLeft(rm[2], " ")
	}
	p.call(pid, rest)
}

func (p *parser) cwdOf(pid string) string {
	if d, ok := p.cwd[pid]; ok {
		return d
	}
	return p.root
}

func (p *parser) call(pid, s string) {
	open := strings.IndexByte(s, '(')
	if open <= 0 || !nameRE.MatchString(s[:open]) {
		return
	}
	name := s[:open]
	argText, after, ok := splitCall(s[open+1:])
	if !ok {
		return
	}
	ret, retNote, ok := parseReturn(after)
	if !ok {
		return
	}
	args := splitArgs(argText)
	cwd := p.cwdOf(pid)

	switch name {
	case "openat":
		if ret >= 0 && len(args) >= 3 {
			p.addOpen(pathOr(retNote, resolve(base(args[0], cwd), unquote(args[1]))), args[2])
		}
	case "open":
		if ret >= 0 && len(args) >= 2 {
			p.addOpen(pathOr(retNote, resolve(cwd, unquote(args[0]))), args[1])
		}
	case "creat":
		if ret >= 0 && len(args) >= 1 {
			p.addWrite(pathOr(retNote, resolve(cwd, unquote(args[0]))))
		}
	case "unlink", "rmdir", "mkdir":
		if ret == 0 && len(args) >= 1 {
			p.addWrite(resolve(cwd, unquote(args[0])))
		}
	case "unlinkat", "mkdirat":
		if ret == 0 && len(args) >= 2 {
			p.addWrite(resolve(base(args[0], cwd), unquote(args[1])))
		}
	case "rename":
		if ret == 0 && len(args) >= 2 {
			p.addWrite(resolve(cwd, unquote(args[0])))
			p.addWrite(resolve(cwd, unquote(args[1])))
		}
	case "renameat", "renameat2":
		if ret == 0 && len(args) >= 4 {
			p.addWrite(resolve(base(args[0], cwd), unquote(args[1])))
			p.addWrite(resolve(base(args[2], cwd), unquote(args[3])))
		}
	case "execve":
		if ret == 0 && len(args) >= 1 {
			p.trace.Execs = append(p.trace.Execs, resolve(cwd, unquote(args[0])))
		}
	case "execveat":
		if ret == 0 && len(args) >= 2 {
			p.trace.Execs = append(p.trace.Execs, resolve(base(args[0], cwd), unquote(args[1])))
		}
	case "chdir":
		if ret == 0 && len(args) >= 1 {
			p.cwd[pid] = resolve(cwd, unquote(args[0]))
		}
	case "fchdir":
		if ret == 0 && len(args) >= 1 {
			if d := note(args[0]); strings.HasPrefix(d, "/") {
				p.cwd[pid] = d
			}
		}
	case "clone", "clone3", "fork", "vfork":
		if ret > 0 {
			p.cwd[strconv.Itoa(ret)] = cwd
		}
	case "socket":
		if len(args) >= 1 {
			switch args[0] {
			case "AF_INET":
				p.addNet("socket:inet")
			case "AF_INET6":
				p.addNet("socket:inet6")
			}
		}
	case "connect", "sendto", "sendmsg":
		for _, dest := range destinations(argText, false) {
			p.addNet(dest)
		}
	case "bind":
		for _, dest := range destinations(argText, true) {
			p.addNet("bind:" + dest)
		}
	}
}

func (p *parser) addOpen(path, flags string) {
	o := Open{Path: path}
	for _, f := range strings.Split(flags, "|") {
		switch strings.TrimSpace(f) {
		case "O_WRONLY", "O_RDWR", "O_CREAT", "O_TRUNC", "O_APPEND":
			o.Write = true
		case "O_DIRECTORY":
			o.Dir = true
		}
	}
	p.trace.Opens = append(p.trace.Opens, o)
}

func (p *parser) addWrite(path string) {
	p.trace.Opens = append(p.trace.Opens, Open{Path: path, Write: true})
}

func (p *parser) addNet(entry string) {
	if !p.seenNet[entry] {
		p.seenNet[entry] = true
		p.trace.Network = append(p.trace.Network, entry)
	}
}

// destinations pulls socket addresses out of a call's arguments. With inetOnly,
// unix sockets are skipped (binding a unix socket is local IPC, not network).
func destinations(argText string, inetOnly bool) []string {
	var out []string
	for _, m := range inetRE.FindAllStringSubmatch(argText, -1) {
		out = append(out, m[2]+":"+m[1])
	}
	for _, m := range inet6RE.FindAllStringSubmatch(argText, -1) {
		out = append(out, "["+m[2]+"]:"+m[1])
	}
	if !inetOnly {
		for _, m := range unixRE.FindAllStringSubmatch(argText, -1) {
			out = append(out, "unix:"+m[1]+unescape(m[2]))
		}
	}
	return out
}

// splitCall finds the ')' that closes a call whose arguments start at s and
// returns the argument text and what follows (" = 3</path>").
func splitCall(s string) (args, after string, ok bool) {
	depth := 1
	inQuote, escaped, inNote := false, false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inQuote:
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inQuote = false
			}
		case inNote:
			if c == '>' {
				inNote = false
			}
		case c == '"':
			inQuote = true
		case c == '<' && i > 0 && isNoteOwner(s[i-1]):
			inNote = true
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth--
			if depth == 0 && c == ')' {
				return s[:i], s[i+1:], true
			}
		}
	}
	return "", "", false
}

// isNoteOwner reports whether a "<" after this byte starts a strace -y
// annotation: file descriptors ("3</path>") and AT_FDCWD ("AT_FDCWD</cwd>").
func isNoteOwner(b byte) bool {
	return (b >= '0' && b <= '9') || b == 'D'
}

// splitArgs splits a call's argument text on top-level commas.
func splitArgs(s string) []string {
	var out []string
	depth, start := 0, 0
	inQuote, escaped, inNote := false, false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inQuote:
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inQuote = false
			}
		case inNote:
			if c == '>' {
				inNote = false
			}
		case c == '"':
			inQuote = true
		case c == '<' && i > 0 && isNoteOwner(s[i-1]):
			inNote = true
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth--
		case c == ',' && depth == 0:
			out = append(out, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
	}
	if rest := strings.TrimSpace(s[start:]); rest != "" {
		out = append(out, rest)
	}
	return out
}

// parseReturn reads " = 3</path>" or " = -1 ENOENT (...)" into the return value
// and the annotation strace -y attached to it, if any.
func parseReturn(after string) (ret int, retNote string, ok bool) {
	after = strings.TrimSpace(after)
	if !strings.HasPrefix(after, "=") {
		return 0, "", false
	}
	after = strings.TrimSpace(after[1:])
	end := 0
	for end < len(after) && (after[end] == '-' || (after[end] >= '0' && after[end] <= '9')) {
		end++
	}
	n, err := strconv.Atoi(after[:end])
	if err != nil {
		return 0, "", false
	}
	if end < len(after) && after[end] == '<' {
		if last := strings.LastIndexByte(after, '>'); last > end {
			retNote = unescape(after[end+1 : last])
		}
	}
	return n, retNote, true
}

// note returns the annotation on an argument such as "3</tmp/x>" or "AT_FDCWD</tmp>".
// strace escapes non-ASCII bytes in annotations the same way it does in strings.
func note(arg string) string {
	i := strings.IndexByte(arg, '<')
	j := strings.LastIndexByte(arg, '>')
	if i < 0 || j <= i {
		return ""
	}
	return unescape(arg[i+1 : j])
}

// base is the directory a *at() call resolves against.
func base(dirfd, cwd string) string {
	if d := note(dirfd); strings.HasPrefix(d, "/") {
		return d
	}
	return cwd
}

func pathOr(retNote, fallback string) string {
	if strings.HasPrefix(retNote, "/") {
		return retNote
	}
	return fallback
}

func resolve(dir, p string) string {
	if p == "" {
		return dir
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(dir, p)
}

// unquote turns a strace string argument ("a\tb"...) into its value.
func unquote(arg string) string {
	arg = strings.TrimSuffix(arg, "...")
	if len(arg) < 2 || arg[0] != '"' || arg[len(arg)-1] != '"' {
		return arg
	}
	return unescape(arg[1 : len(arg)-1])
}

// unescape decodes the C-style escapes strace uses in strings.
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			b.WriteByte(c)
			continue
		}
		i++
		switch e := s[i]; e {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case 'v':
			b.WriteByte('\v')
		case 'f':
			b.WriteByte('\f')
		case 'x':
			if i+2 < len(s) {
				if v, err := strconv.ParseUint(s[i+1:i+3], 16, 8); err == nil {
					b.WriteByte(byte(v))
					i += 2
					continue
				}
			}
			b.WriteByte(e)
		default:
			if e >= '0' && e <= '7' {
				j := i
				for j < len(s) && j < i+3 && s[j] >= '0' && s[j] <= '7' {
					j++
				}
				v, _ := strconv.ParseUint(s[i:j], 8, 8)
				b.WriteByte(byte(v))
				i = j - 1
				continue
			}
			b.WriteByte(e)
		}
	}
	return b.String()
}
