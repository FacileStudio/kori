package settings

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Gates ride the same file-to-resolved path as hooks and cron: a `gates:` key
// in the main settings is the chain, with every field of each spec intact.
func TestGatesComeFromTheFile(t *testing.T) {
	written(t, "gates:\n  - name: tests\n    command: [go, test, ./...]\n    timeout_secs: 300\n")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if len(config.Gates) != 1 {
		t.Fatalf("gates = %+v, want one", config.Gates)
	}
	want := GateSpec{Name: "tests", Command: []string{"go", "test", "./..."}, TimeoutSecs: 300}
	got := config.Gates[0]
	if got.Name != want.Name || !slices.Equal(got.Command, want.Command) ||
		got.TimeoutSecs != want.TimeoutSecs {
		t.Errorf("gate = %+v, want %+v", got, want)
	}
}

// A --gates-file exists to replace the chain: whoever names one is saying
// "these gates, not the ones in the main settings".
func TestGatesFileOverridesTheMainSettings(t *testing.T) {
	written(t, "gates:\n  - name: from-the-file\n    command: [true]\n")
	path := filepath.Join(t.TempDir(), "gates.yml")
	alternate := "gates:\n  - name: from-the-gates-file\n    command: [true]\n"
	if err := os.WriteFile(path, []byte(alternate), 0o600); err != nil {
		t.Fatalf("writing the gates file: %v", err)
	}

	config, err := settings(Config{GatesFile: path})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if len(config.Gates) != 1 || config.Gates[0].Name != "from-the-gates-file" {
		t.Errorf("gates = %+v, want the gates file's chain alone", config.Gates)
	}
}

// The config file may be absent without a word, but a gates file only exists
// because someone named it on the command line: one that is missing or
// malformed is a refusal, not a silent fall-through.
func TestAMissingOrMalformedGatesFileIsAnError(t *testing.T) {
	written(t, "")

	missing := filepath.Join(t.TempDir(), "never-written.yml")
	if _, err := settings(Config{GatesFile: missing}); err == nil {
		t.Fatal("a missing gates file was accepted")
	}

	malformed := filepath.Join(t.TempDir(), "broken.yml")
	if err := os.WriteFile(malformed, []byte("gates: [this is not a list"), 0o600); err != nil {
		t.Fatalf("writing the gates file: %v", err)
	}
	if _, err := settings(Config{GatesFile: malformed}); err == nil {
		t.Fatal("a malformed gates file was accepted")
	}
}
