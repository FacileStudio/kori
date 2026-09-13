package settings

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/FacileStudio/nacelle"
)

// point maps the YAML spelling of an event onto the library's.
var point = map[string]nacelle.HookPoint{
	string(nacelle.BeforeToolCall): nacelle.BeforeToolCall,
	string(nacelle.AfterToolCall):  nacelle.AfterToolCall,
	string(nacelle.BeforeCompact):  nacelle.BeforeCompact,
	string(nacelle.AfterCompact):   nacelle.AfterCompact,
	string(nacelle.SessionStart):   nacelle.SessionStart,
}

// HookPointOf returns the library HookPoint for a YAML event string,
// or BeforeToolCall if the event is not recognised.
func HookPointOf(event string) nacelle.HookPoint {
	if p, ok := point[event]; ok {
		return p
	}
	return nacelle.BeforeToolCall
}

// Validate refuses a spec that could only misbehave at 2am.
func (s HookSpec) Validate() error {
	if _, known := point[s.On]; !known {
		return fmt.Errorf("hook %q: unknown event %q, want before_tool_call, after_tool_call, before_compact, after_compact, or session_start", s.Run, s.On)
	}
	if strings.TrimSpace(s.Run) == "" {
		return fmt.Errorf("a hook on %s has no command to run", s.On)
	}
	if s.Timeout != "" {
		if _, err := time.ParseDuration(s.Timeout); err != nil {
			return fmt.Errorf("hook %q: timeout %q is not a duration: %w", s.Run, s.Timeout, err)
		}
	}
	return nil
}

// Duration is the spec's timeout with its default filled in.
func (s HookSpec) Duration() time.Duration {
	if s.Timeout == "" {
		return 5 * time.Second
	}
	d, _ := time.ParseDuration(s.Timeout)
	return d
}

// Matches reports whether a spec asked to hear about this tool.
func (s HookSpec) Matches(tool string) bool {
	if len(s.Match) == 0 {
		return true
	}
	return slices.Contains(s.Match, tool)
}
