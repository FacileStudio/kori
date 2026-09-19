package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRemoteToolsConstruct(t *testing.T) {
	opts := ToolsOptions{
		Target: &Target{Name: "vm1", Host: "127.0.0.1", Port: 2226, User: "boite"},
		Runner: &mockRunner{},
	}
	tools, closer, err := RemoteTools(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if closer == nil {
		t.Fatal("expected non-nil closer")
	}
	if err := closer.Close(); err != nil {
		t.Fatalf("closer.Close failed: %v", err)
	}
	if len(tools) != 7 {
		t.Fatalf("expected 7 tools, got %d", len(tools))
	}
	expected := map[string]bool{
		"run_command": true, "read_file": true, "write_file": true,
		"edit_file": true, "list_directory": true, "find_files": true,
		"search_content": true,
	}
	for _, tool := range tools {
		delete(expected, tool.Name())
	}
	if len(expected) > 0 {
		t.Fatalf("missing tools: %v", expected)
	}
}

func TestRemoteToolsSSHHardeningFlags(t *testing.T) {
	var capturedArgs []string
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			capturedArgs = args
			return []byte("ok"), nil
		},
	}
	opts := ToolsOptions{
		Target: &Target{Name: "vm1", Backend: "boite", Host: "127.0.0.1", Port: 2226, User: "boite"},
		Runner: runner,
	}
	tools, _, err := RemoteTools(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = tools[0].Run(context.Background(), json.RawMessage(`{"command":"id"}`))
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	argStr := strings.Join(capturedArgs, " ")
	for _, flag := range []string{"BatchMode=yes", "ForwardAgent=no", "StrictHostKeyChecking=no", "UserKnownHostsFile=/dev/null"} {
		if !strings.Contains(argStr, flag) {
			t.Fatalf("missing flag %s in args: %s", flag, argStr)
		}
	}
	for _, arg := range capturedArgs {
		if arg == "-tt" {
			t.Fatal("interactive -tt flag must not be used")
		}
	}
}

func TestRunCommandSuccess(t *testing.T) {
	var remoteCmd string
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			remoteCmd = args[len(args)-1]
			return []byte("line 1\nline 2\n"), nil
		},
	}
	opts := ToolsOptions{WorkDir: "/custom", Runner: runner}
	tools, _, _ := RemoteTools(opts)
	out, err := tools[0].Run(context.Background(), json.RawMessage(`{"command":"pwd","timeout":10}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(remoteCmd, "cd /custom") || !strings.Contains(remoteCmd, "pwd") {
		t.Fatalf("unexpected remote command: %s", remoteCmd)
	}
	if out != "line 1\nline 2" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRunCommandErrors(t *testing.T) {
	opts := ToolsOptions{Runner: &mockRunner{}}
	tools, _, _ := RemoteTools(opts)
	if _, err := tools[0].Run(context.Background(), json.RawMessage(`{"command":""}`)); err == nil {
		t.Fatal("expected error for empty command")
	}
	failRunner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("error text"), errors.New("command failed")
		},
	}
	tools, _, _ = RemoteTools(ToolsOptions{Runner: failRunner})
	out, err := tools[0].Run(context.Background(), json.RawMessage(`{"command":"false"}`))
	if err != nil {
		t.Fatalf("expected nil error reported to model, got: %v", err)
	}
	if !strings.Contains(out, "command failed") {
		t.Fatalf("expected error text in report, got: %s", out)
	}
}

func TestRunCommandTimeout(t *testing.T) {
	sleepRunner := &mockRunner{
		runFunc: func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
			<-ctx.Done()
			return []byte("partial output"), ctx.Err()
		},
	}
	opts := ToolsOptions{CommandTimeout: 10 * time.Millisecond, Runner: sleepRunner}
	tools, _, _ := RemoteTools(opts)
	out, err := tools[0].Run(context.Background(), json.RawMessage(`{"command":"sleep 10"}`))
	if err != nil {
		t.Fatalf("unexpected tool error: %v", err)
	}
	if !strings.Contains(out, "timed out after") {
		t.Fatalf("expected timeout notice, got: %s", out)
	}
}

func TestReadFileSuccess(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if !strings.Contains(args[len(args)-1], "cat -- /workspace/main.go") {
				t.Fatalf("unexpected read command: %v", args)
			}
			return []byte("first\nsecond\nthird\nfourth\n"), nil
		},
	}
	tools, _, _ := RemoteTools(ToolsOptions{WorkDir: "/workspace", Runner: runner})
	readTool := tools[1]
	out, err := readTool.Run(context.Background(), json.RawMessage(`{"path":"main.go","offset":2,"limit":2}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "2\tsecond") || !strings.Contains(out, "3\tthird") {
		t.Fatalf("unexpected numbered output: %s", out)
	}
}

func TestReadFileErrors(t *testing.T) {
	failRunner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("file not found")
		},
	}
	tools, _, _ := RemoteTools(ToolsOptions{Runner: failRunner})
	if _, err := tools[1].Run(context.Background(), json.RawMessage(`{"path":"missing.txt"}`)); err == nil {
		t.Fatal("expected error for missing file")
	}
	binRunner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte{0x00, 0x01, 0x02, 0xff}, nil
		},
	}
	tools, _, _ = RemoteTools(ToolsOptions{Runner: binRunner})
	if _, err := tools[1].Run(context.Background(), json.RawMessage(`{"path":"bin"}`)); err == nil {
		t.Fatal("expected error for binary file")
	}
}

func TestWriteFileSuccess(t *testing.T) {
	var remoteCmd string
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			remoteCmd = args[len(args)-1]
			return nil, nil
		},
	}
	tools, _, _ := RemoteTools(ToolsOptions{WorkDir: "/workspace", Runner: runner})
	writeTool := tools[2]
	out, err := writeTool.Run(context.Background(), json.RawMessage(`{"path":"pkg/app.go","content":"package main\n"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "wrote pkg/app.go (13 bytes)") {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(remoteCmd, "mkdir -p /workspace/pkg") || !strings.Contains(remoteCmd, "base64 -d") {
		t.Fatalf("unexpected remote command: %s", remoteCmd)
	}
}
