package toolview

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/nacelle"
)

func TestGroupLineSingle(t *testing.T) {
	g := Group{
		Name:  "read_file",
		Input: `{"path":"main.go"}`,
		Count: 1,
	}
	line := g.GroupLine(80)
	if !strings.Contains(line, "☰  read_file(main.go)") {
		t.Errorf("GroupLine = %q, want ☰ read_file(main.go)", line)
	}
}

func TestGroupLineBatch(t *testing.T) {
	g := Group{
		Name:      "read_file",
		Count:     3,
		CallNames: []string{"a.go", "b.go", "c.go"},
	}
	line := g.GroupLine(80)
	if !strings.Contains(line, "☰ 3 reads") {
		t.Errorf("GroupLine = %q, want '☰ 3 reads'", line)
	}
	if !strings.Contains(line, "a.go · b.go · c.go") {
		t.Errorf("GroupLine = %q, want filenames", line)
	}
}

func TestGroupLineBatchMCP(t *testing.T) {
	g := Group{
		Name:      "mycelium_memory_search",
		Count:     3,
		CallNames: []string{"a", "b", "c"},
		Tool:      nacelle.ToolEvent{Source: nacelle.ToolSourceMCP},
	}
	line := g.GroupLine(80)
	if !strings.Contains(line, "✻ 3 mcps") {
		t.Errorf("GroupLine = %q, want '✻ 3 mcps'", line)
	}
	if glyph := g.GroupGlyph(); glyph != "✻" {
		t.Errorf("GroupGlyph = %q, want ✻", glyph)
	}
}

func TestGroupGlyphs(t *testing.T) {
	g := Group{Name: "read_file", Count: 1}
	if got := g.GroupGlyph(); got != "☰" {
		t.Errorf("glyph = %q, want ☰", got)
	}

	g.Discarded = true
	if got := g.GroupGlyph(); got != "⊘" {
		t.Errorf("glyph = %q, want ⊘", got)
	}

	g.Discarded = false
	g.Failed = true
	if got := g.GroupGlyph(); got != "✗" {
		t.Errorf("glyph = %q, want ✗", got)
	}

	g = Group{Name: "read_file", Count: 3}
	if got := g.GroupGlyph(); got != "☰" {
		t.Errorf("batch glyph = %q, want ☰", got)
	}
}

func TestGroupFinishCallError(t *testing.T) {
	g := Group{Name: "run_command", Count: 2}
	ev := nacelle.ToolEvent{
		ID:       "1",
		Name:     "run_command",
		Duration: 50 * time.Millisecond,
		Err:      errors.New("fail"),
	}
	g.FinishCall(ev)
	if !g.Failed {
		t.Fatalf("expected g.Failed to be true")
	}
	if len(g.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(g.Errors))
	}
	if !g.End.IsZero() {
		t.Fatalf("expected g.End to be zero before all calls finish")
	}
}

func TestGroupFinishCallComplete(t *testing.T) {
	g := Group{
		Name:  "run_command",
		Count: 1,
		Start: time.Now().Add(-50 * time.Millisecond),
	}
	ev := nacelle.ToolEvent{
		ID:       "1",
		Name:     "run_command",
		Duration: 40 * time.Millisecond,
	}
	g.FinishCall(ev)
	if g.End.IsZero() {
		t.Fatalf("expected g.End to be set after all calls finish")
	}
	if g.FinishedCount != 1 {
		t.Fatalf("expected FinishedCount = 1, got %d", g.FinishedCount)
	}
	if dur := g.Duration(); dur <= 0 {
		t.Errorf("expected positive duration, got %v", dur)
	}
}
