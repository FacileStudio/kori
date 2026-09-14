package tui

import (
	"testing"
)

func TestParseParallelTasksObjectsWithTitles(t *testing.T) {
	input := `{"tasks":[{"title":"audit auth","task":"read internal/auth"},{"title":"run tests","prompt":"go test ./..."},{"title":"build app","instruction":"go build ./..."}]}`
	tasks, titles := ParseParallelTasks(input)
	if len(tasks) != 3 || len(titles) != 3 {
		t.Fatalf("got %d tasks, %d titles, want 3 each", len(tasks), len(titles))
	}
	if tasks[0] != "read internal/auth" || titles[0] != "audit auth" {
		t.Errorf("task 0 = (%q, %q), want (read internal/auth, audit auth)", tasks[0], titles[0])
	}
	if tasks[1] != "go test ./..." || titles[1] != "run tests" {
		t.Errorf("task 1 = (%q, %q), want (go test ./..., run tests)", tasks[1], titles[1])
	}
	if tasks[2] != "go build ./..." || titles[2] != "build app" {
		t.Errorf("task 2 = (%q, %q), want (go build ./..., build app)", tasks[2], titles[2])
	}
}

func TestParseParallelTasksInlineBracketTitles(t *testing.T) {
	input := `{"tasks":["[audit auth] read internal/auth","Warning: run tests carefully","plain task without title"]}`
	tasks, titles := ParseParallelTasks(input)
	if len(tasks) != 3 || len(titles) != 3 {
		t.Fatalf("got %d tasks, %d titles, want 3 each", len(tasks), len(titles))
	}
	if titles[0] != "audit auth" || tasks[0] != "read internal/auth" {
		t.Errorf("item 0 = (%q, %q)", titles[0], tasks[0])
	}
	if titles[1] != "" || tasks[1] != "Warning: run tests carefully" {
		t.Errorf("item 1 = (%q, %q), want empty title for colon-containing prompt", titles[1], tasks[1])
	}
	if titles[2] != "" || tasks[2] != "plain task without title" {
		t.Errorf("item 2 = (%q, %q)", titles[2], tasks[2])
	}
}

func TestParseInlineTaskTitleBrackets(t *testing.T) {
	cases := []struct {
		input     string
		wantTitle string
		wantTask  string
	}{
		{"[audit auth] check tokens", "audit auth", "check tokens"},
		{"[short title] do work", "short title", "do work"},
		{"Warning: check tokens", "", "Warning: check tokens"},
		{"http://example.com/api: check", "", "http://example.com/api: check"},
		{"a single task with no title prefix", "", "a single task with no title prefix"},
	}
	for _, tc := range cases {
		title, task := parseInlineTaskTitle(tc.input)
		if title != tc.wantTitle || task != tc.wantTask {
			t.Errorf("parseInlineTaskTitle(%q) = (%q, %q), want (%q, %q)", tc.input, title, task, tc.wantTitle, tc.wantTask)
		}
	}
}

func TestSplitParallelTasksWithTitles(t *testing.T) {
	raw := "[clean cache] rm -rf .cache, Warning: check logs, plain task"
	tasks, titles := splitParallelTasksWithTitles(raw)
	if len(tasks) != 3 || len(titles) != 3 {
		t.Fatalf("got %d tasks, %d titles, want 3 each", len(tasks), len(titles))
	}
	if titles[0] != "clean cache" || tasks[0] != "rm -rf .cache" {
		t.Errorf("item 0 = (%q, %q)", titles[0], tasks[0])
	}
	if titles[1] != "" || tasks[1] != "Warning: check logs" {
		t.Errorf("item 1 = (%q, %q)", titles[1], tasks[1])
	}
	if titles[2] != "" || tasks[2] != "plain task" {
		t.Errorf("item 2 = (%q, %q)", titles[2], tasks[2])
	}
}
