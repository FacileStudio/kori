package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

type echoTool struct{ name, description string }

func (t echoTool) Name() string        { return t.name }
func (t echoTool) Description() string { return t.description }
func (t echoTool) Schema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"echo": map[string]any{"type": "string"},
	}}
}

func (t echoTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	return fmt.Sprintf("%s heard %s", t.name, input), nil
}

func catalogOf(t *testing.T, count int) *catalog {
	t.Helper()
	tools := make([]nacelle.Tool, 0, count)
	for i := range count {
		tools = append(tools, echoTool{name: fmt.Sprintf("srv_tool_%02d", i), description: "does thing " + fmt.Sprint(i)})
	}
	return &catalog{tools: tools}
}

func TestSearchToolsListsEverythingOnAnEmptyQuery(t *testing.T) {
	got, err := searchToolsTool{catalog: catalogOf(t, 3)}.Run(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for i := range 3 {
		if !strings.Contains(got, fmt.Sprintf("srv_tool_%02d", i)) {
			t.Errorf("listing = %q, want every tool named", got)
		}
	}
}

func TestSearchToolsFiltersAndMisses(t *testing.T) {
	c := catalogOf(t, 3)
	got, _ := searchToolsTool{catalog: c}.Run(context.Background(), []byte(`{"query":"thing 1"}`))
	if !strings.Contains(got, "srv_tool_01") || strings.Contains(got, "srv_tool_00") {
		t.Errorf("listing = %q, want only the match", got)
	}
	missed, _ := searchToolsTool{catalog: c}.Run(context.Background(), []byte(`{"query":"nonexistent"}`))
	if !strings.Contains(missed, "No bridged tool matches") {
		t.Errorf("listing = %q, want the miss said so", missed)
	}
}

func TestToolDetailsReturnsSchemaAndMisses(t *testing.T) {
	got, _ := toolDetailsTool{catalog: catalogOf(t, 1)}.Run(context.Background(), []byte(`{"name":"srv_tool_00"}`))
	if !strings.Contains(got, "does thing 0") || !strings.Contains(got, `"echo"`) {
		t.Errorf("details = %q, want description and schema", got)
	}
	missed, _ := toolDetailsTool{catalog: catalogOf(t, 1)}.Run(context.Background(), []byte(`{"name":"elsewhere"}`))
	if !strings.Contains(missed, "No bridged tool named elsewhere") {
		t.Errorf("details = %q, want the miss said so", missed)
	}
}

func TestCallToolRunsTheNamedBridgedTool(t *testing.T) {
	got, err := callToolTool{catalog: catalogOf(t, 1)}.Run(context.Background(), []byte(`{"name":"srv_tool_00","arguments":{"echo":"hi"}}`))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(got, "srv_tool_00 heard") || !strings.Contains(got, "hi") {
		t.Errorf("output = %q, want the bridged tool's own output", got)
	}
	_, err = callToolTool{catalog: catalogOf(t, 1)}.Run(context.Background(), []byte(`{"name":"elsewhere"}`))
	if err == nil || !strings.Contains(err.Error(), "search_tools") {
		t.Errorf("error = %v, want a refusal naming the way back", err)
	}
}

func TestAGrownBridgedSetIsMountedAsACatalog(t *testing.T) {
	bridged := make([]nacelle.Tool, 0, mcpCatalogThreshold+1)
	for i := range mcpCatalogThreshold + 1 {
		bridged = append(bridged, echoTool{name: fmt.Sprintf("tool_%02d", i), description: "d"})
	}
	mounted := grown(nil, bridged)
	if len(mounted) != 3 {
		t.Errorf("mounted %d tools, want the 3 catalog tools", len(mounted))
	}
	for _, want := range []string{"search_tools", "get_tool_details", "call_tool"} {
		found := false
		for _, tool := range mounted {
			if tool.Name() == want {
				found = true
			}
		}
		if !found {
			t.Errorf("mounted set has no %s", want)
		}
	}
}

func TestASmallBridgedSetIsMountedWhole(t *testing.T) {
	bridged := []nacelle.Tool{echoTool{name: "tool_00", description: "d"}}
	mounted := grown([]nacelle.Tool{}, bridged)
	if len(mounted) != 1 || mounted[0].Name() != "tool_00" {
		t.Errorf("mounted = %+v, want the bridged tool itself", mounted)
	}
}

func TestCallToolNormalizesStringifiedArguments(t *testing.T) {
	c := catalogOf(t, 1)
	got, err := callToolTool{catalog: c}.Run(context.Background(), []byte(`{"name":"srv_tool_00","arguments":"{\"echo\":\"stringified\"}"}`))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(got, "stringified") {
		t.Errorf("got = %q, want stringified arguments parsed", got)
	}
}

func TestCallToolNormalizesEmptyAndNullArguments(t *testing.T) {
	c := catalogOf(t, 1)
	got, err := callToolTool{catalog: c}.Run(context.Background(), []byte(`{"name":"srv_tool_00"}`))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(got, "heard {}") {
		t.Errorf("got = %q, want heard {}", got)
	}

	got, err = callToolTool{catalog: c}.Run(context.Background(), []byte(`{"name":"srv_tool_00","arguments":null}`))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(got, "heard {}") {
		t.Errorf("got = %q, want heard {}", got)
	}
}

func TestCatalogFindSuffixMatching(t *testing.T) {
	c := &catalog{tools: []nacelle.Tool{
		echoTool{name: "cercle_cercle_inbox_read", description: "read inbox"},
	}}
	if tool := c.find("cercle_inbox_read"); tool == nil || tool.Name() != "cercle_cercle_inbox_read" {
		t.Errorf("find(cercle_inbox_read) = %v, want cercle_cercle_inbox_read", tool)
	}
	if tool := c.find("inbox_read"); tool == nil || tool.Name() != "cercle_cercle_inbox_read" {
		t.Errorf("find(inbox_read) = %v, want cercle_cercle_inbox_read", tool)
	}
}
