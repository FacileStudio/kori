package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
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
		Use:   "sandbox [command]",
		Short: "Start kori inside an isolated boite VM sandbox",
		Long: `Manage and start kori agent sessions inside isolated boite QEMU VM sandboxes over SSH.

The agent session executes strictly inside the guest VM's filesystem and process tree.
Use 'kori sandbox list' to discover available instances, or 'kori sandbox <vm-name> [prompt]'
to launch a session.`,
		Example: `  # List available sandbox VMs
  kori sandbox list

  # Start an interactive session in the 'pingu' VM
  kori sandbox pingu

  # Run a prompt headlessly in the sandbox and stream output
  kori sandbox pingu "run tests and fix any failing cases"

  # Sync the kori binary before starting and snapshot on completion
  kori sandbox pingu --sync --snapshot`,
		RunE: func(c *cobra.Command, args []string) error {
			return runSandbox(c, &f, args)
		},
	}
	bindSandboxFlags(cmd, &f)
	cmd.AddCommand(newSandboxListCmd())
	cmd.AddCommand(newSandboxRunCmd(&f))
	return cmd
}

func newSandboxListCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available boite sandbox VMs",
		Long:    "Discover and list all registered boite VM sandbox instances, their runtime status, SSH ports, and paths.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSandboxList(jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output instance list in JSON format")
	return cmd
}

func newSandboxRunCmd(f *sandboxFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <vm-name> [prompt]",
		Short: "Run an agent session inside a sandbox VM",
		Long:  "Start a kori agent session inside the specified boite QEMU VM sandbox over SSH.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runSandbox(c, f, args)
		},
	}
	bindSandboxFlags(cmd, f)
	return cmd
}

func runSandboxList(jsonOutput bool) error {
	instances, err := sandbox.ListInstances()
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(instances)
	}
	if len(instances) == 0 {
		dir, _ := sandbox.InstancesDir()
		fmt.Printf("no boite sandbox instances found in %s\n", dir)
		return nil
	}
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Green)
	for _, inst := range instances {
		fmt.Println(nameStyle.Render(inst.Name))
		fmt.Printf("status=%s\nport=%d\nworkspace=%s\nkey=%s\n\n",
			inst.Status, inst.SSHPort, inst.Workspace, inst.KeyPath)
	}
	return nil
}

func bindSandboxFlags(cmd *cobra.Command, f *sandboxFlags) {
	fl := cmd.Flags()
	fl.BoolVar(&f.sync, "sync", false, "Sync kori binary into the VM before starting")
	fl.BoolVar(&f.noSync, "no-sync", false, "Disable binary synchronization")
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
	if len(args) == 0 && cfg.Sandbox.VMName == "" {
		return cmd.Help()
	}
	opts, err := buildSandboxOptions(cmd, f, cfg, args)
	if err != nil {
		return err
	}
	return sandbox.RunSession(context.Background(), opts)
}
