// The gate chain: project-configured deterministic checks dispatched at two
// entry points. Every gate runs against the path at hand: the edited file on
// the post-edit injection path, the session root when the pull tool sweeps the
// tree. No configured chain keeps every caller on the built-in filet path.

package diagnostics

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync/atomic"
)

// gateChain is the surface an installed chain provides; a nil chain is the
// filet-only path every existing caller gets.
type gateChain interface {
	InjectFile(ctx context.Context, path string) string
	Run(ctx context.Context, path string, repo bool) (string, error)
}

var installed atomic.Value

// UseChain installs the chain the entry points consult; nil restores the
// filet-only path.
func UseChain(c gateChain) {
	installed.Store(chainHolder{c})
}

type chainHolder struct {
	chain gateChain
}

func current() gateChain {
	if v, ok := installed.Load().(chainHolder); ok {
		return v.chain
	}
	return nil
}

// NewChain builds a chain from configured gates. Gates run in order, first
// failure stops the run, and a gate whose output parses as compiler-style
// findings renders like filet's; anything else passes through raw, capped.
func NewChain(gates []Gate) *runner {
	return &runner{gates: gates}
}

type runner struct {
	gates []Gate
}

// InjectFile runs the chain's gates against one edited path and returns
// the text appended to the edit tool result. Gates that fail oddly stay
// silent so a broken gate never denies the edit it is reporting on. A
// format gate that rewrote the path answers with a re-read notice instead
// of its findings, so the model knows its copy of the file went stale.
func (r *runner) InjectFile(ctx context.Context, path string) string {
	for _, g := range r.gates {
		before := snapshot(path)
		out := gateOutcome(g, runGate(ctx, g, path))
		if g.Format && before != nil && !sameFile(before, path) {
			return fmt.Sprintf("%s: reformatted %s — re-read it before further edits", g.Name, path)
		}
		if text, done := chainText(g, out); done {
			return text
		}
	}
	return cleanLine
}

func snapshot(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}

func sameFile(before []byte, path string) bool {
	after, err := os.ReadFile(path)
	return err == nil && bytes.Equal(before, after)
}

// Run sweeps the chain against the scope — the one path, or the session root
// on a repo sweep — first failure stops.
func (r *runner) Run(ctx context.Context, path string, repo bool) (string, error) {
	scope := scopeOf(path, repo)
	for _, g := range r.gates {
		out := gateOutcome(g, runGate(ctx, g, scope))
		if text, done := chainText(g, out); done {
			if out.kind == kindTimedOut {
				return "", fmt.Errorf("%s: timed out after %s: %w", g.Name, g.timeout(), context.DeadlineExceeded)
			}
			return text, nil
		}
	}
	return cleanLine, nil
}

func chainText(g Gate, out outcome) (string, bool) {
	switch out.kind {
	case kindClean:
		return "", false
	case kindTimedOut:
		return fmt.Sprintf("%s: timed out after %s", g.Name, g.timeout()), true
	case kindFindings:
		if len(out.findings) > 0 {
			return render(out.findings), true
		}
		return renderRaw(g.Name, out.raw), true
	}
	return "", false
}
