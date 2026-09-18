package sandbox

import (
	"context"
	"errors"
	"testing"
)

func TestDefaultSyncOptions(t *testing.T) {
	opts := DefaultSyncOptions()
	if opts.User != "boite" {
		t.Fatalf("unexpected default user: %s", opts.User)
	}
}

func TestBuildSCPArgs(t *testing.T) {
	target := &Target{Name: "pingu", Host: "10.0.0.1", Port: 2226, KeyPath: "/key", User: "boite"}
	args := buildSCPArgs(target, "", "/local/file", "/remote/file")
	if len(args) < 5 {
		t.Fatalf("unexpected scp args: %v", args)
	}
	last := args[len(args)-1]
	if last != "boite@10.0.0.1:/remote/file" {
		t.Fatalf("unexpected dest in scp args: %s", last)
	}
}

func TestCopyFileToVM(t *testing.T) {
	if err := CopyFileToVM(context.Background(), nil, DefaultSyncOptions(), "a", "b"); err == nil {
		t.Fatal("expected error for nil target")
	}
	target := &Target{Name: "pingu", Backend: "boite", Port: 2226, KeyPath: "/key"}
	var commands []string
	runner := &mockRunner{
		runFunc: func(_ context.Context, name string, _ ...string) ([]byte, error) {
			commands = append(commands, name)
			return []byte("ok"), nil
		},
	}
	opts := SyncOptions{User: "boite", Runner: runner}
	if err := CopyFileToVM(context.Background(), target, opts, "/local/path", "/remote/path"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commands) != 1 || commands[0] != "scp" {
		t.Fatalf("expected scp command, got %v", commands)
	}
	runnerErr := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("fail"), errors.New("exit 1")
		},
	}
	opts.Runner = runnerErr
	if err := CopyFileToVM(context.Background(), target, opts, "/local/path", "/remote/path"); err == nil {
		t.Fatal("expected error when scp fails")
	}
}
