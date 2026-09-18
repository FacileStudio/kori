package sandbox

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/FacileStudio/nacelle"
)

type editInput struct {
	Path string `json:"path" jsonschema:"required,description=Path to the file relative to the working directory"`
	Old  string `json:"old" jsonschema:"required,description=The exact text to replace. Include enough surrounding lines to make it unique in the file"`
	New  string `json:"new" jsonschema:"required,description=The text to put in its place"`
}

func buildEditTool(s *remoteSession) (nacelle.Tool, error) {
	return nacelle.NewTool("edit_file",
		"Replace an exact piece of text in a file. The old text must appear exactly once: include the lines around it until it does. Read the file first.",
		func(ctx context.Context, in editInput) (string, error) {
			return runEditFile(ctx, s, in)
		})
}

func squeezeWhitespace(text string) string {
	return string(bytes.Join(bytes.Fields([]byte(text)), []byte(" ")))
}

func nearMissText(content, oldText string) string {
	if strings.Contains(squeezeWhitespace(content), squeezeWhitespace(oldText)) {
		return " - the text is there but the whitespace differs; copy it exactly as read_file showed it, without the line-number column"
	}
	return ""
}

func replaceOnce(content, oldText, newText string) (string, error) {
	if oldText == "" {
		return "", fmt.Errorf("the text to replace is empty; use write_file to create a file")
	}
	if oldText == newText {
		return "", fmt.Errorf("the replacement is identical to the original")
	}
	count := strings.Count(content, oldText)
	if count == 1 {
		return strings.Replace(content, oldText, newText, 1), nil
	}
	if count == 0 {
		return "", fmt.Errorf("that exact text is not in the file%s", nearMissText(content, oldText))
	}
	return "", fmt.Errorf("that text appears %d times; include more surrounding lines so it matches only the one you mean", count)
}

func runEditFile(ctx context.Context, s *remoteSession, in editInput) (string, error) {
	targetPath := resolveRemotePath(s.opts.WorkDir, in.Path)
	readCmd := fmt.Sprintf("cat -- %s", quoteArg(targetPath))
	raw, err := s.runSSH(ctx, readCmd)
	if err != nil {
		return "", fmt.Errorf("edit_file: %w (%s)", err, strings.TrimSpace(string(raw)))
	}
	edited, err := replaceOnce(string(raw), in.Old, in.New)
	if err != nil {
		return "", fmt.Errorf("%s: %w", in.Path, err)
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(edited))
	writeCmd := fmt.Sprintf("printf '%%s' %s | base64 -d > %s", quoteArg(encoded), quoteArg(targetPath))
	out, err := s.runSSH(ctx, writeCmd)
	if err != nil {
		return "", fmt.Errorf("edit_file: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return fmt.Sprintf("edited %s", in.Path), nil
}
