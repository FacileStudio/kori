package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// InstanceState represents the serialized configuration and runtime state of a boite VM.
type InstanceState struct {
	Name           string    `json:"name"`
	PID            int       `json:"pid"`
	SSHPort        int       `json:"ssh_port"`
	OverlayPath    string    `json:"overlay_path"`
	ConfigDiskPath string    `json:"config_disk_path"`
	KeyPath        string    `json:"key_path"`
	PubKeyPath     string    `json:"pub_key_path"`
	CreatedAt      time.Time `json:"created_at"`
	Status         string    `json:"status"`
	Workspace      string    `json:"workspace"`
	NoMount        bool      `json:"no_mount"`
	PinnedEnv      []string  `json:"pinned_env"`
}

// InstancesDir returns the filesystem directory where boite instance state folders reside.
func InstancesDir() (string, error) {
	if dir := os.Getenv("BOITE_INSTANCES_DIR"); dir != "" {
		return dir, nil
	}
	if base := os.Getenv("BOITE_HOME"); base != "" {
		return filepath.Join(base, "instances"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".boite", "instances"), nil
}

// FindStatePath returns the path to state.json for a given instance name.
func FindStatePath(name string) (string, error) {
	if name == "" {
		return "", errors.New("instance name cannot be empty")
	}
	dir, err := InstancesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name, "state.json"), nil
}

// LoadInstanceStateFile loads and parses a state.json file from the specified path.
func LoadInstanceStateFile(path string) (*InstanceState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file %s: %w", path, err)
	}
	var state InstanceState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state json: %w", err)
	}
	if err := ValidateState(&state); err != nil {
		return nil, err
	}
	return &state, nil
}

// LoadInstance loads instance state by instance name.
func LoadInstance(name string) (*InstanceState, error) {
	path, err := FindStatePath(name)
	if err != nil {
		return nil, err
	}
	return LoadInstanceStateFile(path)
}

// ValidateState validates that required fields in an instance state are set and valid.
func ValidateState(st *InstanceState) error {
	if st == nil {
		return errors.New("instance state is nil")
	}
	if st.Name == "" {
		return errors.New("instance state name is empty")
	}
	if st.SSHPort <= 0 {
		return fmt.Errorf("invalid SSH port %d for instance %s", st.SSHPort, st.Name)
	}
	if st.KeyPath == "" {
		return fmt.Errorf("missing SSH key path for instance %s", st.Name)
	}
	return nil
}

// ListInstances discovers and loads all valid boite instances on the system.
func ListInstances() ([]*InstanceState, error) {
	dir, err := InstancesDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var instances []*InstanceState
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		st, err := LoadInstance(entry.Name())
		if err == nil && st != nil {
			instances = append(instances, st)
		}
	}
	return instances, nil
}
