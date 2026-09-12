package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withGates(t *testing.T, run func(ctx context.Context, g Gate, scope string) gateOutput) {
	t.Helper()
	previous := runGate
	runGate = run
	t.Cleanup(func() { runGate = previous })
}

var okRun = func(_ context.Context, _ Gate, _ string) gateOutput {
	return gateOutput{code: 0}
}

func TestChainFormatGateReportsTheReformat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.go")
	os.WriteFile(path, []byte("package a\n"), 0o600)
	withGates(t, func(ctx context.Context, g Gate, scope string) gateOutput {
		if g.Name == "fmt" {
			os.WriteFile(path, []byte("package a\n\n"), 0o600)
		}
		return gateOutput{code: 0}
	})
	chain := NewChain([]Gate{
		{Name: "fmt", Cmd: []string{"fmt"}, Format: true},
		{Name: "lint", Cmd: []string{"lint"}},
	})
	got := chain.InjectFile(context.Background(), path)
	want := fmt.Sprintf("fmt: reformatted %s — re-read it before further edits", path)
	if got != want {
		t.Fatalf("InjectFile = %q, want %q", got, want)
	}
}

func TestChainUnchangedFormatGateFallsThrough(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.go")
	os.WriteFile(path, []byte("package a\n"), 0o600)
	withGates(t, func(ctx context.Context, g Gate, scope string) gateOutput {
		return gateOutput{code: 1, stdout: "a.go:1:1: error: bad [gen.test]"}
	})
	chain := NewChain([]Gate{{Name: "fmt", Cmd: []string{"fmt"}, Format: true}})
	got := chain.InjectFile(context.Background(), path)
	if !strings.Contains(got, "a.go:1:1: bad") {
		t.Fatalf("InjectFile = %q, want the finding line", got)
	}
}

func TestChainInjectFileStopsAtFirstFinding(t *testing.T) {
	var seen []string
	withGates(t, func(ctx context.Context, g Gate, scope string) gateOutput {
		seen = append(seen, g.Name+"@"+scope)
		return gateOutput{code: 1, stdout: "a.go:1:1: error: bad [gen.test]"}
	})
	chain := NewChain([]Gate{
		{Name: "filet", Cmd: []string{"filet", "check"}},
		{Name: "tests", Cmd: []string{"go", "test"}},
	})
	got := chain.InjectFile(context.Background(), "a.go")
	if !strings.Contains(got, "a.go:1:1: bad") {
		t.Fatalf("InjectFile = %q, want the finding line", got)
	}
	if len(seen) != 1 || seen[0] != "filet@a.go" {
		t.Fatalf("ran %v, want only the first gate on a.go", seen)
	}
}

func TestChainInjectFileRunsCleanThroughEveryGate(t *testing.T) {
	var seen []string
	withGates(t, func(ctx context.Context, g Gate, scope string) gateOutput {
		seen = append(seen, g.Name)
		return gateOutput{code: 0}
	})
	chain := NewChain([]Gate{
		{Name: "filet", Cmd: []string{"filet", "check"}},
		{Name: "tests", Cmd: []string{"go", "test"}},
	})
	if got := chain.InjectFile(context.Background(), "a.go"); got != cleanLine {
		t.Fatalf("InjectFile = %q, want the clean line", got)
	}
	if len(seen) != 2 {
		t.Fatalf("ran %v, want both gates", seen)
	}
}

func TestChainRunStopsAtFirstFailure(t *testing.T) {
	var seen []string
	withGates(t, func(ctx context.Context, g Gate, scope string) gateOutput {
		seen = append(seen, g.Name)
		if g.Name == "lint" {
			return gateOutput{code: 1, stdout: "not a finding line, just raw output"}
		}
		return gateOutput{code: 0}
	})
	chain := NewChain([]Gate{
		{Name: "lint", Cmd: []string{"lint"}},
		{Name: "tests", Cmd: []string{"go", "test"}},
	})
	got, err := chain.Run(context.Background(), "a.go", false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(got, "not a finding line") {
		t.Fatalf("Run = %q, want raw passthrough of the lint output", got)
	}
	if len(seen) != 1 {
		t.Fatalf("ran %v, want the chain to stop after the first failure", seen)
	}
}

func TestChainRepoRunPassesTheRoot(t *testing.T) {
	var scope string
	withGates(t, func(ctx context.Context, g Gate, s string) gateOutput {
		scope = s
		return gateOutput{code: 0}
	})
	chain := NewChain([]Gate{{Name: "lint", Cmd: []string{"lint"}}})
	if _, err := chain.Run(context.Background(), "a.go", true); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if scope != "." {
		t.Fatalf("scope = %q, want the session root", scope)
	}
}

func TestChainNoGatesIsNilAndLegacyPathRuns(t *testing.T) {
	if NewChain(nil) == nil {
		t.Fatal("NewChain(nil) returned a nil interface holding a nil value")
	}
	previous := runFilet
	runFilet = func(ctx context.Context, scope string) gateOutput {
		return gateToRun(okRun(context.Background(), Gate{}, scope))
	}
	t.Cleanup(func() { runFilet = previous })
	got, err := Run(context.Background(), "a.go", false)
	if err != nil || got != cleanLine {
		t.Fatalf("legacy Run = %q, %v; want clean", got, err)
	}
}

func TestChainGateTimeoutReturnsTimedOut(t *testing.T) {
	withGates(t, func(ctx context.Context, g Gate, scope string) gateOutput {
		return gateOutput{code: -1, err: context.DeadlineExceeded}
	})
	chain := NewChain([]Gate{{Name: "slow", Cmd: []string{"slow"}}})
	if _, err := chain.Run(context.Background(), "a.go", false); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run err = %v, want a timeout error", err)
	}
}

func gateToRun(out gateOutput) gateOutput {
	return gateOutput{stdout: out.stdout, stderr: out.stderr, code: out.code, err: out.err}
}
