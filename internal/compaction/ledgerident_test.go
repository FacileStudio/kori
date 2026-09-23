package compaction

import (
	"slices"
	"testing"
)

func TestIdentifiersPicksTheTokensARewriteMayNotDrop(t *testing.T) {
	body := "Decisions:\n- ran `go test ./...`\n- touched internal/tui/compact.go\n" +
		"- pinned --keep-turns and v1.2.3\n- and an ordinary sentence about it"

	got := Identifiers(body)

	for _, want := range []string{"go test ./...", "internal/tui/compact.go", "--keep-turns", "v1.2.3"} {
		if !slices.Contains(got, want) {
			t.Errorf("Identifiers = %v, want %q among them", got, want)
		}
	}
	for _, unwanted := range []string{"ordinary", "sentence", "about", "it", "Decisions"} {
		if slices.Contains(got, unwanted) {
			t.Errorf("Identifiers = %v, want the plain word %q left out", got, unwanted)
		}
	}
}

func TestIdentifiersNamesEachTokenOnce(t *testing.T) {
	got := Identifiers("`internal/tui/compact.go` and internal/tui/compact.go again")

	if len(got) != 1 || got[0] != "internal/tui/compact.go" {
		t.Errorf("Identifiers = %v, want the one identifier once", got)
	}
}

func TestMissingIdentifiersIsEmptyWhenTheRewriteKeepsThem(t *testing.T) {
	previous := "State:\n- edited internal/compaction/apply.go and ran `go build ./...`"
	body := "State:\n- consolidated: internal/compaction/apply.go, `go build ./...` still green"

	if missing := MissingIdentifiers(previous, body); len(missing) != 0 {
		t.Errorf("MissingIdentifiers = %v, want nothing missing", missing)
	}
}

func TestMissingIdentifiersNamesTheOnesTheRewriteLost(t *testing.T) {
	previous := "State:\n- edited internal/compaction/apply.go and pinned --keep-turns"
	body := "State:\n- consolidated the notes"

	missing := MissingIdentifiers(previous, body)

	for _, want := range []string{"internal/compaction/apply.go", "--keep-turns"} {
		if !slices.Contains(missing, want) {
			t.Errorf("MissingIdentifiers = %v, want %q named", missing, want)
		}
	}
}
