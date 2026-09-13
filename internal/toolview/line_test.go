package toolview

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestPrimaryArg(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"a known key", `{"path":"view.go"}`, "view.go"},
		{"the first known key wins", `{"query":"tools","path":"view.go"}`, "view.go"},
		{"a command", `{"command":"go test ./...","timeout":60}`, "go test ./..."},
		{"the noise around it is dropped", `{"file_path":"run.go","offset":0,"limit":2000}`, "run.go"},
		{"the only key, whatever it is called", `{"expression":"1+1"}`, "1+1"},
		{"nothing to pick between unknown keys", `{"left":1,"right":2}`, ""},
		{"a value that is not a string keeps its json", `{"lines":[1,2,3]}`, "[1,2,3]"},
		{"a number", `{"limit":40}`, "40"},
		{"whitespace is collapsed onto the one line", "{\"command\":\"a\\n  b\"}", "a b"},
		{"an empty object", `{}`, ""},
		{"no input at all", ``, ""},
		{"input that is not an object", `"just a string"`, ""},
		{"input that is not json", `not json at all`, ""},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := PrimaryArg(test.input); got != test.want {
				t.Errorf("PrimaryArg(%s) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestToolLine(t *testing.T) {
	if got := ToolLine("read_file", `not json`, 80); got != "⏺ read_file()" {
		t.Errorf("line = %q, want ⏺ read_file()", got)
	}
	if got := ToolLine("list_tools", ``, 80); got != "⏺ list_tools()" {
		t.Errorf("line = %q, want ⏺ list_tools()", got)
	}
}

func TestToolLineCutToFit(t *testing.T) {
	long := `{"path":"` + strings.Repeat("deep/", 40) + `file.go"}`
	line := ToolLine("read_file", long, 40)

	if width := lipgloss.Width(line); width > 40-DurationRoom {
		t.Errorf("line width %d exceeds budget", width)
	}
	if !strings.HasPrefix(line, "⏺ read_file(deep/") || !strings.HasSuffix(line, "…)") {
		t.Errorf("line = %q, want prefix and ellipsis", line)
	}
}

func TestToolLineNarrowWindow(t *testing.T) {
	if got := ToolLine("run_command", `{"command":"go build ./..."}`, 12); got != "$ run_command()" {
		t.Errorf("line = %q, want name only", got)
	}
}

func TestToolLineRepeatedKey(t *testing.T) {
	got := ToolLine("run_command", `{"command":"ls","command":"rm -rf /"}`, 80)
	if got != "$ run_command(input has a duplicate key)" {
		t.Errorf("line = %q, want duplicate key error", got)
	}
}
