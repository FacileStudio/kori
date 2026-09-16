package agent

import (
	"testing"

	"github.com/FacileStudio/kori/internal/settings"
)

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
