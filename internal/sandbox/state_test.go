package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateState(t *testing.T) {
	if err := ValidateState(nil); err == nil {
		t.Fatal("expected error for nil state")
	}
	invalid := &InstanceState{Name: ""}
	if err := ValidateState(invalid); err == nil {
		t.Fatal("expected error for empty name")
	}
	noPort := &InstanceState{Name: "test", SSHPort: 0}
	if err := ValidateState(noPort); err == nil {
		t.Fatal("expected error for zero port")
	}
	noKey := &InstanceState{Name: "test", SSHPort: 2226, KeyPath: ""}
	if err := ValidateState(noKey); err == nil {
		t.Fatal("expected error for empty key path")
	}
	valid := &InstanceState{Name: "test", SSHPort: 2226, KeyPath: "/path/to/key"}
	if err := ValidateState(valid); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestLoadInstanceStateFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	jsonContent := `{
		"name": "pingu",
		"ssh_port": 2226,
		"key_path": "/home/user/.ssh/id_ed25519",
		"status": "running"
	}`
	if err := os.WriteFile(statePath, []byte(jsonContent), 0o644); err != nil {
		t.Fatalf("writing state file: %v", err)
	}
	st, err := LoadInstanceStateFile(statePath)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if st.Name != "pingu" || st.SSHPort != 2226 {
		t.Fatalf("unexpected state values: %+v", st)
	}
}

func TestFindStatePathEmpty(t *testing.T) {
	_, err := FindStatePath("")
	if err == nil {
		t.Fatal("expected error for empty instance name")
	}
}
