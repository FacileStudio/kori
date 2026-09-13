package agent

import (
	"context"

	"github.com/FacileStudio/nacelle"
)

// runStats is what one headless run measured: the run's total cost and token
// counts from KindDone, the tool calls made along the way, the final
// conversation size from the last KindTurn, and how many compaction passes
// fired.
type runStats struct {
	nacelle.Usage
	ToolCalls          int
	FinalContextTokens int64
	Compactions        int
}

// compactHook returns the AfterCompact hook that counts compaction passes on
// this run. It is wired as an extra hook in buildHeadlessAgent, next to the
// settings hooks.
func (s *runStats) compactHook() map[nacelle.HookPoint][]nacelle.Hook {
	return map[nacelle.HookPoint][]nacelle.Hook{
		nacelle.AfterCompact: {func(context.Context, nacelle.HookEvent) nacelle.HookResult {
			s.Compactions++
			return nacelle.HookResult{}
		}},
	}
}
