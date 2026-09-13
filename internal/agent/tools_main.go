package agent

import (
	"fmt"
	"os"

	"github.com/FacileStudio/nacelle"
	"github.com/FacileStudio/nacelle/tools"

	"github.com/FacileStudio/nacelle-tui/internal/settings"
	"github.com/FacileStudio/nacelle-tui/internal/tasks"
)

// localTools opens the file and command tools and, when asked, adds the web
// one — the caller owns closing the returned Set.
//
// Every failure after tools.New succeeds closes the Set on the way out.
// Without that it is dropped on the floor: the caller's own defer only ever
// sees the nil this returns on an error, so the *os.Root behind it — a real
// descriptor — outlives the only reference to it. The process exits moments
// later today and nothing notices, which is the kind of leak that becomes
// real the first time something rebuilds a tool set without restarting.
//
// The handle is held in a plain local rather than in the returned one, and
// that is load-bearing rather than style. A named result is assigned by the
// return statement *before* any deferred function runs, so with the Set named
// there, `return nil, nil, err` had already overwritten it — the deferred
// close then panicked on a nil receiver instead of releasing anything. The
// error paths still hand the caller nil, which is what it wants; only the
// closing needs a name the returns cannot reach.
func localTools(config Config) (_ *tools.Set, local []nacelle.Tool, err error) {
	opened, err := tools.New(tools.Config{
		Root:          config.Root,
		AllowBash:     *config.Bash,
		PathIsolation: *config.PathIsolation,
		DenyElevation: *config.DenyElevation,
		CommandEnv:    commandEnv(config),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("opening %s: %w", config.Root, err)
	}

	local, err = opened.Tools()
	if err != nil {
		opened.Close()
		return nil, nil, fmt.Errorf("building the tool set: %w", err)
	}

	var reaching []nacelle.Tool
	if reaching, err = webTools(config); err != nil {
		opened.Close()
		return nil, nil, err
	}
	local = append(local, reaching...)

	return opened, withPersistentShell(config, local), nil
}

// withPersistentShell swaps the stateless run_command for one persistent
// shell per session: same name, same schema, same result shape, only the
// execution backend changes. Strict confinement stays untouched — its
// escape checks exist to keep a command from moving out of the working
// directory, and a shell that remembers where it moved to is exactly what
// they forbid. The stateless tool stays on as the replacement's fallback,
// so a session whose persistent shell cannot start still runs commands the
// old way.
func withPersistentShell(config Config, local []nacelle.Tool) []nacelle.Tool {
	if !*config.Bash || settings.DerefBool(config.PathIsolation) {
		return local
	}
	at := -1
	for i, tool := range local {
		if tool.Name() == "run_command" {
			at = i
			break
		}
	}
	if at < 0 {
		return local
	}
	session := newShellSession(config.Root, commandEnv(config), settings.DerefBool(config.DenyElevation))
	replacement, err := newShellTool(session, local[at])
	if err != nil {
		return local
	}
	local[at] = replacement
	return local
}

// commandEnv is what run_command hands its children. With security
// .env_isolation off — the default — the process environment is passed
// whole, so a command sees the PATH, toolchains and exports of the shell
// nacelle-tui was launched from. With it on, CommandEnv stays nil and the
// tools package's own minimal base (PATH, HOME) applies, the same guarded
// posture the MCP servers get.
func commandEnv(config Config) []string {
	if settings.DerefBool(config.EnvIsolation) {
		return nil
	}
	return os.Environ()
}

// withTasks mounts the plan tool when the settings ask for one. It is
// deliberately not part of the set localTools builds. The SDK gives a nested
// run the parent's tools minus only the delegate tool itself, so anything
// localTools returns is inherited by a delegate whether that makes sense or
// not. The plan is drawn above the prompt on the one screen there is; a
// delegate calling this would replace what the parent wrote, and nothing
// afterwards puts it back.
//
// So it goes on after withParallelAgents has taken its copy. Same set, one line
// later, and the delegate never sees it.
func withTasks(config Config, local []nacelle.Tool) []nacelle.Tool {
	if !*config.Tasks {
		return local
	}
	return append(local, tasks.NewTool())
}
