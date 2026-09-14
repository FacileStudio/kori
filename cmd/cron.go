package cmd

import (
	"os"

	"github.com/FacileStudio/kori/internal/agent"
	"github.com/spf13/cobra"
)

func newCronCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cron [command]",
		Short: "Manage scheduled headless agent jobs",
		Long: "Manage scheduled headless agent jobs that run from systemd or crontab.\n" +
			"Jobs are defined as YAML files in ~/.kori/jobs/.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return agent.ListCronJobs()
		},
	}
	cmd.AddCommand(newCronListCmd())
	cmd.AddCommand(newCronRunCmd())
	cmd.AddCommand(newCronTrustCmd())
	cmd.AddCommand(newCronInstallCmd())
	return cmd
}

func newCronListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"status"},
		Short:   "List all registered cron jobs",
		RunE: func(_ *cobra.Command, _ []string) error {
			return agent.ListCronJobs()
		},
	}
}

func newCronRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <name>",
		Short: "Execute a cron job once in headless mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return agent.RunCronJob(args[0])
		},
	}
}

func newCronTrustCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trust <name>",
		Short: "Review and trust a job file",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return agent.TrustCronJob(args[0], os.Stdin)
		},
	}
}

func newCronInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <name>",
		Short: "Install a systemd service and timer for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return agent.InstallCronJob(args[0])
		},
	}
}
