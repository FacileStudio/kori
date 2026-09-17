package cmd

import (
	"cmp"
	"errors"
	"strings"

	"github.com/FacileStudio/kori/internal/sandbox"
	"github.com/FacileStudio/kori/internal/settings"
	"github.com/spf13/cobra"
)

func resolveSandboxVM(cfg settings.Config, args []string) (string, []string, error) {
	if len(args) > 0 {
		return args[0], args[1:], nil
	}
	if cfg.Sandbox.VMName != "" {
		return cfg.Sandbox.VMName, nil, nil
	}
	return "", nil, errors.New("vm name is required: pass <vm-name> argument or set sandbox.vm_name in settings")
}

func resolveSandboxWorkdir(flagWorkdir string, cfg settings.Config) string {
	return cmp.Or(flagWorkdir, cfg.Sandbox.Workdir, cfg.Sandbox.Root, "/workspace")
}

func resolveSandboxSync(cmd *cobra.Command, f *sandboxFlags, cfg settings.Config) bool {
	if f.noSync {
		return false
	}
	if cmd.Flags().Changed("sync") {
		return f.sync
	}
	if cfg.Sandbox.AutoSync != nil {
		return *cfg.Sandbox.AutoSync
	}
	return false
}

func resolveSandboxSnapshot(cmd *cobra.Command, f *sandboxFlags, cfg settings.Config) bool {
	if cmd.Flags().Changed("snapshot") {
		return f.snapshot
	}
	if cfg.Sandbox.AutoSnapshot != nil {
		return *cfg.Sandbox.AutoSnapshot
	}
	return f.snapshot
}

func buildSandboxOptions(cmd *cobra.Command, f *sandboxFlags, cfg settings.Config, args []string) (sandbox.SessionOptions, error) {
	vmName, promptArgs, err := resolveSandboxVM(cfg, args)
	if err != nil {
		return sandbox.SessionOptions{}, err
	}
	prompt := f.printPrompt
	if prompt == "" && len(promptArgs) > 0 {
		prompt = strings.Join(promptArgs, " ")
	}
	return sandbox.SessionOptions{
		VMName:      vmName,
		WorkDir:     resolveSandboxWorkdir(f.workdir, cfg),
		User:        f.user,
		Sync:        resolveSandboxSync(cmd, f, cfg),
		NoSync:      f.noSync,
		Snapshot:    resolveSandboxSnapshot(cmd, f, cfg),
		PrintPrompt: prompt,
		Args:        promptArgs,
	}, nil
}
