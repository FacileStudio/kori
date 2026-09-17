package cmd

import (
	"testing"

	"github.com/FacileStudio/kori/internal/settings"
)

func TestNewSandboxCmd(t *testing.T) {
	cmd := newSandboxCmd()
	if cmd.Use != "sandbox [command]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Short == "" || cmd.Long == "" || cmd.Example == "" {
		t.Fatal("expected non-empty Short, Long, and Example descriptions")
	}
	subcommands := cmd.Commands()
	if len(subcommands) < 2 {
		t.Fatalf("expected at least 2 subcommands, got %d", len(subcommands))
	}
	foundList := false
	foundRun := false
	for _, sub := range subcommands {
		if sub.Name() == "list" {
			foundList = true
		}
		if sub.Name() == "run" {
			foundRun = true
		}
	}
	if !foundList || !foundRun {
		t.Fatalf("expected list and run subcommands, foundList=%t, foundRun=%t", foundList, foundRun)
	}
}

func TestSandboxListCmd(t *testing.T) {
	cmd := newSandboxListCmd()
	if cmd.Use != "list" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}
	if len(cmd.Aliases) == 0 || cmd.Aliases[0] != "ls" {
		t.Fatalf("expected alias 'ls', got %v", cmd.Aliases)
	}
	if err := runSandboxList(false); err != nil {
		t.Fatalf("runSandboxList error: %v", err)
	}
	if err := runSandboxList(true); err != nil {
		t.Fatalf("runSandboxList json error: %v", err)
	}
}

func TestSandboxRunCmd(t *testing.T) {
	f := sandboxFlags{}
	cmd := newSandboxRunCmd(&f)
	if cmd.Use != "run <vm-name> [prompt]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
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
