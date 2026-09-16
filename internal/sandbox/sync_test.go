package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSyncOptions(t *testing.T) {
	opts := DefaultSyncOptions()
	if opts.RemotePath != "/tmp/kori" || opts.User != "boite" {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
}

func TestFindLocalBinary(t *testing.T) {
	bin, err := FindLocalBinary()
	if err != nil {
		t.Logf("no local binary found: %v", err)
	} else if bin == "" {
		t.Fatal("expected non-empty binary path")
	}
}

func TestCheckRemoteBinary(t *testing.T) {
	inst := &InstanceState{Name: "pingu", SSHPort: 2226, KeyPath: "/key"}
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("/usr/local/bin/kori\n"), nil
		},
	}
	opts := SyncOptions{User: "boite", Runner: runner}
	path, err := CheckRemoteBinary(context.Background(), inst, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/usr/local/bin/kori" {
		t.Fatalf("expected /usr/local/bin/kori, got %s", path)
	}
}

func TestCopyBinaryToVM(t *testing.T) {
	inst := &InstanceState{Name: "pingu", SSHPort: 2226, KeyPath: "/key"}
	var commands []string
	runner := &mockRunner{
		runFunc: func(_ context.Context, name string, _ ...string) ([]byte, error) {
			commands = append(commands, name)
			return []byte("ok"), nil
		},
	}
	opts := SyncOptions{User: "boite", RemotePath: "/tmp/kori", Runner: runner}
	err := CopyBinaryToVM(context.Background(), inst, opts, "/tmp/kori")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commands) != 2 || commands[0] != "scp" || commands[1] != "ssh" {
		t.Fatalf("expected scp and ssh, got: %v", commands)
	}
}

func TestEnsureKoriBinary(t *testing.T) {
	inst := &InstanceState{Name: "pingu", SSHPort: 2226, KeyPath: "/key"}
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("/usr/local/bin/kori\n"), nil
		},
	}
	opts := SyncOptions{User: "boite", Runner: runner}
	res, err := EnsureKoriBinary(context.Background(), inst, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "/usr/local/bin/kori" {
		t.Fatalf("expected existing binary, got: %s", res)
	}
}

func TestEnsureKoriBinaryCopy(t *testing.T) {
	tmpDir := t.TempDir()
	dummyBin := filepath.Join(tmpDir, "kori")
	if err := os.WriteFile(dummyBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}
	inst := &InstanceState{Name: "pingu", SSHPort: 2226, KeyPath: "/key"}
	callCount := 0
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			callCount++
			if callCount == 1 {
				return nil, errors.New("not found")
			}
			return []byte("ok"), nil
		},
	}
	opts := SyncOptions{LocalPath: dummyBin, RemotePath: "/tmp/kori", Runner: runner}
	res, err := EnsureKoriBinary(context.Background(), inst, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "/tmp/kori" {
		t.Fatalf("expected /tmp/kori, got %s", res)
	}
}
