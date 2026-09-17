package cmd

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/FacileStudio/kori/internal/sandbox"
	"github.com/FacileStudio/kori/internal/settings"
)

type sandboxListEntry struct {
	Name      string `json:"name"`
	Backend   string `json:"backend"`
	Host      string `json:"host,omitempty"`
	Port      int    `json:"port"`
	User      string `json:"user,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Key       string `json:"key,omitempty"`
	Status    string `json:"status,omitempty"`
}

func collectConfigTargetEntries(cfg settings.Config) []sandboxListEntry {
	var entries []sandboxListEntry
	for name, tgt := range cfg.Sandbox.Targets {
		backend := cmp.Or(tgt.Backend, "ssh")
		host := tgt.Host
		if host == "" && backend == "boite" {
			host = "127.0.0.1"
		}
		port := tgt.Port
		if port <= 0 {
			port = 22
			if backend == "boite" {
				port = 2226
			}
		}
		entries = append(entries, sandboxListEntry{
			Name:      name,
			Backend:   backend,
			Host:      host,
			Port:      port,
			User:      tgt.User,
			Workspace: cmp.Or(tgt.Workdir, tgt.Root),
			Key:       tgt.SSHKeyPath,
			Status:    "configured",
		})
	}
	return entries
}

func hasEntry(entries []sandboxListEntry, name string) bool {
	for _, e := range entries {
		if e.Name == name {
			return true
		}
	}
	return false
}

func collectBoiteInstanceEntries(entries []sandboxListEntry) []sandboxListEntry {
	instances, err := sandbox.ListInstances()
	if err != nil {
		return entries
	}
	for _, inst := range instances {
		if hasEntry(entries, inst.Name) {
			continue
		}
		entries = append(entries, sandboxListEntry{
			Name:      inst.Name,
			Backend:   "boite",
			Host:      "127.0.0.1",
			Port:      inst.SSHPort,
			Workspace: inst.Workspace,
			Key:       inst.KeyPath,
			Status:    inst.Status,
		})
	}
	return entries
}

func printSandboxTextList(entries []sandboxListEntry) {
	if len(entries) == 0 {
		fmt.Println("no sandbox targets configured or boite instances found")
		return
	}
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Green)
	for _, entry := range entries {
		fmt.Println(nameStyle.Render(entry.Name))
		fmt.Printf("backend=%s\nhost=%s\nport=%d\nstatus=%s\nworkspace=%s\nkey=%s\n\n",
			entry.Backend, entry.Host, entry.Port, entry.Status, entry.Workspace, entry.Key)
	}
}

func runSandboxList(jsonOutput bool) error {
	cfg, err := settings.Settings("", settings.Config{})
	if err != nil {
		return err
	}
	entries := collectConfigTargetEntries(cfg)
	entries = collectBoiteInstanceEntries(entries)
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}
	printSandboxTextList(entries)
	return nil
}
