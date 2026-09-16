package agent

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/FacileStudio/kori/internal/settings"
)

var cronNameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// listCronJobs prints the configured jobs and how to run or arm them. It arms
// nothing. Jobs come from ~/.kori/jobs/, one file each, so the listing is
// the folder as of this invocation.
func listCronJobs() error {
	config, files, err := loadCronState()
	if err != nil {
		return err
	}
	jobs := make([]settings.CronJob, 0, len(files))
	for _, f := range files {
		jobs = append(jobs, f.Job)
	}
	if settings.DerefBool(config.JSON) {
		return printCronJSON(jobs)
	}
	if len(files) == 0 {
		fmt.Println("no cron jobs in " + settings.JobsDir())
		return nil
	}
	for _, f := range files {
		trusted, err := settings.IsTrusted(f.Path, f.Raw)
		if err != nil {
			return err
		}
		job := f.Job
		fmt.Printf("%-20s when=%-18s enabled=%t commands=%t trusted=%t workdir=%s delivery=%s\n",
			job.Name, job.When, jobEnabled(job), jobCommands(job), trusted, job.Workdir, job.Delivery)
	}
	fmt.Println("\nrun one now:        kori cron run <name>")
	fmt.Println("trust one:          kori cron trust <name>")
	fmt.Println("arm its timer:      kori cron install <name>")
	return nil
}

// installCronJob generates, writes, and arms the systemd service + timer pair
// for one job under ~/.config/systemd/user/. It enables and starts the timer
// automatically so the job runs on schedule without further manual steps.
func installCronJob(name string) error {
	_, files, err := loadCronState()
	if err != nil {
		return err
	}
	f, err := findJobFile(files, name)
	if err != nil {
		return err
	}
	if err := ensureTrusted(f); err != nil {
		return err
	}
	job := f.Job
	if err := checkCronInstallable(job); err != nil {
		return err
	}
	bin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating kori: %w", err)
	}
	workdir := expandHome(job.Workdir)
	svc, timer := cronUnits(job.Name, bin, workdir, cmp.Or(job.When, "daily"), cmp.Or(job.Timeout, "300"))

	dir := filepath.Join(expandHome("~"), ".config", "systemd", "user")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, fmt.Sprintf("kori-%s.service", job.Name)), []byte(svc), 0o644)
	os.WriteFile(filepath.Join(dir, fmt.Sprintf("kori-%s.timer", job.Name)), []byte(timer), 0o644)
	if exec.Command("systemctl", "--user", "enable", fmt.Sprintf("kori-%s.timer", job.Name)).Run() != nil {
		return exec.Command("systemctl", "--user", "start", fmt.Sprintf("kori-%s.timer", job.Name)).Run()
	}
	return nil
}

// cronUnits generates systemd service and timer unit strings.
func cronUnits(name, bin, workdir, schedule, timeout string) (service, timer string) {
	exec := strings.Join([]string{
		systemdQuote(bin),
		systemdQuote("cron"),
		systemdQuote("run"),
		systemdQuote(name),
	}, " ")
	service = fmt.Sprintf(`[Unit]
Description=kori cron %[1]s

[Service]
Type=oneshot
WorkingDirectory=%[2]s
ExecStart=%[3]s
TimeoutStartSec=%[4]s

[Install]
WantedBy=default.target
`, name, workdir, exec, timeout)
	timer = fmt.Sprintf(`[Unit]
Description=schedule for kori cron %[1]s

[Timer]
OnCalendar=%[2]s
Unit=kori-%[1]s.service
Persistent=false

[Install]
WantedBy=timers.target
`, name, schedule)
	return service, timer
}

// systemdQuote makes one ExecStart token safe against whitespace splitting:
// double quotes with the two escapes systemd processes inside them.
func systemdQuote(arg string) string {
	arg = strings.ReplaceAll(arg, `\`, `\\`)
	arg = strings.ReplaceAll(arg, `"`, `\"`)
	return `"` + arg + `"`
}

// expandHome expands a path starting with ~/ to the user's home directory.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}

// checkCronInstallable enforces the arm-time policy: a unit-safe name, an
// explicit enabled: true after a test run, and a workdir — applyJob only sets
// Root from an explicit workdir, so a job without one would execute wherever
// the scheduler starts the unit. cron run needs the workdir rule alone; the
// enabled rule is install-only.
func checkCronInstallable(job settings.CronJob) error {
	if !cronNameRe.MatchString(job.Name) {
		return fmt.Errorf("invalid cron job name %q: unit file names only allow letters, digits, and . _ -", job.Name)
	}
	if !jobEnabled(job) {
		return fmt.Errorf("job %q is disabled: run `kori cron run %s`, confirm the output, then set enabled: true",
			job.Name, job.Name)
	}
	if job.Workdir == "" {
		return fmt.Errorf("job %q has no workdir: without one the run executes in the scheduler's working directory, not the project's; set workdir: /path/to/dir on the job", job.Name)
	}
	return nil
}

// jobEnabled and jobCommands read a job's pointer settings with the policy
// defaults filled in: a job is disabled and shell-less until it says otherwise.
func jobEnabled(job settings.CronJob) bool {
	return job.Enabled != nil && *job.Enabled
}

func jobCommands(job settings.CronJob) bool {
	return job.Commands != nil && *job.Commands
}
