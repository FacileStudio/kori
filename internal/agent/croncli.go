package agent

import (
	"encoding/json"
	"fmt"

	"github.com/FacileStudio/kori/internal/settings"
)

// UsageError marks a malformed cron invocation: main exits 2, the CLI
// standard's usage-error code, instead of 1.
type UsageError struct {
	err error
}

func (e *UsageError) Error() string { return e.err.Error() }

func usagef(format string, args ...any) error {
	return &UsageError{err: fmt.Errorf(format, args...)}
}

// printCronUsage writes the cron subcommand's usage to stdout; help exits 0.
func printCronUsage() error {
	fmt.Println(`usage: kori cron <command> [args]

  list              show the jobs in ~/.kori/jobs/
  run <name>        run one job now through the headless path
  trust <name>      review one job file and approve its contents
  install <name>    print the systemd service and timer that arm one job
  help              show this text

  --json            with list, print one JSON document instead of the text table`)
	return nil
}

// cronJobJSON is one job as cron list --json prints it: the fields the text
// table shows, with the pointer settings resolved to their policy defaults.
type cronJobJSON struct {
	Name     string `json:"name"`
	When     string `json:"when"`
	Enabled  bool   `json:"enabled"`
	Commands bool   `json:"commands"`
	Workdir  string `json:"workdir"`
	Delivery string `json:"delivery"`
}

func cronJobsJSON(jobs []settings.CronJob) ([]byte, error) {
	out := make([]cronJobJSON, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, cronJobJSON{
			Name:     job.Name,
			When:     job.When,
			Enabled:  job.IsEnabled(),
			Commands: jobCommands(job),
			Workdir:  job.Workdir,
			Delivery: job.Delivery,
		})
	}
	return json.Marshal(out)
}

// printCronJSON prints the one JSON document the CLI standard's --json asks
// for: data alone on stdout, no colors, no footer.
func printCronJSON(jobs []settings.CronJob) error {
	data, err := cronJobsJSON(jobs)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
