package agent

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ListCronJobs lists all scheduled jobs.
func ListCronJobs() error {
	return listCronJobs()
}

// RunCronJob executes a scheduled cron job by name.
func RunCronJob(name string) error {
	return runCronJob(name)
}

// TrustCronJob reviews and trusts a cron job file.
func TrustCronJob(name string, in io.Reader) error {
	return trustCronJob(name, in)
}

// InstallCronJob installs a cron job to crontab.
func InstallCronJob(name string) error {
	return installCronJob(name)
}

// InstallCronJobOptions installs a cron job to crontab with optional print-only mode.
func InstallCronJobOptions(name string, printOnly bool) error {
	return installCronJobOptions(name, printOnly)
}

// UninstallCronJob removes a cron job from crontab.
func UninstallCronJob(name string) error {
	return uninstallCronJob(name)
}

// EnsureUserPath augments PATH with common user binary directories.
func EnsureUserPath() {
	ensureUserPath()
}

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
