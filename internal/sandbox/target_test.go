package sandbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/kori/internal/settings"
)

func TestTargetFromInstance(t *testing.T) {
	if TargetFromInstance(nil) != nil {
		t.Fatal("expected nil for nil instance")
	}
	st := &InstanceState{
		Name:      "testvm",
		SSHPort:   2226,
		KeyPath:   "/path/to/key",
		Workspace: "/workspace",
		Status:    "running",
	}
	tgt := TargetFromInstance(st)
	if tgt.Name != "testvm" || tgt.Backend != "boite" || tgt.Host != "127.0.0.1" || tgt.Port != 2226 {
		t.Fatalf("unexpected target: %+v", tgt)
	}
	if tgt.User != "boite" || tgt.KeyPath != "/path/to/key" || tgt.Workdir != "/workspace" || tgt.Status != "running" {
		t.Fatalf("unexpected target details: %+v", tgt)
	}
}

func TestTargetFromSSH(t *testing.T) {
	tgt := TargetFromSSH("remote", "192.168.1.100", 2222, "deploy", "/key")
	if tgt.Name != "remote" || tgt.Backend != "ssh" || tgt.Host != "192.168.1.100" || tgt.Port != 2222 {
		t.Fatalf("unexpected ssh target: %+v", tgt)
	}
	if tgt.User != "deploy" || tgt.KeyPath != "/key" || tgt.Status != "running" {
		t.Fatalf("unexpected ssh target details: %+v", tgt)
	}
}

func createStagingConfig() settings.Config {
	targets := map[string]settings.SandboxTarget{
		"staging": {
			Backend:    "ssh",
			Host:       "staging.example.com",
			Port:       2200,
			User:       "admin",
			SSHKeyPath: "~/.ssh/custom",
			Workdir:    "/var/www",
		},
	}
	return settings.Config{
		Sandbox: settings.Sandbox{
			Targets: targets,
		},
	}
}

func TestResolveTarget_FromConfigTargets(t *testing.T) {
	cfg := createStagingConfig()
	tgt, err := ResolveTarget("staging", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tgt.Backend != "ssh" || tgt.Host != "staging.example.com" || tgt.Port != 2200 || tgt.User != "admin" {
		t.Fatalf("unexpected resolved target: %+v", tgt)
	}
	if tgt.Workdir != "/var/www" || tgt.KeyPath != "~/.ssh/custom" {
		t.Fatalf("unexpected resolved target paths: %+v", tgt)
	}
}

func TestResolveTarget_FromBoiteInstance(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("BOITE_INSTANCES_DIR", tmp)
	instDir := filepath.Join(tmp, "local-box")
	if err := os.MkdirAll(instDir, 0o755); err != nil {
		t.Fatalf("failed to create instance dir: %v", err)
	}
	stateJSON := `{"name":"local-box","ssh_port":2230,"key_path":"/key/path","status":"running","workspace":"/ws"}`
	if err := os.WriteFile(filepath.Join(instDir, "state.json"), []byte(stateJSON), 0o644); err != nil {
		t.Fatalf("failed to write state.json: %v", err)
	}
	cfg := settings.Config{}
	tgt, err := ResolveTarget("local-box", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tgt.Backend != "boite" || tgt.Port != 2230 || tgt.Workdir != "/ws" {
		t.Fatalf("unexpected boite target: %+v", tgt)
	}
}

func TestResolveTarget_DirectSSH(t *testing.T) {
	t.Setenv("BOITE_INSTANCES_DIR", t.TempDir())
	cfg := settings.Config{
		Sandbox: settings.Sandbox{
			User:       "globaluser",
			SSHKeyPath: "~/.ssh/id_rsa",
			Workdir:    "/default/work",
		},
	}
	tgt, err := ResolveTarget("ubuntu@10.0.0.5:2222", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tgt.Backend != "ssh" || tgt.Host != "10.0.0.5" || tgt.Port != 2222 || tgt.User != "ubuntu" {
		t.Fatalf("unexpected parsed ssh target: %+v", tgt)
	}
	if tgt.Workdir != "/default/work" || tgt.KeyPath != "~/.ssh/id_rsa" {
		t.Fatalf("unexpected paths on parsed ssh target: %+v", tgt)
	}
}

func TestResolveTarget_Default(t *testing.T) {
	targets := map[string]settings.SandboxTarget{
		"cloud": {
			Backend: "ssh",
			Host:    "cloud.host",
			Port:    22,
		},
	}
	cfg := settings.Config{
		Sandbox: settings.Sandbox{
			Default: "cloud",
			Targets: targets,
		},
	}
	tgt, err := ResolveTarget("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tgt.Name != "cloud" || tgt.Host != "cloud.host" {
		t.Fatalf("unexpected default target resolved: %+v", tgt)
	}
}
