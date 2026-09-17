package cmd

import (
	"context"

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
		Short: "Start kori inside an isolated sandbox VM or remote SSH host",
		Long: `Manage and start kori agent sessions inside isolated microVM sandboxes or remote SSH hosts.

The agent session executes strictly inside the guest environment's filesystem and process tree.
Use 'kori sandbox list' to discover available targets, or 'kori sandbox <target> [prompt]'
to launch a session.`,
		Example: `  # List available sandbox targets and VMs
  kori sandbox list

  # Start an interactive session in a target
  kori sandbox <target>

  # Run a prompt headlessly in the sandbox and stream output
  kori sandbox <target> "run tests and fix any failing cases"

  # Sync the kori binary before starting and snapshot on completion
  kori sandbox <target> --sync --snapshot`,
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
		Short:   "List available sandbox targets and boite VMs",
		Long:    "Discover and list all configured sandbox targets and registered boite VM instances.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSandboxList(jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output instance list in JSON format")
	return cmd
}

func newSandboxRunCmd(f *sandboxFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <target> [prompt]",
		Short: "Run an agent session inside a sandbox target",
		Long:  "Start a kori agent session inside the specified sandbox VM or remote SSH host.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runSandbox(c, f, args)
		},
	}
	bindSandboxFlags(cmd, f)
	return cmd
}

func runSandbox(cmd *cobra.Command, f *sandboxFlags, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
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
