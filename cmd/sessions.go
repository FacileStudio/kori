package cmd

import (
	"fmt"

	"github.com/FacileStudio/kori/internal/agent"
	"github.com/FacileStudio/kori/internal/sessions"
	"github.com/FacileStudio/kori/internal/settings"
	"github.com/spf13/cobra"
)

func newSessionsCmd(version string) *cobra.Command {
	var f sessionsListFlags
	cmd := &cobra.Command{
		Use:     "sessions [command]",
		Aliases: []string{"session"},
		Short:   "Manage background and interactive agent sessions",
		Long: `Manage background and interactive kori agent sessions.

Use 'kori sessions list' (or 'kori list') to view active and past sessions.
Use 'kori sessions attach <id>' to resume an interactive session.
Use 'kori sessions kill <id>' to terminate a running background session.`,
		RunE: func(c *cobra.Command, args []string) error {
			return c.Help()
		},
	}
	bindSessionsListFlags(cmd, &f)
	cmd.AddCommand(newSessionsListCmd(&f))
	cmd.AddCommand(newSessionsAttachCmd(version))
	cmd.AddCommand(newSessionsKillCmd())
	return cmd
}

func newResumeCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:     "resume <id>",
		Aliases: []string{"attach"},
		Short:   "Resume an agent session interactively",
		Long:    "Attach to and resume an existing agent session by session ID or PID.",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSessionsAttach(version, args[0])
		},
	}
}

func newSessionsAttachCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:     "attach <id>",
		Aliases: []string{"resume"},
		Short:   "Resume an agent session interactively",
		Long:    "Attach to and resume an existing agent session by session ID or PID.",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSessionsAttach(version, args[0])
		},
	}
}

func newSessionsKillCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "kill <id>",
		Aliases: []string{"stop"},
		Short:   "Kill a running background session",
		Long:    "Terminate a running background agent session by session ID or PID.",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSessionsKill(args[0])
		},
	}
}

func runSessionsAttach(version string, id string) error {
	cfg, err := settings.Settings("", settings.Config{})
	if err != nil {
		return err
	}
	cfg.Resume = &id
	return agent.RunSessionWithFlags(version, cfg)
}

func runSessionsKill(id string) error {
	session, err := sessions.GetSession(id)
	if err != nil {
		return err
	}
	if err := sessions.KillSession(id); err != nil {
		return err
	}
	fmt.Printf("killed session %s (PID %d)\n", id, session.PID)
	return nil
}
