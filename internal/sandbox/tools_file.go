package sandbox

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/FacileStudio/nacelle"
)

type readInput struct {
	Path   string `json:"path" jsonschema:"required,description=Path to the file relative to the working directory"`
	Offset int    `json:"offset,omitempty" jsonschema:"description=First line to show. Counting starts at 1. Omit to start at the beginning"`
	Limit  int    `json:"limit,omitempty" jsonschema:"description=How many lines to show. Omit for as many as fit"`
}

type writeInput struct {
	Path    string `json:"path" jsonschema:"required,description=Path to the file relative to the working directory"`
	Content string `json:"content" jsonschema:"required,description=The complete new contents of the file"`
}

func buildReadTool(s *remoteSession) (nacelle.Tool, error) {
	return nacelle.NewToolWithOptions("read_file",
		"Read a file from the working directory. Returns the contents with line numbers, so you can quote a line back exactly. Use it before editing anything.",
		func(ctx context.Context, in readInput) (string, error) {
			return runReadFile(ctx, s, in)
		},
		nacelle.ToolOptions{ReadOnly: true})
}

func formatNumberedLines(content string, offset, limit, maxBytes int) string {
	lines := strings.Split(content, "\n")
	start := offset
	if start > 0 {
		if start > len(lines) {
			return fmt.Sprintf("[the file has %d lines; line %d is past the end]", len(lines), start)
		}
		lines = lines[start-1:]
	} else {
		start = 1
	}
	if limit > 0 && limit < len(lines) {
		lines = lines[:limit]
	}
	var out strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&out, "%6d\t%s\n", start+i, line)
	}
	return truncateText(out.String(), maxBytes)
}

func runReadFile(ctx context.Context, s *remoteSession, in readInput) (string, error) {
	targetPath := resolveRemotePath(s.opts.WorkDir, in.Path)
	remoteCmd := fmt.Sprintf("cat -- %s", quoteArg(targetPath))
	out, err := s.runSSH(ctx, remoteCmd)
	if err != nil {
		return "", fmt.Errorf("read_file: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	if !utf8.Valid(out) {
		return "", fmt.Errorf("%s is not text", in.Path)
	}
	return formatNumberedLines(string(out), in.Offset, in.Limit, s.opts.MaxReadBytes), nil
}

func buildWriteTool(s *remoteSession) (nacelle.Tool, error) {
	return nacelle.NewTool("write_file",
		"Create a file, or replace one entirely. The content you give is the whole file, not a fragment. To change part of an existing file use edit_file instead, which cannot silently discard the rest.",
		func(ctx context.Context, in writeInput) (string, error) {
			return runWriteFile(ctx, s, in)
		})
}

func runWriteFile(ctx context.Context, s *remoteSession, in writeInput) (string, error) {
	targetPath := resolveRemotePath(s.opts.WorkDir, in.Path)
	parentDir := path.Dir(targetPath)
	encoded := base64.StdEncoding.EncodeToString([]byte(in.Content))
	remoteCmd := fmt.Sprintf("mkdir -p %s && printf '%%s' %s | base64 -d > %s", quoteArg(parentDir), quoteArg(encoded), quoteArg(targetPath))
	out, err := s.runSSH(ctx, remoteCmd)
	if err != nil {
		return "", fmt.Errorf("write_file: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return fmt.Sprintf("wrote %s (%d bytes)", in.Path, len(in.Content)), nil
}
