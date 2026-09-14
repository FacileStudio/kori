package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

// The scaffold must round-trip: written as the template, read back through
// the strict decoder, and merged with the defaults it must not disturb.
func TestScaffoldRoundTripsToTheDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".kori.yml")

	created, err := Scaffold(path)
	if err != nil || !created {
		t.Fatalf("scaffold: created=%t err=%v", created, err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("the scaffolded file does not parse: %v", err)
	}

	resolved := Defaults("")
	resolved.mergeStrings(loaded)
	resolved.merge(loaded)

	base := Defaults("")
	want, _ := yaml.Marshal(base)
	got, _ := yaml.Marshal(resolved)
	if string(want) != string(got) {
		t.Errorf("scaffold+merge drifted from the defaults:\nbase %s\nresolved %s", want, got)
	}

	again, err := Scaffold(path)
	if err != nil || again {
		t.Errorf("scaffold rewrote an existing file: created=%t err=%v", again, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("existing file vanished: %v", err)
	}
}

// The scaffold must name every setting or the file stops being the
// discoverability surface it exists to be: a key missing from the template is
// a setting nobody finds, and the round-trip above cannot catch it because an
// unmentioned key just falls through to the default it already matches.
func TestTemplateScaffoldsDenyElevation(t *testing.T) {
	if !strings.Contains(Template, "deny_elevation: true") {
		t.Error("the template does not scaffold deny_elevation: true")
	}
}
