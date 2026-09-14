package agent

import (
	"errors"
	"fmt"
	"time"

	"github.com/FacileStudio/nacelle"

	"github.com/FacileStudio/kori/internal/settings"
)

// applyJob projects a job's overrides onto the resolved session config, group
// by group, member by member: an empty member falls back to the config, a set
// one wins. The two autonomy fields deliberately invert the interactive
// defaults and are applied last so a job cannot re-arm them: a run no one is
// watching cannot answer an approval prompt, so the gate is never armed and
// shell stays off unless the job opts in.
func applyJob(config settings.Config, job settings.CronJob) settings.Config {
	cfg := config
	if job.Workdir != "" {
		cfg.Root = expandHome(job.Workdir)
	}
	cfg = mergeProvider(cfg, job)
	cfg = mergeSecurity(cfg, job)
	cfg = mergeToggles(cfg, job)
	cfg = mergeReasoning(cfg, job)
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

// mergeProvider copies the job's provider members that say something, the
// group's model beating the flat model key.
func mergeProvider(cfg settings.Config, job settings.CronJob) settings.Config {
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
	return cfg
}

// mergeSecurity copies the job's set security pointers; approve_tools is
// ignored because applyJob forces it off afterwards.
func mergeSecurity(cfg settings.Config, job settings.CronJob) settings.Config {
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

// mergeToggles copies the job's set tool toggles; run_command is handled in
// applyJob where the commands: default lives.
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
	return cfg
}

// mergeReasoning copies the job's set reasoning members.
func mergeReasoning(cfg settings.Config, job settings.CronJob) settings.Config {
	if job.Reasoning.Effort != "" {
		cfg.Effort = job.Reasoning.Effort
	}
	if job.Reasoning.Thinking != nil {
		cfg.Thinking = job.Reasoning.Thinking
	}
	if job.Reasoning.Budget != nil {
		cfg.Budget = job.Reasoning.Budget
	}
	return cfg
}

// mergeLimits copies the job's set limit members.
func mergeLimits(cfg settings.Config, job settings.CronJob) settings.Config {
	if job.Limits.MaxIterations != nil {
		cfg.MaxIterations = job.Limits.MaxIterations
	}
	if job.Limits.CompactAt != nil {
		cfg.CompactAt = job.Limits.CompactAt
	}
	return cfg
}

// jobHooks builds the library hooks a job's own hooks: block lists. They ride
// the headless build's extra parameter next to the compaction counter.
func jobHooks(job settings.CronJob) (map[nacelle.HookPoint][]nacelle.Hook, error) {
	if len(job.Hooks) == 0 {
		return nil, nil
	}
	return settings.BuildHooks(job.Hooks)
}

func runCronJob(name string) error {
	config, files, err := loadCronState()
	if err != nil {
		return err
	}
	f, err := findJobFile(files, name)
	if err != nil {
		return err
	}
	job := f.Job
	if err := ensureTrusted(f); err != nil {
		return err
	}
	if job.Prompt == "" {
		return fmt.Errorf("job %q has no prompt: nothing to run", name)
	}
	if err := validateDelivery(job.Delivery); err != nil {
		return err
	}
	cfg := applyJob(config, job)
	extra, err := jobHooks(job)
	if err != nil {
		return err
	}
	started := time.Now()
	text, stats, runErr := runHeadlessConfig(job.Prompt, cfg, extra)
	rec, status := stats.record(job, cfg.Model, started, runErr)
	logErr := appendCronLog(job.Delivery, job.Name, status, text)
	return errors.Join(runErr, logErr, appendCronRun(rec))
}
