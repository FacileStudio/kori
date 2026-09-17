package sandbox

import (
	"cmp"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/FacileStudio/kori/internal/settings"
)

// Target represents an isolated execution target (Boite microVM or SSH host).
type Target struct {
	Name         string `json:"name" yaml:"name"`
	Backend      string `json:"backend" yaml:"backend"`
	Host         string `json:"host" yaml:"host"`
	Port         int    `json:"port" yaml:"port"`
	User         string `json:"user" yaml:"user"`
	KeyPath      string `json:"key_path" yaml:"key_path"`
	Workdir      string `json:"workdir" yaml:"workdir"`
	Status       string `json:"status" yaml:"status"`
	AutoSync     *bool  `json:"auto_sync,omitempty" yaml:"auto_sync,omitempty"`
	AutoSnapshot *bool  `json:"auto_snapshot,omitempty" yaml:"auto_snapshot,omitempty"`
}

// TargetFromInstance creates a Target configured from a Boite InstanceState.
func TargetFromInstance(st *InstanceState) *Target {
	if st == nil {
		return nil
	}
	port := st.SSHPort
	if port <= 0 {
		port = 2226
	}
	return &Target{
		Name:    st.Name,
		Backend: "boite",
		Host:    "127.0.0.1",
		Port:    port,
		User:    "boite",
		KeyPath: st.KeyPath,
		Workdir: st.Workspace,
		Status:  st.Status,
	}
}

// TargetFromSSH creates a Target configured for generic SSH execution.
func TargetFromSSH(name, host string, port int, user, keyPath string) *Target {
	p := port
	if p <= 0 {
		p = 22
	}
	h := host
	if h == "" {
		h = "127.0.0.1"
	}
	return &Target{
		Name:    name,
		Backend: "ssh",
		Host:    h,
		Port:    p,
		User:    user,
		KeyPath: keyPath,
		Status:  "running",
	}
}

func parseSSHAddress(addr string) (user, host string, port int) {
	remaining := addr
	if at := strings.Index(remaining, "@"); at != -1 {
		user = remaining[:at]
		remaining = remaining[at+1:]
	}
	if h, pStr, err := net.SplitHostPort(remaining); err == nil {
		host = h
		if p, err := strconv.Atoi(pStr); err == nil {
			port = p
		}
	} else {
		host = remaining
		port = 22
	}
	return user, host, port
}

func resolveConfigBoiteTarget(name string, t settings.SandboxTarget) (*Target, error) {
	vmName := cmp.Or(t.VMName, name)
	inst, err := LoadInstance(vmName)
	if err != nil {
		return nil, fmt.Errorf("loading boite instance %s: %w", vmName, err)
	}
	target := TargetFromInstance(inst)
	if t.User != "" {
		target.User = t.User
	}
	if t.Port > 0 {
		target.Port = t.Port
	}
	if t.SSHKeyPath != "" {
		target.KeyPath = t.SSHKeyPath
	}
	if t.Workdir != "" || t.Root != "" {
		target.Workdir = cmp.Or(t.Workdir, t.Root)
	}
	target.AutoSync = t.AutoSync
	target.AutoSnapshot = t.AutoSnapshot
	return target, nil
}

func resolveTargetFromConfig(name string, cfg settings.Config) (*Target, bool, error) {
	if cfg.Sandbox.Targets == nil {
		return nil, false, nil
	}
	t, ok := cfg.Sandbox.Targets[name]
	if !ok {
		return nil, false, nil
	}
	if t.Backend == "boite" {
		target, err := resolveConfigBoiteTarget(name, t)
		return target, true, err
	}
	port := t.Port
	if port <= 0 {
		port = 22
	}
	return &Target{
		Name:         name,
		Backend:      "ssh",
		Host:         cmp.Or(t.Host, name),
		Port:         port,
		User:         cmp.Or(t.User, cfg.Sandbox.User),
		KeyPath:      cmp.Or(t.SSHKeyPath, cfg.Sandbox.SSHKeyPath),
		Workdir:      cmp.Or(t.Workdir, t.Root, cfg.Sandbox.Workdir, cfg.Sandbox.Root, "/workspace"),
		Status:       "running",
		AutoSync:     t.AutoSync,
		AutoSnapshot: t.AutoSnapshot,
	}, true, nil
}

func resolveTargetFromInstance(name string, cfg settings.Config) (*Target, bool) {
	inst, err := LoadInstance(name)
	if err != nil || inst == nil {
		return nil, false
	}
	target := TargetFromInstance(inst)
	if cfg.Sandbox.User != "" {
		target.User = cfg.Sandbox.User
	}
	if cfg.Sandbox.Workdir != "" || cfg.Sandbox.Root != "" {
		target.Workdir = cmp.Or(cfg.Sandbox.Workdir, cfg.Sandbox.Root)
	}
	target.AutoSync = cfg.Sandbox.AutoSync
	target.AutoSnapshot = cfg.Sandbox.AutoSnapshot
	return target, true
}

func resolveTargetFromAddress(name string, cfg settings.Config) *Target {
	u, h, p := parseSSHAddress(name)
	return &Target{
		Name:         name,
		Backend:      "ssh",
		Host:         h,
		Port:         p,
		User:         cmp.Or(u, cfg.Sandbox.User),
		KeyPath:      cfg.Sandbox.SSHKeyPath,
		Workdir:      cmp.Or(cfg.Sandbox.Workdir, cfg.Sandbox.Root, "/workspace"),
		Status:       "running",
		AutoSync:     cfg.Sandbox.AutoSync,
		AutoSnapshot: cfg.Sandbox.AutoSnapshot,
	}
}

// ResolveTarget resolves target parameters from settings, local Boite state, or SSH address.
func ResolveTarget(name string, cfg settings.Config) (*Target, error) {
	targetName := cmp.Or(name, cfg.Sandbox.Default, cfg.Sandbox.VMName)
	if targetName == "" {
		return nil, errors.New("sandbox target is required: pass <target> argument or set sandbox.default in settings")
	}
	if target, found, err := resolveTargetFromConfig(targetName, cfg); found {
		return target, err
	}
	if target, found := resolveTargetFromInstance(targetName, cfg); found {
		return target, nil
	}
	return resolveTargetFromAddress(targetName, cfg), nil
}
