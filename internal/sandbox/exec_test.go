package sandbox

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestDefaultExecOptions(t *testing.T) {
	opts := DefaultExecOptions()
	if opts.User != "boite" || opts.WorkDir != "/workspace" {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
}

func TestFormatRemoteCommand(t *testing.T) {
	cmd1 := FormatRemoteCommand("kori", "/workspace", []string{"-bash"})
	if cmd1 != "kori -root /workspace -bash" {
		t.Fatalf("unexpected formatted cmd: %s", cmd1)
	}
	cmd2 := FormatRemoteCommand("", "/workspace", []string{"--root", "/custom", "-bash"})
	if strings.Count(cmd2, "root") != 1 {
		t.Fatalf("unexpected duplicate root flag: %s", cmd2)
	}
	cmd3 := FormatRemoteCommand("kori", "", []string{"hello world"})
	if cmd3 != "kori \"hello world\"" {
		t.Fatalf("unexpected quoted output: %s", cmd3)
	}
}

func TestBuildSSHArgs(t *testing.T) {
	inst := &InstanceState{Name: "pingu", SSHPort: 2226, KeyPath: "/key"}
	args := BuildSSHArgs(inst, "boite", "kori -bash")
	if len(args) < 7 || args[0] != "-tt" || args[len(args)-1] != "kori -bash" {
		t.Fatalf("unexpected ssh args: %v", args)
	}
}

func TestBuildSSHCommand(t *testing.T) {
	if _, err := BuildSSHCommand(context.Background(), nil, DefaultExecOptions()); err == nil {
		t.Fatal("expected error for nil instance")
	}
	inst := &InstanceState{Name: "pingu", SSHPort: 2226, KeyPath: "/key"}
	opts := DefaultExecOptions()
	opts.Args = []string{"-bash"}
	cmd, err := BuildSSHCommand(context.Background(), inst, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Path != "ssh" && !strings.HasSuffix(cmd.Path, "ssh") {
		t.Fatalf("expected ssh binary, got %s", cmd.Path)
	}
}

func TestRunSSH(t *testing.T) {
	inst := &InstanceState{Name: "pingu", SSHPort: 2226, KeyPath: "/key"}
	called := false
	runner := &mockRunner{
		execFunc: func(_ context.Context, cmd *exec.Cmd) error {
			called = true
			return nil
		},
	}
	opts := DefaultExecOptions()
	opts.Runner = runner
	if err := RunSSH(context.Background(), inst, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected runner.RunInteractive to be called")
	}
}
