package cmd

import (
	"testing"

	"github.com/FacileStudio/kori/internal/settings"
)

func TestNewSandboxCmd(t *testing.T) {
	cmd := newSandboxCmd()
	if cmd.Use != "sandbox <vm-name> [prompt]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Short == "" || cmd.Long == "" {
		t.Fatal("expected non-empty Short and Long descriptions")
	}
}

func TestBuildSandboxOptions_WithArgs(t *testing.T) {
	cmd := newSandboxCmd()
	f := sandboxFlags{
		sync:     true,
		snapshot: true,
		workdir:  "/custom/dir",
		user:     "customuser",
	}
	if err := cmd.Flags().Set("sync", "true"); err != nil {
		t.Fatalf("setting sync flag: %v", err)
	}
	if err := cmd.Flags().Set("snapshot", "true"); err != nil {
		t.Fatalf("setting snapshot flag: %v", err)
	}
	cfg := settings.Defaults("test")
	opts, err := buildSandboxOptions(cmd, &f, cfg, []string{"testvm", "hello", "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.VMName != "testvm" {
		t.Fatalf("expected VMName testvm, got %s", opts.VMName)
	}
	if opts.WorkDir != "/custom/dir" {
		t.Fatalf("expected WorkDir /custom/dir, got %s", opts.WorkDir)
	}
	if opts.User != "customuser" {
		t.Fatalf("expected User customuser, got %s", opts.User)
	}
	if !opts.Sync || !opts.Snapshot {
		t.Fatal("expected Sync and Snapshot to be true")
	}
	if opts.PrintPrompt != "hello world" {
		t.Fatalf("expected prompt 'hello world', got %s", opts.PrintPrompt)
	}
}

func TestBuildSandboxOptions_DefaultsFromConfig(t *testing.T) {
	cmd := newSandboxCmd()
	f := sandboxFlags{user: "boite"}
	autoSync := true
	autoSnap := false
	cfg := settings.Config{
		Sandbox: settings.Sandbox{
			VMName:       "pingu",
			Root:         "/home/yann/project",
			AutoSync:     &autoSync,
			AutoSnapshot: &autoSnap,
		},
	}
	opts, err := buildSandboxOptions(cmd, &f, cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.VMName != "pingu" {
		t.Fatalf("expected default VMName pingu, got %s", opts.VMName)
	}
	if opts.WorkDir != "/home/yann/project" {
		t.Fatalf("expected default workdir, got %s", opts.WorkDir)
	}
	if !opts.Sync || opts.Snapshot {
		t.Fatal("expected Sync true and Snapshot false from config")
	}
}

func TestBuildSandboxOptions_NoSyncOverrides(t *testing.T) {
	cmd := newSandboxCmd()
	f := sandboxFlags{noSync: true, user: "boite"}
	autoSync := true
	cfg := settings.Config{
		Sandbox: settings.Sandbox{
			VMName:   "pingu",
			AutoSync: &autoSync,
		},
	}
	opts, err := buildSandboxOptions(cmd, &f, cfg, []string{"pingu"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Sync {
		t.Fatal("expected Sync false when --no-sync is set")
	}
}

func TestBuildSandboxOptions_MissingVM(t *testing.T) {
	cmd := newSandboxCmd()
	f := sandboxFlags{user: "boite"}
	cfg := settings.Config{}
	_, err := buildSandboxOptions(cmd, &f, cfg, nil)
	if err == nil {
		t.Fatal("expected error when no VM name provided in args or config")
	}
}

func TestBuildSandboxOptions_ExplicitPrint(t *testing.T) {
	cmd := newSandboxCmd()
	f := sandboxFlags{printPrompt: "run tests", user: "boite"}
	cfg := settings.Defaults("test")
	opts, err := buildSandboxOptions(cmd, &f, cfg, []string{"pingu", "ignored", "positional"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.PrintPrompt != "run tests" {
		t.Fatalf("expected explicit print flag, got %s", opts.PrintPrompt)
	}
}
