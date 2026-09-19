package cmd

import (
	"github.com/spf13/cobra"

	"github.com/FacileStudio/kori/internal/settings"
)

type sandboxFlags struct {
	snapshot    bool
	workdir     string
	user        string
	printPrompt string
}

func newSandboxCmd() *cobra.Command {
	var f sandboxFlags
	cmd := &cobra.Command{
		Use:   "sandbox [vm] [prompt]",
		Short: "Run a kori session against a local boite microVM",
		Long: `Run kori on the host with every tool call executing inside a local boite
microVM over SSH. The model reaches the VM's workspace but never sees the host
filesystem, and no API key or kori binary is ever shipped into the guest.

Use 'kori sandbox list' to discover available VMs, or 'kori sandbox <vm> [prompt]'
to start a session. SSH hosts that are not boite VMs are handled by 'kori remote'.`,
		Example: `  # List local boite VMs and configured sandbox targets
  kori sandbox list

  # Start an interactive session in a VM
  kori sandbox pingu

  # Run a prompt headlessly in the VM and stream output
  kori sandbox pingu "run the test suite and report failures"

  # Snapshot the VM overlay disk when the session ends
  kori sandbox pingu --snapshot`,
		Args: cobra.ArbitraryArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return runSandbox(c, &f, args)
		},
	}
	bindSandboxFlags(cmd, &f)
	cmd.AddCommand(newSandboxListCmd())
	return cmd
}

func newSandboxListCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List local boite VMs and configured sandbox targets",
		Long:    "Discover and list every configured sandbox target and registered boite VM instance.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSandboxList(jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output the target list as JSON")
	return cmd
}

func rootVersion(c *cobra.Command) string {
	if c == nil {
		return ""
	}
	return c.Root().Version
}

func runSandbox(cmd *cobra.Command, f *sandboxFlags, args []string) error {
	cfg, err := settings.Settings("", settings.Config{})
	if err != nil {
		return err
	}
	if len(args) == 0 && cfg.Sandbox.Default == "" && cfg.Sandbox.VMName == "" {
		return cmd.Help()
	}
	opts, err := buildSandboxOptions(cmd, f, cfg, args)
	if err != nil {
		return err
	}
	if err := runTargetSession(cmd.Context(), rootVersion(cmd), cfg, opts); err != nil {
		return err
	}
	return triggerSnapshotIfRequested(cmd.Context(), opts.Target, opts)
}
