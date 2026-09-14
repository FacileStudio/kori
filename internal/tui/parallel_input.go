package tui

import (
	"encoding/json"
	"strings"
)

// ParseParallelTasks extracts task instructions and optional titles from
// the model's tool input payload or command strings.
func ParseParallelTasks(rawInput string) ([]string, []string) {
	var payload struct {
		Tasks json.RawMessage `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(rawInput), &payload); err != nil || len(payload.Tasks) == 0 {
		return nil, nil
	}
	var rawItems []json.RawMessage
	if err := json.Unmarshal(payload.Tasks, &rawItems); err == nil {
		tasks := make([]string, 0, len(rawItems))
		titles := make([]string, 0, len(rawItems))
		for _, item := range rawItems {
			p, t := parseSingleTaskItem(item)
			tasks = append(tasks, p)
			titles = append(titles, t)
		}
		return tasks, titles
	}
	var single string
	if err := json.Unmarshal(payload.Tasks, &single); err == nil {
		trimmed := strings.TrimSpace(single)
		if strings.HasPrefix(trimmed, "[") {
			return ParseParallelTasks(`{"tasks":` + trimmed + `}`)
		}
		t, p := parseInlineTaskTitle(trimmed)
		return []string{p}, []string{t}
	}
	return nil, nil
}

type taskItem struct {
	Title       string `json:"title"`
	Desc        string `json:"description"`
	Task        string `json:"task"`
	Prompt      string `json:"prompt"`
	Instruction string `json:"instruction"`
}

func (obj taskItem) empty() bool {
	return obj.Task == "" && obj.Prompt == "" && obj.Title == "" && obj.Instruction == ""
}

func (obj taskItem) resolve() (string, string) {
	prompt := obj.Task
	switch {
	case prompt != "":
	case obj.Prompt != "":
		prompt = obj.Prompt
	case obj.Instruction != "":
		prompt = obj.Instruction
	default:
		prompt = obj.Title
	}
	title := obj.Title
	if title == "" {
		title = obj.Desc
	}
	return prompt, title
}

func parseSingleTaskItem(item json.RawMessage) (string, string) {
	var obj taskItem
	if err := json.Unmarshal(item, &obj); err == nil && !obj.empty() {
		return obj.resolve()
	}
	var str string
	if err := json.Unmarshal(item, &str); err == nil {
		t, p := parseInlineTaskTitle(str)
		return p, t
	}
	return string(item), ""
}

// parseInlineTaskTitle extracts an optional bracketed title from a task description.
func parseInlineTaskTitle(s string) (string, string) {
	s = strings.TrimSpace(s)
	if title, prompt, ok := parseBracketTitle(s); ok {
		return title, prompt
	}
	return "", s
}

func parseBracketTitle(s string) (string, string, bool) {
	if !strings.HasPrefix(s, "[") {
		return "", s, false
	}
	end := strings.Index(s, "]")
	if end <= 1 || end >= len(s)-1 {
		return "", s, false
	}
	title := strings.TrimSpace(s[1:end])
	prompt := strings.TrimSpace(s[end+1:])
	if len(strings.Fields(title)) > 8 {
		return "", s, false
	}
	return title, prompt, true
}
