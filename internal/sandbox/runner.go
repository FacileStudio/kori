// Package sandbox provides boite VM sandbox integration, state discovery,
// isolation verification, binary synchronization, and SSH execution.
package sandbox

import (
	"context"
	"os/exec"
)

// Runner abstracts command execution for sandbox operations.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	RunInteractive(ctx context.Context, cmd *exec.Cmd) error
}

// DefaultRunner executes commands using standard os/exec facilities.
type DefaultRunner struct{}

// NewDefaultRunner constructs a standard Runner backed by os/exec.
func NewDefaultRunner() Runner {
	return &DefaultRunner{}
}

// Run executes a command and returns its combined output.
func (r *DefaultRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// RunInteractive executes a command directly attached to standard streams.
func (r *DefaultRunner) RunInteractive(ctx context.Context, cmd *exec.Cmd) error {
	return cmd.Run()
}
