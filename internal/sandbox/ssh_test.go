package sandbox

import (
	"strings"
	"testing"
)

func TestBuildRemoteSSHArgs_BoiteRelaxesHostKeys(t *testing.T) {
	target := &Target{Name: "vm", Backend: "boite", Host: "127.0.0.1", Port: 2226, User: "boite", KeyPath: "/key"}
	args := strings.Join(buildRemoteSSHArgs(target, "boite", "whoami", ""), " ")
	for _, flag := range []string{"BatchMode=yes", "ForwardAgent=no", "StrictHostKeyChecking=no", "UserKnownHostsFile=/dev/null"} {
		if !strings.Contains(args, flag) {
			t.Fatalf("missing %s in %s", flag, args)
		}
	}
	if !strings.Contains(args, "-p 2226") || !strings.Contains(args, "-i /key") || !strings.Contains(args, "-- boite@127.0.0.1") {
		t.Fatalf("unexpected args: %s", args)
	}
}

func TestBuildRemoteSSHArgs_RemoteVerifiesHostKeys(t *testing.T) {
	target := &Target{Name: "host", Backend: "ssh", Host: "build.example.com", Port: 22, User: "deploy"}
	args := strings.Join(buildRemoteSSHArgs(target, "deploy", "id", "/tmp/sock"), " ")
	for _, forbidden := range []string{"StrictHostKeyChecking=no", "UserKnownHostsFile=/dev/null"} {
		if strings.Contains(args, forbidden) {
			t.Fatalf("a remote host must verify host keys, found %s in %s", forbidden, args)
		}
	}
	for _, want := range []string{"ForwardAgent=no", "ControlMaster=auto", "ControlPath=/tmp/sock", "ControlPersist=60s"} {
		if !strings.Contains(args, want) {
			t.Fatalf("missing %s in %s", want, args)
		}
	}
}

// An SSH host with no configured port must not get a -p: an explicit -p 22
// overrides Port from ~/.ssh/config, sending an alias there to the wrong port.
func TestBuildRemoteSSHArgs_OmitsPortWhenUnset(t *testing.T) {
	target := &Target{Name: "alias", Backend: "ssh", Host: "myserver"}
	args := strings.Join(buildRemoteSSHArgs(target, "", "id", ""), " ")
	if strings.Contains(args, " -p ") {
		t.Fatalf("expected no -p so ssh resolves the port, got %s", args)
	}
	if !strings.HasSuffix(args, " id") {
		t.Fatalf("expected the command last: %s", args)
	}
}

func TestWorkdirPrefix(t *testing.T) {
	if got := workdirPrefix(""); got != "" {
		t.Fatalf("expected no prefix for an empty workdir, got %q", got)
	}
	if got := workdirPrefix("/srv/app"); got != "cd /srv/app && " {
		t.Fatalf("unexpected prefix: %q", got)
	}
	if got := workdirPrefix("/path with space"); got != `cd '/path with space' && ` {
		t.Fatalf("unexpected quoted prefix: %q", got)
	}
}

func TestQuoteArg(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/srv/app", "/srv/app"},
		{"", "''"},
		{"/path with space", "'/path with space'"},
		{"$(curl evil)", "'$(curl evil)'"},
		{"`id`", "'`id`'"},
		{"it's", `'it'\''s'`},
		{"~", `"$HOME"`},
		{"~/notes", `"$HOME"/notes`},
		{"~/my notes", `"$HOME"/'my notes'`},
	}
	for _, c := range cases {
		if got := quoteArg(c.in); got != c.want {
			t.Fatalf("quoteArg(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestResolveRemotePath(t *testing.T) {
	cases := []struct{ workDir, p, want string }{
		{"/srv/app", "main.go", "/srv/app/main.go"},
		{"/srv/app", "/etc/hosts", "/etc/hosts"},
		{"/srv/app", "~/notes", "~/notes"},
		{"", "main.go", "main.go"},
		{"", "", "."},
	}
	for _, c := range cases {
		if got := resolveRemotePath(c.workDir, c.p); got != c.want {
			t.Fatalf("resolveRemotePath(%q, %q) = %q, want %q", c.workDir, c.p, got, c.want)
		}
	}
}

func TestBuildRemoteSSHArgs_NoControlPath(t *testing.T) {
	target := &Target{Backend: "ssh", Host: "h", Port: 22}
	args := strings.Join(buildRemoteSSHArgs(target, "", "cmd", ""), " ")
	if strings.Contains(args, "ControlMaster") {
		t.Fatalf("expected no multiplexing without a socket: %s", args)
	}
	if !strings.HasSuffix(args, "cmd") {
		t.Fatalf("expected the command last: %s", args)
	}
}
