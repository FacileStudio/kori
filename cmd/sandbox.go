package cmd

import (
	"context"
	"errors"
	"strings"

	"github.com/FacileStudio/kori/internal/sandbox"
	"github.com/FacileStudio/kori/internal/settings"
	"github.com/spf13/cobra"
)

type sandboxFlags struct {
	sync        bool
	noSync      bool
	snapshot    bool
	workdir     string
	user        string
	printPrompt string
}

func newSandboxCmd() *cobra.Command {
	var f sandboxFlags
	cmd := &cobra.Command{
		Use:   "sandbox <vm-name> [prompt]",
		Short: "Start kori inside a boite VM sandbox",
		Long: "Start a kori agent session inside an isolated boite QEMU VM sandbox over SSH.\n" +
			"The session runs inside the guest VM with its filesystem strictly isolated.",
		RunE: func(c *cobra.Command, args []string) error {
			return runSandbox(c, &f, args)
		},
	}
	bindSandboxFlags(cmd, &f)
	return cmd
}

func bindSandboxFlags(cmd *cobra.Command, f *sandboxFlags) {
	fl := cmd.Flags()
	fl.BoolVar(&f.sync, "sync", false, "Sync host workspace into the VM before starting")
	fl.BoolVar(&f.noSync, "no-sync", false, "Disable workspace synchronization")
	fl.BoolVar(&f.snapshot, "snapshot", false, "Create a snapshot of the VM overlay disk on exit")
	fl.StringVarP(&f.workdir, "workdir", "w", "", "Working directory inside the VM")
	fl.StringVarP(&f.user, "user", "u", "boite", "SSH user for connecting to the VM")
	fl.StringVarP(&f.printPrompt, "print", "p", "", "Run prompt in headless mode inside the VM and stream output")
}

func runSandbox(cmd *cobra.Command, f *sandboxFlags, args []string) error {
	cfg, err := settings.Settings("", settings.Config{})
	if err != nil {
		return err
	}
	opts, err := buildSandboxOptions(cmd, f, cfg, args)
	if err != nil {
		return err
	}
	return sandbox.RunSession(context.Background(), opts)
}

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
	if flagWorkdir != "" {
		return flagWorkdir
	}
	if cfg.Sandbox.Workdir != "" {
		return cfg.Sandbox.Workdir
	}
	if cfg.Sandbox.Root != "" {
		return cfg.Sandbox.Root
	}
	return "/workspace"
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

func buildSandboxOptions(cmd *cobra.Command, f *sandboxFlags, cfg settings.Config, args []string) (sandbox.SessionOptions, error) {
	vmName, promptArgs, err := resolveSandboxVM(cfg, args)
	if err != nil {
		return sandbox.SessionOptions{}, err
	}

	workdir := resolveSandboxWorkdir(f.workdir, cfg)
	sync := resolveSandboxSync(cmd, f, cfg)

	snapshot := f.snapshot
	if !cmd.Flags().Changed("snapshot") && cfg.Sandbox.AutoSnapshot != nil {
		snapshot = *cfg.Sandbox.AutoSnapshot
	}

	prompt := f.printPrompt
	if prompt == "" && len(promptArgs) > 0 {
		prompt = strings.Join(promptArgs, " ")
	}

	return sandbox.SessionOptions{
		VMName:      vmName,
		WorkDir:     workdir,
		User:        f.user,
		Sync:        sync,
		NoSync:      f.noSync,
		Snapshot:    snapshot,
		PrintPrompt: prompt,
		Args:        promptArgs,
	}, nil
}
