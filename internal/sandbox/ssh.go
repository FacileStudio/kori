package sandbox

import (
	"fmt"
	"strconv"
)

// resolveTargetHostPort returns the host and port for a target, defaulting to
// loopback and the boite SSH port for a boite VM that has none.
func resolveTargetHostPort(target *Target) (string, int) {
	host := "127.0.0.1"
	port := 22
	if target == nil {
		return host, port
	}
	if target.Host != "" {
		host = target.Host
	}
	if target.Port > 0 {
		port = target.Port
	} else if target.Backend == "boite" {
		port = 2226
	}
	return host, port
}

// workdirPrefix returns the cd prefix a remote command needs, or nothing when
// no workspace is configured: the command then runs in the SSH login directory.
func workdirPrefix(workDir string) string {
	if workDir == "" {
		return ""
	}
	return "cd " + quoteArg(workDir) + " && "
}

func resolveSSHUserDestination(target *Target, user, host string) string {
	u := user
	if u == "" && target != nil {
		u = target.User
	}
	if u != "" {
		return fmt.Sprintf("%s@%s", u, host)
	}
	return host
}

// buildRemoteSSHArgs builds the non-interactive ssh invocation every tool and
// probe shares. It disables agent forwarding and prompting everywhere, and
// turns on connection multiplexing when controlPath is set so the handshake is
// paid once per session instead of once per tool call.
//
// Host-key verification is relaxed for boite targets only: the microVM's key is
// generated per instance and never in known_hosts, and the connection is
// loopback. A remote host is a machine on a network, so it verifies against the
// user's own known_hosts and ssh_config exactly as a hand-typed ssh would; with
// BatchMode on, an unknown host is refused rather than silently trusted.
//
// A `--` ends option parsing before the destination, so a configured host value
// that happens to start with `-` is a hostname to ssh and not an option it
// would otherwise run.
func buildRemoteSSHArgs(target *Target, user, remoteCmd, controlPath string) []string {
	host, port := resolveTargetHostPort(target)
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ForwardAgent=no",
		"-o", "LogLevel=ERROR",
	}
	if target != nil && target.Backend == "boite" {
		args = append(args,
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
		)
	}
	if controlPath != "" {
		args = append(args,
			"-o", "ControlMaster=auto",
			"-o", "ControlPath="+controlPath,
			"-o", "ControlPersist=60s",
		)
	}
	args = append(args, "-p", strconv.Itoa(port))
	if target != nil && target.KeyPath != "" {
		args = append(args, "-i", target.KeyPath)
	}
	args = append(args, "--", resolveSSHUserDestination(target, user, host))
	if remoteCmd != "" {
		args = append(args, remoteCmd)
	}
	return args
}
