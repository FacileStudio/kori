package sandbox

import (
	"context"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/nacelle"
)

// ToolsOptions configures remote SSH-backed tool execution.
type ToolsOptions struct {
	Target         *Target
	WorkDir        string
	Runner         Runner
	CommandTimeout time.Duration
	MaxOutputBytes int
	MaxReadBytes   int
}

type remoteSession struct {
	opts ToolsOptions
}

type remoteCloser struct {
	cleanup func() error
}

func (c *remoteCloser) Close() error {
	if c.cleanup != nil {
		return c.cleanup()
	}
	return nil
}

func quoteArg(p string) string {
	if strings.ContainsAny(p, " \t\n\"'$`\\*?[]()~;&|<>") {
		return fmt.Sprintf("%q", p)
	}
	return p
}

func resolveRemotePath(workDir, p string) string {
	trimmed := strings.TrimSpace(p)
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "~") {
		return path.Clean(trimmed)
	}
	base := workDir
	if base == "" {
		base = "/workspace"
	}
	return path.Clean(path.Join(base, trimmed))
}

func buildRemoteSSHArgs(target *Target, user, remoteCmd string) []string {
	host, port := resolveTargetHostPort(target)
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ForwardAgent=no",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-p", strconv.Itoa(port),
	}
	if target != nil && target.KeyPath != "" {
		args = append(args, "-i", target.KeyPath)
	}
	args = append(args, resolveSSHUserDestination(target, user, host))
	if remoteCmd != "" {
		args = append(args, remoteCmd)
	}
	return args
}

func (s *remoteSession) runSSH(ctx context.Context, remoteCmd string) ([]byte, error) {
	runner := s.opts.Runner
	user := ""
	if s.opts.Target != nil {
		user = s.opts.Target.User
	}
	args := buildRemoteSSHArgs(s.opts.Target, user, remoteCmd)
	return runner.Run(ctx, "ssh", args...)
}

func newRemoteSession(opts ToolsOptions) *remoteSession {
	resolved := opts
	if resolved.WorkDir == "" {
		resolved.WorkDir = "/workspace"
	}
	if resolved.CommandTimeout == 0 {
		resolved.CommandTimeout = 2 * time.Minute
	}
	if resolved.MaxOutputBytes == 0 {
		resolved.MaxOutputBytes = 64 * 1024
	}
	if resolved.MaxReadBytes == 0 {
		resolved.MaxReadBytes = 48 * 1024
	}
	if resolved.Runner == nil {
		resolved.Runner = NewDefaultRunner()
	}
	return &remoteSession{opts: resolved}
}

// RemoteTools constructs the set of remote sandbox tools and an associated closer.
func RemoteTools(opts ToolsOptions) ([]nacelle.Tool, io.Closer, error) {
	s := newRemoteSession(opts)
	runCmd, err := buildCommandTool(s)
	if err != nil {
		return nil, nil, err
	}
	readFile, err := buildReadTool(s)
	if err != nil {
		return nil, nil, err
	}
	writeFile, err := buildWriteTool(s)
	if err != nil {
		return nil, nil, err
	}
	editFile, err := buildEditTool(s)
	if err != nil {
		return nil, nil, err
	}
	listDir, err := buildListTool(s)
	if err != nil {
		return nil, nil, err
	}
	findFiles, err := buildFindTool(s)
	if err != nil {
		return nil, nil, err
	}
	searchContent, err := buildSearchTool(s)
	if err != nil {
		return nil, nil, err
	}
	all := []nacelle.Tool{runCmd, readFile, writeFile, editFile, listDir, findFiles, searchContent}
	return all, &remoteCloser{}, nil
}
