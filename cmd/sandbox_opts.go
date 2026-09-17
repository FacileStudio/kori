package cmd

import (
	"cmp"
	"strings"

	"github.com/FacileStudio/kori/internal/sandbox"
	"github.com/FacileStudio/kori/internal/settings"
	"github.com/spf13/cobra"
)

func resolveSandboxTargetName(cfg settings.Config, args []string) (string, []string) {
	if len(args) > 0 {
		return args[0], args[1:]
	}
	return cmp.Or(cfg.Sandbox.Default, cfg.Sandbox.VMName), nil
}

func resolveSandboxSync(cmd *cobra.Command, f *sandboxFlags, target *sandbox.Target, cfg settings.Config) bool {
	if f.noSync {
		return false
	}
	if cmd.Flags().Changed("sync") {
		return f.sync
	}
	if target != nil && target.AutoSync != nil {
		return *target.AutoSync
	}
	if cfg.Sandbox.AutoSync != nil {
		return *cfg.Sandbox.AutoSync
	}
	return false
}

func resolveSandboxSnapshot(cmd *cobra.Command, f *sandboxFlags, target *sandbox.Target, cfg settings.Config) bool {
	if cmd.Flags().Changed("snapshot") {
		return f.snapshot
	}
	if target != nil && target.AutoSnapshot != nil {
		return *target.AutoSnapshot
	}
	if cfg.Sandbox.AutoSnapshot != nil {
		return *cfg.Sandbox.AutoSnapshot
	}
	return f.snapshot
}

func bindSandboxFlags(cmd *cobra.Command, f *sandboxFlags) {
	fl := cmd.Flags()
	fl.BoolVar(&f.sync, "sync", false, "Sync kori binary into the target before starting")
	fl.BoolVar(&f.noSync, "no-sync", false, "Disable binary synchronization")
	fl.BoolVar(&f.snapshot, "snapshot", false, "Create a snapshot of the VM overlay disk on exit (boite only)")
	fl.StringVarP(&f.workdir, "workdir", "w", "", "Working directory inside the target")
	fl.StringVarP(&f.user, "user", "u", "", "SSH user for connecting to the target")
	fl.StringVarP(&f.printPrompt, "print", "p", "", "Run prompt in headless mode inside the target and stream output")
}

func buildSandboxOptions(cmd *cobra.Command, f *sandboxFlags, cfg settings.Config, args []string) (sandbox.SessionOptions, error) {
	targetName, promptArgs := resolveSandboxTargetName(cfg, args)
	target, err := sandbox.ResolveTarget(targetName, cfg)
	if err != nil {
		return sandbox.SessionOptions{}, err
	}
	if f.user != "" {
		target.User = f.user
	}
	if f.workdir != "" {
		target.Workdir = f.workdir
	}
	prompt := f.printPrompt
	if prompt == "" && len(promptArgs) > 0 {
		prompt = strings.Join(promptArgs, " ")
	}
	return sandbox.SessionOptions{
		Target:      target,
		VMName:      target.Name,
		WorkDir:     target.Workdir,
		User:        target.User,
		Sync:        resolveSandboxSync(cmd, f, target, cfg),
		NoSync:      f.noSync,
		Snapshot:    resolveSandboxSnapshot(cmd, f, target, cfg),
		PrintPrompt: prompt,
		Args:        promptArgs,
	}, nil
}
