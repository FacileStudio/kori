package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxDefaults(t *testing.T) {
	cfg := Defaults("test")
	if cfg.Sandbox.VMName != "pingu" {
		t.Fatalf("expected VMName pingu, got %s", cfg.Sandbox.VMName)
	}
	if cfg.Sandbox.Port != 2226 {
		t.Fatalf("expected Port 2226, got %d", cfg.Sandbox.Port)
	}
	if cfg.Sandbox.Root != "/home/yann/project" {
		t.Fatalf("expected Root /home/yann/project, got %s", cfg.Sandbox.Root)
	}
	if cfg.Sandbox.AutoSync == nil || !*cfg.Sandbox.AutoSync {
		t.Fatal("expected AutoSync true")
	}
	if cfg.Sandbox.AutoSnapshot == nil || *cfg.Sandbox.AutoSnapshot {
		t.Fatal("expected AutoSnapshot false")
	}
}

func testSandboxMergeObj() (Config, Config) {
	base := Defaults("test")
	syncVal := false
	snapVal := true
	over := Config{
		Sandbox: Sandbox{
			VMName:       "tux",
			Port:         2230,
			SSHKeyPath:   "/custom/key",
			Root:         "/custom/root",
			Workdir:      "/custom/workdir",
			AutoSync:     &syncVal,
			AutoSnapshot: &snapVal,
		},
	}
	base.merge(over)
	return base, over
}

func TestSandboxMerge(t *testing.T) {
	base, _ := testSandboxMergeObj()
	if base.Sandbox.VMName != "tux" || base.Sandbox.Port != 2230 {
		t.Fatalf("expected tux:2230, got %s:%d", base.Sandbox.VMName, base.Sandbox.Port)
	}
	if base.Sandbox.SSHKeyPath != "/custom/key" || base.Sandbox.Root != "/custom/root" {
		t.Fatalf("expected /custom/key and /custom/root, got %+v", base.Sandbox)
	}
	if base.Sandbox.Workdir != "/custom/workdir" {
		t.Fatalf("expected workdir /custom/workdir, got %s", base.Sandbox.Workdir)
	}
	if base.Sandbox.AutoSync == nil || *base.Sandbox.AutoSync {
		t.Fatal("expected AutoSync false")
	}
	if base.Sandbox.AutoSnapshot == nil || !*base.Sandbox.AutoSnapshot {
		t.Fatal("expected AutoSnapshot true")
	}
}

func TestSandboxLoadYAML(t *testing.T) {
	yamlData := `sandbox:
  vm_name: arctic
  port: 2244
  ssh_key_path: /tmp/key
  root: /guest/ws
  workdir: /guest/custom
  auto_sync: false
  auto_snapshot: true
`
	tmp := filepath.Join(t.TempDir(), ".kori.yml")
	if err := os.WriteFile(tmp, []byte(yamlData), 0o644); err != nil {
		t.Fatalf("writing tmp config: %v", err)
	}
	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if loaded.Sandbox.VMName != "arctic" || loaded.Sandbox.Port != 2244 {
		t.Fatalf("expected arctic:2244, got %s:%d", loaded.Sandbox.VMName, loaded.Sandbox.Port)
	}
	if loaded.Sandbox.AutoSync == nil || *loaded.Sandbox.AutoSync {
		t.Fatal("expected AutoSync false")
	}
}
