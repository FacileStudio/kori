package agent

import (
	"strings"
	"testing"

	"github.com/FacileStudio/kori/internal/settings"
)

func TestCronUnits(t *testing.T) {
	svc, timer := cronUnits("daily-brief", "/opt/kori/bin/kori", "/home/y/code/app", "Mon..Fri *-*-* 09:00", "300")

	for _, want := range []string{
		"WorkingDirectory=/home/y/code/app",
		`ExecStart="/opt/kori/bin/kori" "cron" "run" "daily-brief"`,
		"TimeoutStartSec=300",
	} {
		if !strings.Contains(svc, want) {
			t.Errorf("service unit missing %q:\n%s", want, svc)
		}
	}
	if strings.Contains(svc, "TimeoutStopSec") {
		t.Errorf("service unit still emits TimeoutStopSec:\n%s", svc)
	}
	for _, want := range []string{
		"OnCalendar=Mon..Fri *-*-* 09:00",
		"Unit=kori-daily-brief.service",
	} {
		if !strings.Contains(timer, want) {
			t.Errorf("timer unit missing %q:\n%s", want, timer)
		}
	}
}

func TestSystemdQuote(t *testing.T) {
	if got, want := systemdQuote(`a"b\c`), `"a\"b\\c"`; got != want {
		t.Errorf("systemdQuote = %s, want %s", got, want)
	}
	if got, want := systemdQuote("prog with space"), `"prog with space"`; got != want {
		t.Errorf("systemdQuote = %s, want %s", got, want)
	}
}

func TestCheckCronInstallable(t *testing.T) {
	tru := true
	job := settings.CronJob{Name: "daily-brief", Workdir: "~/x", Enabled: &tru}
	if err := checkCronInstallable(job); err != nil {
		t.Errorf("checkCronInstallable(valid job) = %v, want nil", err)
	}

	for _, bad := range []string{"", "has space", "sl/ash", `quo"te`, "café", "job$"} {
		job.Name = bad
		if err := checkCronInstallable(job); err == nil {
			t.Errorf("checkCronInstallable(name %q) = nil, want error", bad)
		}
	}

	job.Name = "daily-brief"
	job.Workdir = ""
	if err := checkCronInstallable(job); err == nil {
		t.Errorf("checkCronInstallable(no workdir) = nil, want error")
	}

	job.Workdir = "~/x"
	job.Enabled = nil
	if err := checkCronInstallable(job); err == nil {
		t.Errorf("checkCronInstallable(disabled) = nil, want error")
	}
}
