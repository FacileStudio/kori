package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/kori/internal/sandbox"
	"github.com/FacileStudio/kori/internal/settings"
)

func TestRootVersion(t *testing.T) {
	if rootVersion(nil) != "" {
		t.Fatal("expected empty string for nil command")
	}
	root := &cobra.Command{Use: "root", Version: "1.2.3"}
	sub := &cobra.Command{Use: "sub"}
	root.AddCommand(sub)
	if rootVersion(sub) != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %s", rootVersion(sub))
	}
}

func TestTriggerSnapshotIfRequested(t *testing.T) {
	ctx := context.Background()
	if err := triggerSnapshotIfRequested(ctx, nil, sandbox.SessionOptions{Snapshot: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sshTarget := &sandbox.Target{Name: "host", Backend: "ssh"}
	if err := triggerSnapshotIfRequested(ctx, sshTarget, sandbox.SessionOptions{Snapshot: true}); err != nil {
		t.Fatalf("unexpected error for non-boite backend: %v", err)
	}
}

func TestRunTargetSession_PreflightFailure(t *testing.T) {
	ctx := context.Background()
	target := &sandbox.Target{
		Name:    "pingu",
		Backend: "boite",
		Status:  "stopped",
	}
	cfg := settings.Defaults("test")
	opts := sandbox.SessionOptions{
		Target:  target,
		WorkDir: "/workspace",
		User:    "boite",
	}
	err := runTargetSession(ctx, "0.0.1", cfg, opts)
	if err == nil || !strings.Contains(err.Error(), "must be running") {
		t.Fatalf("expected stopped error from preflight, got %v", err)
	}
}
