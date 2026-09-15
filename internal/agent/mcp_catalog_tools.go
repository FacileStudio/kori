package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

type toolDetailsTool struct{ catalog *catalog }

func (toolDetailsTool) Name() string { return "get_tool_details" }

func (toolDetailsTool) Description() string {
	return "Return one bridged MCP tool's full description and input schema. Call search_tools first, " +
		"then this before the first call to any tool whose schema you have not read. Returns the " +
		"schema JSON, not an example call."
}

func (toolDetailsTool) Schema() map[string]any {
	return map[string]any{"type": "object", "required": []string{"name"}, "properties": map[string]any{
		"name": map[string]any{"type": "string", "description": "The tool's exact name, as search_tools returned it."},
	}}
}

func (t toolDetailsTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("reading name: %w", err)
	}
	tool := t.catalog.find(in.Name)
	if tool == nil {
		return fmt.Sprintf("No bridged tool named %s. search_tools finds what exists.", in.Name), nil
	}
	schema, err := json.Marshal(tool.Schema())
	if err != nil {
		schema = []byte("{}")
	}
	return fmt.Sprintf("%s\n\n%s\n\nInput schema:\n%s", tool.Name(), tool.Description(), schema), nil
}

type callToolTool struct{ catalog *catalog }

func (callToolTool) Name() string { return "call_tool" }

func (callToolTool) Description() string {
	return "Run a bridged MCP tool by name with a JSON arguments object. The name must be one " +
		"search_tools surfaced; arguments must match the schema get_tool_details returned. Returns " +
		"the tool's own text output, not a summary — pass it on, trimmed, when it answers the user."
}

func (callToolTool) Schema() map[string]any {
	return map[string]any{"type": "object", "required": []string{"name"}, "properties": map[string]any{
		"name":      map[string]any{"type": "string", "description": "The tool's exact name, as search_tools returned it."},
		"arguments": map[string]any{"type": "object", "description": "The tool's input, matching the schema get_tool_details returned. Omitted when the schema has no properties."},
	}}
}

func (t callToolTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("reading name and arguments: %w", err)
	}
	tool := t.catalog.find(in.Name)
	if tool == nil {
		return "", fmt.Errorf("no bridged tool named %s; run search_tools first", in.Name)
	}
	return tool.Run(ctx, normalizeCallToolArgs(in.Arguments))
}
