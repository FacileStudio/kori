package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/FacileStudio/nacelle"

	"github.com/FacileStudio/kori/internal/settings"
)

func applyJob(config settings.Config, job settings.CronJob) settings.Config {
	cfg := config
	if job.Workdir != "" {
		cfg.Root = expandHome(job.Workdir)
	}
	cfg = mergeConfig(cfg, job)
	cfg = mergeToggles(cfg, job)
	cfg = mergeLimits(cfg, job)
	cfg.Gates = append(cfg.Gates, job.Gates...)
	bash := false
	if job.Commands != nil {
		bash = *job.Commands
	}
	if job.Toggles.Bash != nil {
		bash = *job.Toggles.Bash
	}
	cfg.Bash = &bash
	approve := false
	cfg.ApproveTools = &approve
	return cfg
}

func mergeConfig(cfg settings.Config, job settings.CronJob) settings.Config {
	if job.Provider.Backend != "" {
		cfg.Backend = job.Provider.Backend
	}
	if job.Provider.BaseURL != "" {
		cfg.BaseURL = job.Provider.BaseURL
	}
	if job.Provider.APIKey != "" {
		cfg.APIKey = job.Provider.APIKey
	}
	if job.Provider.Model != "" {
		cfg.Model = job.Provider.Model
	} else if job.Model != "" {
		cfg.Model = job.Model
	}
	if job.Security.PathIsolation != nil {
		cfg.PathIsolation = job.Security.PathIsolation
	}
	if job.Security.DenyElevation != nil {
		cfg.DenyElevation = job.Security.DenyElevation
	}
	if job.Security.EnvIsolation != nil {
		cfg.EnvIsolation = job.Security.EnvIsolation
	}
	return cfg
}

func mergeToggles(cfg settings.Config, job settings.CronJob) settings.Config {
	if job.Toggles.ParallelAgents != nil {
		cfg.ParallelAgents = job.Toggles.ParallelAgents
	}
	if job.Toggles.Fetch != nil {
		cfg.Fetch = job.Toggles.Fetch
	}
	if job.Toggles.Tasks != nil {
		cfg.Tasks = job.Toggles.Tasks
	}
	if job.Toggles.Diagnostics != nil {
		cfg.Diagnostics = job.Toggles.Diagnostics
	}
	if job.Toggles.SearchContent != nil {
		cfg.SearchContent = job.Toggles.SearchContent
	}
	if job.Toggles.FindFiles != nil {
		cfg.FindFiles = job.Toggles.FindFiles
	}
	return cfg
}

func mergeLimits(cfg settings.Config, job settings.CronJob) settings.Config {
	if job.Reasoning.Effort != "" {
		cfg.Effort = job.Reasoning.Effort
	}
	if job.Reasoning.Thinking != nil {
		cfg.Thinking = job.Reasoning.Thinking
	}
	if job.Reasoning.Budget != nil {
		cfg.Budget = job.Reasoning.Budget
	}
	if job.Limits.MaxIterations != nil {
		cfg.MaxIterations = job.Limits.MaxIterations
	}
	if job.Limits.CompactAt != nil {
		cfg.CompactAt = job.Limits.CompactAt
	}
	return cfg
}

func jobHooks(job settings.CronJob) (map[nacelle.HookPoint][]nacelle.Hook, error) {
	if len(job.Hooks) == 0 {
		return nil, nil
	}
	return settings.BuildHooks(job.Hooks)
}

func jobContext(timeout string) (context.Context, context.CancelFunc) {
	if timeout == "" {
		return context.Background(), func() {}
	}
	d, err := time.ParseDuration(timeout)
	if err != nil || d <= 0 {
		return context.Background(), func() {}
	}
	return context.WithTimeout(context.Background(), d)
}

func loadExecutableJob(name string) (settings.CronJob, settings.Config, error) {
	config, files, err := loadCronState()
	if err != nil {
		return settings.CronJob{}, settings.Config{}, err
	}
	f, err := findJobFile(files, name)
	if err != nil {
		return settings.CronJob{}, settings.Config{}, err
	}
	if err := ensureTrusted(f); err != nil {
		return settings.CronJob{}, settings.Config{}, err
	}
	if f.Job.Prompt == "" {
		return settings.CronJob{}, settings.Config{}, fmt.Errorf("job %q has no prompt: nothing to run", name)
	}
	if err := validateDelivery(f.Job.Delivery); err != nil {
		return settings.CronJob{}, settings.Config{}, err
	}
	return f.Job, applyJob(config, f.Job), nil
}

func runCronJob(name string) error {
	ensureUserPath()
	job, cfg, err := loadExecutableJob(name)
	if err != nil {
		return err
	}
	extra, err := jobHooks(job)
	if err != nil {
		return err
	}
	ctx, cancel := jobContext(job.Timeout)
	defer cancel()
	started := time.Now()
	text, stats, runErr := runHeadlessConfigToContext(ctx, os.Stdout, job.Prompt, cfg, extra)
	rec, status := stats.record(job, cfg.Model, started, runErr)
	logText := text
	if runErr != nil && strings.TrimSpace(logText) == "" {
		logText = fmt.Sprintf("Error: %s\n", runErr.Error())
	}
	logErr := appendCronLog(job.Delivery, job.Name, status, logText)
	return errors.Join(runErr, logErr, appendCronRun(rec))
}
