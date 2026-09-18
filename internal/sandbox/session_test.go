package sandbox

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestInstance(t *testing.T, name string, status string) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("BOITE_INSTANCES_DIR", tmp)
	instDir := filepath.Join(tmp, name)
	if err := os.MkdirAll(instDir, 0o755); err != nil {
		t.Fatalf("failed to create instance dir: %v", err)
	}
	stateJSON := `{"name":"` + name + `","ssh_port":2226,"key_path":"/key","status":"` + status + `"}`
	if err := os.WriteFile(filepath.Join(instDir, "state.json"), []byte(stateJSON), 0o644); err != nil {
		t.Fatalf("failed to write state.json: %v", err)
	}
	return tmp
}

func TestRunSessionValidation(t *testing.T) {
	err := RunSession(context.Background(), SessionOptions{})
	if err == nil {
		t.Fatal("expected error for empty target")
	}
}

func TestRunSessionNotFound(t *testing.T) {
	t.Setenv("BOITE_INSTANCES_DIR", t.TempDir())
	err := RunSession(context.Background(), SessionOptions{VMName: "nonexistent"})
	if err == nil || !strings.Contains(err.Error(), "loading sandbox instance") {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunSessionSuccess(t *testing.T) {
	setupTestInstance(t, "pingu", "running")
	calledExec := false
	runner := &mockRunner{
		runFunc: func(_ context.Context, name string, _ ...string) ([]byte, error) {
			if name == "ssh" {
				return []byte("boite\n/home/boite\nVM_HARNESS=12345\n"), nil
			}
			return []byte("ok"), nil
		},
		execFunc: func(_ context.Context, _ *exec.Cmd) error {
			calledExec = true
			return nil
		},
	}
	opts := SessionOptions{
		VMName:      "pingu",
		PrintPrompt: "echo hello",
		Runner:      runner,
	}
	err := RunSession(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !calledExec {
		t.Fatal("expected SSH execution")
	}
}

func TestRunSessionWithSnapshot(t *testing.T) {
	setupTestInstance(t, "pingu", "running")
	var commands []string
	runner := &mockRunner{
		runFunc: func(_ context.Context, name string, _ ...string) ([]byte, error) {
			commands = append(commands, name)
			if name == "ssh" {
				return []byte("boite\n/home/boite\nVM_HARNESS=12345\n"), nil
			}
			return []byte("ok"), nil
		},
	}
	opts := SessionOptions{
		VMName:    "pingu",
		Snapshot:  true,
		SkipGuard: true,
		Runner:    runner,
	}
	err := RunSession(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	foundBoite := false
	for _, cmd := range commands {
		if cmd == "boite" {
			foundBoite = true
		}
	}
	if !foundBoite {
		t.Fatal("expected boite snapshot command to be executed")
	}
}

func TestRunSessionExecError(t *testing.T) {
	setupTestInstance(t, "pingu", "running")
	runner := &mockRunner{
		execFunc: func(_ context.Context, _ *exec.Cmd) error {
			return errors.New("ssh failed")
		},
	}
	opts := SessionOptions{
		VMName:    "pingu",
		SkipGuard: true,
		Runner:    runner,
	}
	err := RunSession(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "ssh failed") {
		t.Fatalf("expected exec error, got %v", err)
	}
}
