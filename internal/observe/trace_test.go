package observe

import (
	"reflect"
	"strings"
	"testing"
)

func parse(t *testing.T, log string) *Trace {
	t.Helper()
	tr, err := ParseTrace(strings.NewReader(log), "/tmp/p")
	if err != nil {
		t.Fatalf("ParseTrace: %v", err)
	}
	return tr
}

func TestParseTraceRecordsSuccessfulOpensWithResolvedPaths(t *testing.T) {
	tr := parse(t, `
2559  execve("/opt/node22/bin/node", ["node", "probe.mjs", "invoices/march.csv"], 0x7ffee0585bb8 /* 210 vars */) = 0
2559  openat(AT_FDCWD</tmp/p>, "invoices/march.csv", O_RDONLY|O_CLOEXEC) = 20</tmp/p/invoices/march.csv>
2559  openat(AT_FDCWD</tmp/p>, "missing.txt", O_RDONLY) = -1 ENOENT (No such file or directory)
2559  openat(AT_FDCWD</tmp/p>, "out/report.json", O_WRONLY|O_CREAT|O_TRUNC|O_CLOEXEC, 0666) = 21</tmp/p/out/report.json>
2559  openat(AT_FDCWD</tmp/p>, "invoices", O_RDONLY|O_NONBLOCK|O_CLOEXEC|O_DIRECTORY) = 22</tmp/p/invoices>
2559  open("notes.txt", O_RDWR) = 4
`)
	want := []Open{
		{Path: "/tmp/p/invoices/march.csv"},
		{Path: "/tmp/p/out/report.json", Write: true},
		{Path: "/tmp/p/invoices", Dir: true},
		{Path: "/tmp/p/notes.txt", Write: true},
	}
	if !reflect.DeepEqual(tr.Opens, want) {
		t.Errorf("Opens =\n%+v\nwant\n%+v", tr.Opens, want)
	}
	if !reflect.DeepEqual(tr.Execs, []string{"/opt/node22/bin/node"}) {
		t.Errorf("Execs = %v", tr.Execs)
	}
}

func TestParseTraceJoinsUnfinishedAndResumedCalls(t *testing.T) {
	tr := parse(t, `
3001  openat(AT_FDCWD</tmp/p>, "a.txt", O_RDONLY|O_CLOEXEC <unfinished ...>
3002  connect(7<socket:[901]>, {sa_family=AF_INET, sin_port=htons(443), sin_addr=inet_addr("93.184.216.34")}, 16 <unfinished ...>
3001  <... openat resumed>) = 5</tmp/p/a.txt>
3002  <... connect resumed>) = -1 EINPROGRESS (Operation now in progress)
`)
	if !reflect.DeepEqual(tr.Opens, []Open{{Path: "/tmp/p/a.txt"}}) {
		t.Errorf("Opens = %+v", tr.Opens)
	}
	if !reflect.DeepEqual(tr.Network, []string{"93.184.216.34:443"}) {
		t.Errorf("Network = %v", tr.Network)
	}
}

func TestParseTraceRecordsNetworkAttemptsEvenWhenTheyFail(t *testing.T) {
	tr := parse(t, `
3681  socket(AF_INET, SOCK_STREAM|SOCK_CLOEXEC, IPPROTO_IP) = 3<socket:[5445]>
3681  connect(3<socket:[5445]>, {sa_family=AF_INET, sin_port=htons(9), sin_addr=inet_addr("127.0.0.1")}, 16) = -1 ECONNREFUSED (Connection refused)
3681  socket(AF_INET6, SOCK_STREAM|SOCK_CLOEXEC, IPPROTO_IP) = -1 EAFNOSUPPORT (Address family not supported by protocol)
3681  connect(4<socket:[5446]>, {sa_family=AF_INET6, sin6_port=htons(443), sin6_flowinfo=htonl(0), inet_pton(AF_INET6, "2606:2800:220:1::248", &sin6_addr), sin6_scope_id=0}, 28) = 0
3681  connect(5<socket:[5447]>, {sa_family=AF_UNIX, sun_path="/var/run/docker.sock"}, 110) = 0
3681  sendto(6<socket:[5448]>, "\x12\x34"..., 29, MSG_NOSIGNAL, {sa_family=AF_INET, sin_port=htons(53), sin_addr=inet_addr("8.8.8.8")}, 16) = 29
3681  sendmsg(6<socket:[5448]>, {msg_name={sa_family=AF_INET, sin_port=htons(53), sin_addr=inet_addr("10.0.0.2")}, msg_namelen=16, msg_iov=[{iov_base="\x01", iov_len=1}], msg_iovlen=1, msg_controllen=0, msg_flags=0}, 0) = 1
3681  bind(7<socket:[5449]>, {sa_family=AF_INET, sin_port=htons(8080), sin_addr=inet_addr("0.0.0.0")}, 16) = 0
3681  socket(AF_NETLINK, SOCK_RAW|SOCK_CLOEXEC, NETLINK_ROUTE) = 8<NETLINK:[5450]>
3681  connect(3<socket:[5445]>, {sa_family=AF_INET, sin_port=htons(9), sin_addr=inet_addr("127.0.0.1")}, 16) = -1 ECONNREFUSED (Connection refused)
`)
	want := []string{
		"socket:inet",
		"127.0.0.1:9",
		"socket:inet6",
		"[2606:2800:220:1::248]:443",
		"unix:/var/run/docker.sock",
		"8.8.8.8:53",
		"10.0.0.2:53",
		"bind:0.0.0.0:8080",
	}
	if !reflect.DeepEqual(tr.Network, want) {
		t.Errorf("Network =\n%v\nwant\n%v", tr.Network, want)
	}
}

func TestParseTraceKeepsOnlySuccessfulExecs(t *testing.T) {
	tr := parse(t, `
4000  execve("/bin/sh", ["sh", "-c", "curl https://x, then quit"], 0x7ffd /* 3 vars */) = 0
4001  execve("/usr/local/bin/curl", ["curl", "https://x"], 0x7ffd /* 3 vars */) = -1 ENOENT (No such file or directory)
4001  execve("/usr/bin/curl", ["curl", "https://x"], 0x7ffd /* 3 vars */) = 0
`)
	if !reflect.DeepEqual(tr.Execs, []string{"/bin/sh", "/usr/bin/curl"}) {
		t.Errorf("Execs = %v", tr.Execs)
	}
}

func TestParseTraceResolvesPathCallsAgainstTrackedWorkingDirectory(t *testing.T) {
	tr := parse(t, `
5000  chdir("sub") = 0
5000  clone(child_stack=NULL, flags=CLONE_CHILD_CLEARTID|CLONE_CHILD_SETTID|SIGCHLD, child_tidptr=0x7f) = 5001
5001  rename("a.tmp", "a.json") = 0
5000  unlink("old.txt") = 0
5000  unlink("gone.txt") = -1 ENOENT (No such file or directory)
5000  mkdirat(AT_FDCWD</tmp/p/sub>, "cache", 0777) = 0
5000  unlinkat(AT_FDCWD</tmp/p/sub>, "cache/x", 0) = 0
5000  renameat2(AT_FDCWD</tmp/p/sub>, "x.part", AT_FDCWD</tmp/p/sub>, "x.bin", RENAME_NOREPLACE) = 0
5000  fchdir(3</tmp/p/other>) = 0
5000  mkdir("made", 0755) = 0
`)
	var got []string
	for _, o := range tr.Opens {
		if !o.Write {
			t.Errorf("%s should be a write", o.Path)
		}
		got = append(got, o.Path)
	}
	want := []string{
		"/tmp/p/sub/a.tmp", "/tmp/p/sub/a.json", "/tmp/p/sub/old.txt",
		"/tmp/p/sub/cache", "/tmp/p/sub/cache/x", "/tmp/p/sub/x.part", "/tmp/p/sub/x.bin",
		"/tmp/p/other/made",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("writes =\n%v\nwant\n%v", got, want)
	}
}

func TestParseTraceHandlesAnnotationsWithParenthesesAndEscapes(t *testing.T) {
	tr := parse(t, `
6000  openat(AT_FDCWD</tmp/My Folder (1)>, "a.txt", O_RDONLY) = 3</tmp/My Folder (1)/a.txt>
6000  openat(AT_FDCWD</tmp/p>, "caf\303\251 \"q\".txt", O_RDONLY) = -1 ENOENT (No such file or directory)
6000  openat(AT_FDCWD</tmp/p>, "caf\303\251 (1).txt", O_RDONLY) = 3</tmp/p/caf\303\251 (1).txt>
6000  open("tab\there.txt", O_WRONLY|O_CREAT, 0644) = 5
`)
	want := []Open{
		{Path: "/tmp/My Folder (1)/a.txt"},
		{Path: "/tmp/p/café (1).txt"},
		{Path: "/tmp/p/tab\there.txt", Write: true},
	}
	if !reflect.DeepEqual(tr.Opens, want) {
		t.Errorf("Opens = %+v, want %+v", tr.Opens, want)
	}
}

func TestParseTraceIgnoresNoise(t *testing.T) {
	tr := parse(t, `
strace: Process 5001 attached
5000  +++ exited with 0 +++
5000  --- SIGCHLD {si_signo=SIGCHLD, si_code=CLD_EXITED} ---
5001  <... openat resumed>) = 3</tmp/p/orphan>
not a trace line

5000  openat(AT_FDCWD</tmp/p>, "half", O_RDONLY
`)
	if len(tr.Opens) != 0 || len(tr.Network) != 0 || len(tr.Execs) != 0 {
		t.Errorf("expected nothing, got %+v", tr)
	}
}

func TestUnescape(t *testing.T) {
	cases := map[string]string{
		`plain`:          "plain",
		`a\nb\tc`:        "a\nb\tc",
		`caf\303\251`:    "café",
		`\x41\x42`:       "AB",
		`quote\"s\\`:     `quote"s\`,
		`trailing\`:      `trailing\`,
		`\0`:             "\x00",
		`mixed \1012 ok`: "mixed A2 ok",
	}
	for in, want := range cases {
		if got := unescape(in); got != want {
			t.Errorf("unescape(%q) = %q, want %q", in, got, want)
		}
	}
}
