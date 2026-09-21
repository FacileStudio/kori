package tui

import (
	"github.com/FacileStudio/kori/internal/compaction"
	"github.com/FacileStudio/nacelle"
)

// alignedEvictCut is the thin TUI delegate for the compaction package's
// boundary-safe cut. It stays a one-line wrapper so the existing callers and
// their tests keep their name, while the rule itself lives beside the zones and
// blocks it protects.
func alignedEvictCut(conv []nacelle.Message, want int) int {
	return compaction.AlignedCut(conv, want)
}
