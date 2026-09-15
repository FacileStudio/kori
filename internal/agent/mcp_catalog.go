package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/FacileStudio/nacelle"
)

// mcpCatalogThreshold is the bridged-tool count above which the full catalog
// stops being mounted and the three catalog tools stand in for it. Past that,
// every definition resident at once costs context and degrades selection —
// the corpus put the crossover somewhere between twenty and thirty.
const mcpCatalogThreshold = 50

// catalog stands in for the full bridged tool set when it is too big to mount
// whole. It holds the real tools and exposes three: search_tools to find
// candidates, get_tool_details to read one schema, call_tool to execute one.
// It holds nothing a call can change, so it is safe under concurrent Run, as
// nacelle.Tool requires.
type catalog struct {
	tools []nacelle.Tool
}

func (c *catalog) find(name string) nacelle.Tool {
	for _, t := range c.tools {
		if t.Name() == name {
			return t
		}
	}
	for _, t := range c.tools {
		if strings.HasSuffix(t.Name(), "_"+name) {
			return t
		}
	}
	return nil
}

// search filters on name and description, case-insensitively, and returns the
// full list for an empty query — a model that does not know what a server
// carries is better served by everything than by nothing.
func (c *catalog) search(query string) []nacelle.Tool {
	query = strings.ToLower(query)
	var hits []nacelle.Tool
	for _, t := range c.tools {
		if query == "" || strings.Contains(strings.ToLower(t.Name()+" "+t.Description()), query) {
			hits = append(hits, t)
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Name() < hits[j].Name() })
	return hits
}

type searchToolsTool struct{ catalog *catalog }

func (searchToolsTool) Name() string { return "search_tools" }

func (searchToolsTool) Description() string {
	return "Search the MCP-bridged tools by name or description. Call this before assuming a bridged " +
		"tool exists; an empty query lists every one. Returns name and one-line description pairs, " +
		"not input schemas — get_tool_details reads those."
}

func (searchToolsTool) Schema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"query": map[string]any{"type": "string", "description": "Words from the job: name fragments or capabilities. Empty or omitted lists everything."},
	}}
}

func (t searchToolsTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("reading query: %w", err)
	}
	hits := t.catalog.search(in.Query)
	if len(hits) == 0 {
		return fmt.Sprintf("No bridged tool matches %q. Fewer or broader words find more.", in.Query), nil
	}
	var body strings.Builder
	for _, hit := range hits {
		fmt.Fprintf(&body, "%s — %s\n", hit.Name(), hit.Description())
	}
	return body.String(), nil
}

// newCatalogTools mounts the three catalog tools over one bridged set.
func newCatalogTools(bridged []nacelle.Tool) []nacelle.Tool {
	c := &catalog{tools: bridged}
	return []nacelle.Tool{searchToolsTool{c}, toolDetailsTool{c}, callToolTool{c}}
}
