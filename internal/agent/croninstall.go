package agent

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/FacileStudio/kori/internal/settings"
	"github.com/adhocore/gronx"
)

var cronNameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

var cronNameStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Green)

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
		fmt.Printf(cronNameStyle.Render("%s"), job.Name)
		fmt.Printf("\nwhen=%s\nenabled=%t\ncommands=%t\ntrusted=%t\nworkdir=%s\ndelivery=%s\n\n",
			job.When, job.IsEnabled(), job.Commands != nil && *job.Commands, trusted, job.Workdir, job.Delivery)
	}
	fmt.Println("(You can update your job files in the ~/.kori/jobs/ folder)")
	return nil
}

// installCronJob outputs a single crontab line that can be copied into `crontab -e`.
// It validates the job using gronx and prints the schedule and command to run.
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
	schedule := cmp.Or(job.When, "daily")
	if schedule == "daily" || schedule == "*-*-*" {
		schedule = "0 0 * * *"
	}
	if !gronx.IsValid(schedule) {
		return fmt.Errorf("invalid cron expression %q in job %q: must be a valid cron expression", schedule, job.Name)
	}
	cmd := fmt.Sprintf("%s cron run %s", bin, name)
	fmt.Printf("%s %s\n\n", schedule, cmd)
	fmt.Println("(Use \"crontab -e\" command and paste the line above into it then save and close)")
	return nil
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
	if job.Enabled == nil || !*job.Enabled {
		return fmt.Errorf("job %q is disabled: run `kori cron run %s`, confirm the output, then set enabled: true",
			job.Name, job.Name)
	}
	if job.Workdir == "" {
		return fmt.Errorf("job %q has no workdir: without one the run executes in the scheduler's working directory, not the project's; set workdir: /path/to/dir on the job", job.Name)
	}
	return nil
}
