package sandbox

import (
	"context"
	"os/exec"
	"testing"
)

type mockRunner struct {
	runFunc  func(ctx context.Context, name string, args ...string) ([]byte, error)
	execFunc func(ctx context.Context, cmd *exec.Cmd) error
}

func (m *mockRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, name, args...)
	}
	return []byte("ok"), nil
}

func (m *mockRunner) RunInteractive(ctx context.Context, cmd *exec.Cmd) error {
	if m.execFunc != nil {
		return m.execFunc(ctx, cmd)
	}
	return nil
}

func TestDefaultRunner(t *testing.T) {
	runner := NewDefaultRunner()
	out, err := runner.Run(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "hello\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestDefaultRunnerInteractive(t *testing.T) {
	runner := NewDefaultRunner()
	cmd := exec.Command("true")
	if err := runner.RunInteractive(context.Background(), cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
