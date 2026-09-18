package sandbox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestEditFileSuccess(t *testing.T) {
	step := 0
	var writeCmd string
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			step++
			if step == 1 {
				return []byte("line 1\nhello world\nline 3\n"), nil
			}
			writeCmd = args[len(args)-1]
			return nil, nil
		},
	}
	tools, _, _ := RemoteTools(ToolsOptions{Runner: runner})
	editTool := tools[3]
	out, err := editTool.Run(context.Background(), json.RawMessage(`{"path":"doc.txt","old":"hello world","new":"hello universe"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "edited doc.txt" {
		t.Fatalf("unexpected output: %s", out)
	}
	expectedB64 := base64.StdEncoding.EncodeToString([]byte("line 1\nhello universe\nline 3\n"))
	if !strings.Contains(writeCmd, expectedB64) {
		t.Fatalf("write command did not contain expected base64 content: %s", writeCmd)
	}
}

func TestEditFileErrors(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("one\ntwo\ntwo\nthree\n"), nil
		},
	}
	tools, _, _ := RemoteTools(ToolsOptions{Runner: runner})
	editTool := tools[3]
	if _, err := editTool.Run(context.Background(), json.RawMessage(`{"path":"f.txt","old":"missing","new":"x"}`)); err == nil {
		t.Fatal("expected error for non-existent match")
	}
	if _, err := editTool.Run(context.Background(), json.RawMessage(`{"path":"f.txt","old":"two","new":"x"}`)); err == nil {
		t.Fatal("expected error for multiple matches")
	}
	if _, err := editTool.Run(context.Background(), json.RawMessage(`{"path":"f.txt","old":"","new":"x"}`)); err == nil {
		t.Fatal("expected error for empty old text")
	}
	if _, err := editTool.Run(context.Background(), json.RawMessage(`{"path":"f.txt","old":"one","new":"one"}`)); err == nil {
		t.Fatal("expected error when replacement is identical")
	}
}

func TestListDirectorySuccess(t *testing.T) {
	raw := "./\n../\n.git/\nnode_modules/\nsrc/\nREADME.md\n.env\n"
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(raw), nil
		},
	}
	tools, _, _ := RemoteTools(ToolsOptions{Runner: runner})
	listTool := tools[4]
	out, err := listTool.Run(context.Background(), json.RawMessage(`{"path":""}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, ".git/") || strings.Contains(out, "node_modules/") {
		t.Fatalf("expected skipped dirs to be omitted: %s", out)
	}
	if !strings.Contains(out, "src/") || !strings.Contains(out, "README.md") || !strings.Contains(out, ".env") {
		t.Fatalf("expected entries missing from: %s", out)
	}
}

func TestListDirectoryEmpty(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("./\n../\n"), nil
		},
	}
	tools, _, _ := RemoteTools(ToolsOptions{Runner: runner})
	listTool := tools[4]
	out, err := listTool.Run(context.Background(), json.RawMessage(`{"path":"empty"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "nothing to list in empty") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestFindFilesSuccess(t *testing.T) {
	raw := "./main.go\n./internal/app.go\n./internal/app_test.go\n./README.md\n"
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(raw), nil
		},
	}
	tools, _, _ := RemoteTools(ToolsOptions{Runner: runner})
	findTool := tools[5]
	out, err := findTool.Run(context.Background(), json.RawMessage(`{"pattern":"**/*.go"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "main.go") || !strings.Contains(out, "internal/app.go") {
		t.Fatalf("expected go files in output: %s", out)
	}
	if strings.Contains(out, "README.md") {
		t.Fatalf("did not expect markdown in go glob: %s", out)
	}
}

func TestFindFilesNoMatch(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("./main.go\n"), nil
		},
	}
	tools, _, _ := RemoteTools(ToolsOptions{Runner: runner})
	findTool := tools[5]
	out, err := findTool.Run(context.Background(), json.RawMessage(`{"pattern":"*.rs"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "no files match *.rs") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestSearchContentSuccess(t *testing.T) {
	raw := "./main.go:12:func RunServer() {\n./lib/server.go:44:func RunServer() {\n./README.md:1:RunServer doc\n"
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(raw), nil
		},
	}
	tools, _, _ := RemoteTools(ToolsOptions{Runner: runner})
	searchTool := tools[6]
	out, err := searchTool.Run(context.Background(), json.RawMessage(`{"pattern":"RunServer","glob":"**/*.go"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "main.go:12: func RunServer() {") {
		t.Fatalf("expected formatted main.go match: %s", out)
	}
	if strings.Contains(out, "README.md") {
		t.Fatalf("expected README.md excluded by glob: %s", out)
	}
}

func TestSearchContentErrors(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, nil
		},
	}
	tools, _, _ := RemoteTools(ToolsOptions{Runner: runner})
	searchTool := tools[6]
	if _, err := searchTool.Run(context.Background(), json.RawMessage(`{"pattern":"[invalid"}`)); err == nil {
		t.Fatal("expected error for invalid regex")
	}
	out, err := searchTool.Run(context.Background(), json.RawMessage(`{"pattern":"NotFoundPattern"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "no matches for NotFoundPattern") {
		t.Fatalf("unexpected output: %s", out)
	}
}
