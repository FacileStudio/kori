package agent

import (
	"testing"
)

func TestLocalToolsFiltersDisabledSearchTools(t *testing.T) {
	dir := t.TempDir()
	off := false

	config := Config{
		Session: Session{Root: dir},
		Toggles: Toggles{
			Bash:          &off,
			Fetch:         &off,
			FindFiles:     &off,
			SearchContent: &off,
		},
		Security: Security{PathIsolation: &off, DenyElevation: &off, EnvIsolation: &off},
	}

	set, local, err := localTools(config)
	if err != nil {
		t.Fatalf("localTools: %v", err)
	}
	defer set.Close()

	for _, tool := range local {
		if tool.Name() == "find_files" {
			t.Error("find_files should be filtered when FindFiles toggle is false")
		}
		if tool.Name() == "search_content" {
			t.Error("search_content should be filtered when SearchContent toggle is false")
		}
	}
}

func TestLocalToolsMountsSearchToolsByDefault(t *testing.T) {
	dir := t.TempDir()
	off := false

	config := Config{
		Session: Session{Root: dir},
		Toggles: Toggles{
			Bash:  &off,
			Fetch: &off,
		},
		Security: Security{PathIsolation: &off, DenyElevation: &off, EnvIsolation: &off},
	}

	set, local, err := localTools(config)
	if err != nil {
		t.Fatalf("localTools: %v", err)
	}
	defer set.Close()

	hasFind, hasSearch := false, false
	for _, tool := range local {
		if tool.Name() == "find_files" {
			hasFind = true
		}
		if tool.Name() == "search_content" {
			hasSearch = true
		}
	}
	if !hasFind {
		t.Error("find_files missing when FindFiles toggle is default")
	}
	if !hasSearch {
		t.Error("search_content missing when SearchContent toggle is default")
	}
}
