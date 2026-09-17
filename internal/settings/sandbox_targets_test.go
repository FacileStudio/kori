package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func testBaseMergeConfig() Config {
	syncTrue := true
	local := SandboxTarget{
		Backend:  "boite",
		VMName:   "pingu",
		Port:     2226,
		Root:     "/workspace",
		AutoSync: &syncTrue,
	}
	old := SandboxTarget{Backend: "ssh", Host: "old.host"}
	return Config{
		Sandbox: Sandbox{
			Default: "local",
			Targets: map[string]SandboxTarget{
				"local": local,
				"old":   old,
			},
		},
	}
}

func testOverMergeConfig() Config {
	syncFalse := false
	snapTrue := true
	local := SandboxTarget{
		Port:         2228,
		AutoSync:     &syncFalse,
		AutoSnapshot: &snapTrue,
		Workdir:      "/workspace/app",
	}
	cloud := SandboxTarget{
		Backend:    "ssh",
		Host:       "192.168.1.50",
		Port:       22,
		User:       "ubuntu",
		SSHKeyPath: "~/.ssh/cloud_key",
		Workdir:    "/srv/project",
	}
	return Config{
		Sandbox: Sandbox{
			Default: "cloud",
			Targets: map[string]SandboxTarget{
				"local": local,
				"cloud": cloud,
			},
		},
	}
}

func TestSandboxMergeTargets(t *testing.T) {
	base := testBaseMergeConfig()
	base.merge(testOverMergeConfig())

	if base.Sandbox.Default != "cloud" || len(base.Sandbox.Targets) != 3 {
		t.Fatalf("expected Default cloud and 3 targets: %+v", base.Sandbox)
	}
	local := base.Sandbox.Targets["local"]
	if local.Backend != "boite" || local.Port != 2228 || local.Workdir != "/workspace/app" {
		t.Fatalf("unexpected local target: %+v", local)
	}
	cloud := base.Sandbox.Targets["cloud"]
	if cloud.Backend != "ssh" || cloud.Host != "192.168.1.50" || cloud.User != "ubuntu" {
		t.Fatalf("unexpected cloud target: %+v", cloud)
	}
	old := base.Sandbox.Targets["old"]
	if old.Backend != "ssh" || old.Host != "old.host" {
		t.Fatalf("unexpected old target: %+v", old)
	}
}

func testSandboxTargetYAML() string {
	return `sandbox:
  default: remote
  user: globaluser
  vm_name: fallback-vm
  targets:
    local:
      backend: boite
      vm_name: local-pingu
      port: 2226
      ssh_key_path: ~/.ssh/id_ed25519
      root: /workspace
      auto_sync: true
      auto_snapshot: false
    remote:
      backend: ssh
      host: dev.server.internal
      port: 2222
      user: dev
      ssh_key_path: ~/.ssh/custom_id
      root: /root/dir
      workdir: /home/dev/work
      auto_sync: false
      auto_snapshot: true
`
}

func TestSandboxLoadYAMLWithTargets(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), ".kori.yml")
	if err := os.WriteFile(tmp, []byte(testSandboxTargetYAML()), 0o644); err != nil {
		t.Fatalf("writing tmp config: %v", err)
	}
	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if loaded.Sandbox.Default != "remote" || loaded.Sandbox.User != "globaluser" {
		t.Fatalf("unexpected loaded config: %+v", loaded.Sandbox)
	}
	loc := loaded.Sandbox.Targets["local"]
	if loc.Backend != "boite" || loc.VMName != "local-pingu" || loc.Port != 2226 {
		t.Fatalf("unexpected local target fields: %+v", loc)
	}
	rem := loaded.Sandbox.Targets["remote"]
	if rem.Backend != "ssh" || rem.Host != "dev.server.internal" || rem.Port != 2222 {
		t.Fatalf("unexpected remote target fields: %+v", rem)
	}
}
