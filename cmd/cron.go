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
		Long: `Manage scheduled headless agent jobs that run from systemd or crontab.

Jobs are defined as YAML files in ~/.kori/jobs/. Use one of the
subcommands to list, run, trust, or install them. Each job can be
trusted with kori cron trust <name> before it can be run or
installed. Use kori cron install <name> to generate systemd unit
files for automated scheduling.`,
		RunE: func(c *cobra.Command, _ []string) error {
			return c.Help()
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
		Long:    "List all registered cron jobs from YAML files in ~/.kori/jobs/, showing their armed state and trust status.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return agent.ListCronJobs()
		},
	}
}

func newCronRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <name>",
		Short: "Execute a cron job once in headless mode",
		Long:  "Run a specific cron job once in headless mode. The job runs without interactive prompts.",
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
		Long:  "Review a cron job file and approve its contents before running it.",
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
		Long:  "Print a systemd service and timer unit file for a cron job, which you can save and enable.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return agent.InstallCronJob(args[0])
		},
	}
}
