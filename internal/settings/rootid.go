package settings

import "strings"

// targetRootPrefix marks a session root that names a remote target rather
// than a directory on this machine. Remote and sandbox sessions run their
// tools over SSH on the target; the process on this host never opens that
// path. When a target defines no workdir the session runs in the SSH login
// directory, and recording the host's own working directory as the session
// root was the 0.69 regression: the banner, the system prompt and the
// session log all claimed a host path the session never touched, and
// auto-resume grouped a remote session with whatever local project the
// command happened to be launched from.
//
// The tag flows only through display and bookkeeping — the banner, the
// session block of the system prompt, the session log header and the
// project list — each of which special-cases it here. The SSH tools take
// their working directory from SessionOptions.WorkDir and never from
// config.Root.
const targetRootPrefix = "target:"

// TargetRoot returns the session root that labels a target session with no
// configured workdir: "target:<name>". An empty name yields "" so a caller
// without a target gets a plain local root back.
func TargetRoot(name string) string {
	if name == "" {
		return ""
	}
	return targetRootPrefix + name
}

// TargetRootName reports the target name behind a target-labeled root, or
// "" when the root is an ordinary path.
func TargetRootName(root string) string {
	if !IsTargetRoot(root) {
		return ""
	}
	return strings.TrimPrefix(root, targetRootPrefix)
}

// IsTargetRoot reports whether root labels a remote target rather than a
// host path. Callers that would resolve or open root must pass it through
// untouched when this is true.
func IsTargetRoot(root string) bool {
	return strings.HasPrefix(root, targetRootPrefix)
}
