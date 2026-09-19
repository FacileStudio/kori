package cmd

import (
	"github.com/spf13/cobra"

	"github.com/FacileStudio/kori/internal/settings"
)

func newRemoteCmd() *cobra.Command {
	var f remoteFlags
	cmd := &cobra.Command{
		Use:   "remote [host] [prompt]",
		Short: "Run a kori session against a remote SSH host",
		Long: `Run kori on the host with every tool call executing on a remote SSH host. The
model reaches the host's workspace but never sees the local filesystem, and the
provider keys stay here: only tool calls and their output cross the connection.

The host may be a remote.targets entry, a direct user@host:port address, or a
host alias from ~/.ssh/config. Local boite microVMs are handled by 'kori sandbox';
use 'kori remote list' to see the configured hosts.`,
		Example: `  # List configured remote hosts
  kori remote list

  # Start an interactive session on a configured host
  kori remote staging

  # Connect straight to an address
  kori remote deploy@build.example.com:2222

  # Run a prompt headlessly and stream the output
  kori remote staging "rebase onto main and run the tests"`,
		Args: cobra.ArbitraryArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return runRemote(c, &f, args)
		},
	}
	bindRemoteFlags(cmd, &f)
	cmd.AddCommand(newRemoteListCmd())
	return cmd
}

func runRemote(cmd *cobra.Command, f *remoteFlags, args []string) error {
	cfg, err := settings.Settings("", settings.Config{})
	if err != nil {
		return err
	}
	if len(args) == 0 && cfg.Remote.Default == "" {
		return cmd.Help()
	}
	opts, err := buildRemoteOptions(f, cfg, args)
	if err != nil {
		return err
	}
	return runTargetSession(cmd.Context(), rootVersion(cmd), cfg, opts)
}
