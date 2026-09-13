package agent

import (
	"bufio"
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/FacileStudio/nacelle-tui/internal/settings"
)

// findCronCommand scans os.Args for the cron subcommand: flags typed ahead of
// it stay where they are, where the settings flag parser reads them; the scan
// only skips leading dash-tokens, so a flag that takes a value must follow
// the subcommand instead.
func findCronCommand() (prefix, sub []string, ok bool) {
	args := os.Args[1:]
	first := 0
	for first < len(args) && strings.HasPrefix(args[first], "-") {
		first++
	}
	if first >= len(args) || args[first] != "cron" {
		return nil, nil, false
	}
	return args[:first], args[first+1:], true
}

// checkCronFlag handles the `cron` subcommand, which fronts the headless run
// path so a scheduled job can be invoked from systemd or crontab with no TUI.
// It returns true when it recognised a cron command, and the caller should then
// treat its error as the process's result.
func checkCronFlag() (bool, error) {
	prefix, sub, ok := findCronCommand()
	if !ok {
		return false, nil
	}
	sub = hoistJSON(prefix, sub)
	if len(sub) == 0 || sub[0] == "list" || sub[0] == "status" {
		return true, listCronJobs()
	}
	switch sub[0] {
	case "help", "-h", "--help":
		return true, printCronUsage()
	case "run":
		if len(sub) < 2 {
			return true, usagef("usage: nacelle cron run <name>")
		}
		return true, runCronJob(sub[1])
	case "trust":
		if len(sub) < 2 {
			return true, usagef("usage: nacelle cron trust <name>")
		}
		return true, trustCronJob(sub[1], os.Stdin)
	case "install":
		if len(sub) < 2 {
			return true, usagef("usage: nacelle cron install <name>")
		}
		return true, installCronJob(sub[1])
	default:
		return true, usagef("unknown cron command %q: want run, trust, install, or list", sub[0])
	}
}

// hoistJSON moves a trailing -json ahead of the subcommand: the settings flag
// parser stops at the first non-flag word, which "cron" always is, so a flag
// typed after the subcommand would otherwise be invisible to it. There is a
// precedent for rewriting os.Args mid-dispatch in extractPrintFlag.
func hoistJSON(prefix, sub []string) []string {
	rest := make([]string, 0, len(sub))
	json := ""
	for _, arg := range sub {
		if arg == "-json" || arg == "--json" || strings.HasPrefix(arg, "-json=") || strings.HasPrefix(arg, "--json=") {
			json = arg
			continue
		}
		rest = append(rest, arg)
	}
	if json != "" {
		os.Args = append(append([]string{os.Args[0]}, prefix...), append([]string{json, "cron"}, rest...)...)
	}
	return rest
}

var cronConfigOnce = sync.OnceValues(func() (settings.Config, error) {
	flags := settings.FromFlags(settings.Defaults(""))
	return settings.Settings(DefaultSystemPrompt(), flags)
})

// loadCronConfig resolves the settings layer once per process: the flag
// parser cannot be run twice, and every cron command in one invocation reads
// the same config anyway. Job files stay outside the cache — the folder is
// re-read per command so the sync stays live.
func loadCronConfig() (settings.Config, error) {
	return cronConfigOnce()
}

// loadCronState reads the resolved config and every job file in
// ~/.nacelle/jobs/. Every cron command goes through here, so a file added or
// removed takes effect on the next invocation with no daemon.
func loadCronState() (settings.Config, []settings.JobFile, error) {
	config, err := loadCronConfig()
	if err != nil {
		return settings.Config{}, nil, err
	}
	files, err := settings.LoadJobFiles(settings.JobsDir())
	if err != nil {
		return settings.Config{}, nil, err
	}
	return config, files, nil
}

func findJobFile(files []settings.JobFile, name string) (settings.JobFile, error) {
	for _, f := range files {
		if f.Job.Name == name {
			return f, nil
		}
	}
	return settings.JobFile{}, fmt.Errorf("no cron job named %q", name)
}

// ensureTrusted refuses a job whose file is not the exact content a human
// trusted. The refusal names the trust command so the way forward is on
// screen, not in the docs.
func ensureTrusted(f settings.JobFile) error {
	ok, err := settings.IsTrusted(f.Path, f.Raw)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return fmt.Errorf("job %q is not trusted: review %s, then run `nacelle cron trust %s`", f.Job.Name, f.Path, f.Job.Name)
}

// trustCronJob shows one job file's contents and records the human's approval
// keyed on the file path and content hash. An edit re-arms the gate; removal
// of the file retires the record with the job.
func trustCronJob(name string, in io.Reader) error {
	_, files, err := loadCronState()
	if err != nil {
		return err
	}
	f, err := findJobFile(files, name)
	if err != nil {
		return err
	}
	fmt.Printf("job:     %s\nfile:    %s\nwhen:    %s\ndelivery: %s\n\n%s\n",
		f.Job.Name, f.Path, cmp.Or(f.Job.When, "daily"), cmp.Or(f.Job.Delivery, "none"), string(f.Raw))
	fmt.Print("\ntrust this job to run unattended? [y/N] ")
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("job %q left untrusted", name)
	}
	return settings.Save(f.Path, f.Raw)
}
