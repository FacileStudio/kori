package sandbox

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/FacileStudio/nacelle"
)

var skippedDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	"build": true, "target": true, ".svelte-kit": true, ".next": true,
	"__pycache__": true, ".venv": true, "coverage": true,
}

type listInput struct {
	Path string `json:"path,omitempty" jsonschema:"description=Directory to list relative to the working directory. Omit it for the working directory itself"`
}

type globInput struct {
	Pattern string `json:"pattern" jsonschema:"required,description=A glob such as **/*.go or cmd/**. ** matches any number of directories"`
}

type grepInput struct {
	Pattern string `json:"pattern" jsonschema:"required,description=A Go regular expression to search for"`
	Glob    string `json:"glob,omitempty" jsonschema:"description=Only search files matching this glob — for example **/*.go"`
}

func buildListTool(s *remoteSession) (nacelle.Tool, error) {
	return nacelle.NewToolWithOptions("list_directory",
		"List one directory: the files in it, and the subdirectories in it marked with a trailing slash. Use it to find your way around a tree you have not seen, before searching it. Generated directories such as .git, node_modules and vendor are left out, exactly as they are when searching.",
		func(ctx context.Context, in listInput) (string, error) {
			return runListDir(ctx, s, in)
		},
		nacelle.ToolOptions{ReadOnly: true})
}

func filterListing(raw string) []string {
	var names []string
	for line := range strings.SplitSeq(strings.TrimSpace(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "./" || trimmed == "../" {
			continue
		}
		if base, ok := strings.CutSuffix(trimmed, "/"); ok {
			if skippedDirs[base] || strings.HasPrefix(base, ".") {
				continue
			}
			names = append(names, base+"/")
			continue
		}
		names = append(names, trimmed)
	}
	sort.Strings(names)
	return names
}

func runListDir(ctx context.Context, s *remoteSession, in listInput) (string, error) {
	targetDir := resolveRemotePath(s.opts.WorkDir, in.Path)
	remoteCmd := fmt.Sprintf("cd %s && ls -1pa", quoteArg(targetDir))
	out, err := s.runSSH(ctx, remoteCmd)
	if err != nil {
		return "", fmt.Errorf("list_directory: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	listed := filterListing(string(out))
	if len(listed) == 0 {
		return fmt.Sprintf("nothing to list in %s", in.Path), nil
	}
	return truncateText(strings.Join(listed, "\n"), s.opts.MaxOutputBytes), nil
}

func buildFindTool(s *remoteSession) (nacelle.Tool, error) {
	return nacelle.NewToolWithOptions("find_files",
		"List files matching a glob, relative to the working directory. Use it to learn what exists before reading anything. Generated directories such as .git, node_modules and vendor are skipped.",
		func(ctx context.Context, in globInput) (string, error) {
			return runFindFiles(ctx, s, in)
		},
		nacelle.ToolOptions{ReadOnly: true})
}

func runFindFiles(ctx context.Context, s *remoteSession, in globInput) (string, error) {
	pattern := strings.TrimSpace(in.Pattern)
	if pattern == "" {
		return "", fmt.Errorf("no pattern given")
	}
	remoteCmd := fmt.Sprintf("cd %s && find . -name . -o -type d \\( -name '.*' -o -name node_modules -o -name vendor -o -name dist -o -name build -o -name target -o -name .svelte-kit -o -name .next -o -name __pycache__ -o -name .venv -o -name coverage \\) -prune -o -type f -print", quoteArg(s.opts.WorkDir))
	out, err := s.runSSH(ctx, remoteCmd)
	if err != nil {
		return "", fmt.Errorf("find_files: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	var found []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		clean := strings.TrimPrefix(strings.TrimSpace(line), "./")
		if clean != "" && (matchGlob(pattern, clean) || matchGlob(pattern, "./"+clean)) {
			found = append(found, clean)
		}
	}
	if len(found) == 0 {
		return fmt.Sprintf("no files match %s", in.Pattern), nil
	}
	sort.Strings(found)
	return truncateText(strings.Join(found, "\n"), s.opts.MaxOutputBytes), nil
}

func buildSearchTool(s *remoteSession) (nacelle.Tool, error) {
	return nacelle.NewToolWithOptions("search_content",
		"Search file contents with a regular expression, returning matching lines with their file and line number. Narrow it with a glob when you know the file type. Use this to find where something is defined or used, rather than reading files one at a time.",
		func(ctx context.Context, in grepInput) (string, error) {
			return runSearchContent(ctx, s, in)
		},
		nacelle.ToolOptions{ReadOnly: true})
}

func formatGrepMatches(raw, glob string) []string {
	var matches []string
	for line := range strings.SplitSeq(strings.TrimSpace(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 3)
		if len(parts) < 3 {
			continue
		}
		file := strings.TrimPrefix(parts[0], "./")
		if glob != "" && !matchGlob(glob, file) {
			continue
		}
		matches = append(matches, fmt.Sprintf("%s:%s: %s", file, parts[1], strings.TrimSpace(parts[2])))
	}
	return matches
}

func runSearchContent(ctx context.Context, s *remoteSession, in grepInput) (string, error) {
	if _, err := regexp.Compile(in.Pattern); err != nil {
		return "", fmt.Errorf("that is not a valid regular expression: %w", err)
	}
	remoteCmd := fmt.Sprintf("cd %s && find . -name . -o -type d \\( -name '.*' -o -name node_modules -o -name vendor -o -name dist -o -name build -o -name target -o -name .svelte-kit -o -name .next -o -name __pycache__ -o -name .venv -o -name coverage \\) -prune -o -type f -exec grep -I -s -n -E %s /dev/null {} +", quoteArg(s.opts.WorkDir), quoteArg(in.Pattern))
	out, _ := s.runSSH(ctx, remoteCmd)
	matches := formatGrepMatches(string(out), strings.TrimSpace(in.Glob))
	if len(matches) == 0 {
		return fmt.Sprintf("no matches for %s", in.Pattern), nil
	}
	return truncateText(strings.Join(matches, "\n"), s.opts.MaxOutputBytes), nil
}
