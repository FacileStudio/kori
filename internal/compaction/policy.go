// Package compaction owns kori's context-management strategy: how a long
// conversation is partitioned into zones, which region is eligible for
// tombstoning or pruning, and how the surviving turns are reassembled around a
// persistent state ledger.
//
// It is pure. Nothing here owns a model, a clock or a backend, so the whole
// policy is table-driven from a test. The TypeSafe judge and the HTTP client
// that speaks to it live in their own packages; the TUI keeps only the wiring.
package compaction

import "github.com/FacileStudio/nacelle"

// The shipped tier ladder and the two ends a pass never touches. Settings alias
// these so the numbers cannot drift between the two layers.
const (
	DefaultSoftRatio      = 0.65
	DefaultMidRatio       = 0.80
	DefaultHardRatio      = 0.90
	DefaultKeepTurns      = 3
	DefaultAnchorMessages = 1
)

// Ratios is the tier ladder as fractions of the backend's context window.
// Soft tombstones history with no model call, mid adds a batched judge pass and
// a ledger summary, hard force-summarizes what is left.
type Ratios struct{ Soft, Mid, Hard float64 }

// Tier is the rung a measured conversation size has reached.
type Tier uint8

const (
	// Below is under every ratio with nothing to do.
	Below Tier = iota
	// Soft is the deterministic, model-free tier.
	Soft
	// Mid adds the judge and the ledger summary.
	Mid
	// Hard force-summarizes history and trims to the pinned ends.
	Hard
)

// String names a tier the way a status line or a report reads it.
func (t Tier) String() string {
	switch t {
	case Soft:
		return "soft"
	case Mid:
		return "mid"
	case Hard:
		return "hard"
	default:
		return "below"
	}
}

// Zone names the region one contiguous run of messages belongs to.
type Zone uint8

const (
	// ZoneAnchor is the pinned head: the first AnchorMessages messages, never
	// rewritten, summarized or pruned.
	ZoneAnchor Zone = iota
	// ZoneLedger is the single message carrying the state-ledger sentinel.
	ZoneLedger
	// ZoneHistory is everything between the ledger and the active window; the
	// only region eligible for tombstoning or pruning.
	ZoneHistory
	// ZoneActive is the newest KeepTurns messages, kept verbatim.
	ZoneActive
)

// Span is one contiguous run of a conversation: Zone says what it is, and
// Start and End are the half-open index range it covers.
type Span struct {
	Zone       Zone
	Start, End int
}

// Policy is everything one session needs to decide what to compact: the ratio
// ladder, the window it is measured against, the absolute ceiling that overrides
// it, and the two ends a pass never touches. Window is 0 when the backend reports
// none; Ceiling is the compact_at fallback for exactly that case.
type Policy struct {
	Ratios         Ratios
	Window         int64
	Ceiling        int64
	KeepTurns      int
	AnchorMessages int
}

// Tier is the rung size falls on. With a known window it is a ratio comparison,
// with compact_at flooring it at the soft tier when a session pins one. With no
// window the ratios are undefined, so the ceiling is the only trigger and it
// buys the full pass: a backend that reports no window has no soft ratio to
// measure a free tombstone against.
func (p Policy) Tier(size int64) Tier {
	if p.Window <= 0 {
		if p.Ceiling > 0 && size >= p.Ceiling {
			return Mid
		}
		return Below
	}
	switch {
	case reaches(size, p.Ratios.Hard, p.Window):
		return Hard
	case reaches(size, p.Ratios.Mid, p.Window):
		return Mid
	case reaches(size, p.Ratios.Soft, p.Window):
		return Soft
	case p.Ceiling > 0 && size >= p.Ceiling:
		return Soft
	default:
		return Below
	}
}

// Trigger is the size at which this policy asks for a pass: the compact_at
// ceiling when one is set, otherwise the soft ratio of the window. Zero means
// there is no threshold to cross.
func (p Policy) Trigger() int64 {
	if p.Ceiling > 0 {
		return p.Ceiling
	}
	if p.Window > 0 {
		return int64(p.Ratios.Soft * float64(p.Window))
	}
	return 0
}

// reaches compares against a rounded threshold rather than the raw product, so
// a ratio like 0.65 of 200000 lands exactly on 130000 instead of just above it.
func reaches(size int64, ratio float64, window int64) bool {
	if ratio <= 0 {
		return false
	}
	return size >= int64(ratio*float64(window)+0.5)
}

// Plan partitions the conversation by index into anchor, ledger, history and
// active spans, in that order. The active boundary is pulled back so it never
// opens on a ToolResult whose call the cut dropped, the pinned head is extended
// over the replies answering the calls it carries so it never splits a pair
// either, and a missing ledger is simply left out — a session has none until its
// first pass builds one. Spans are contiguous and cover the whole conversation;
// only the history spans are eligible for a pass.
func Plan(conv []nacelle.Message, p Policy) []Span {
	n := len(conv)
	anchor := anchorEnd(conv, clamp(p.AnchorMessages, 0, n))
	active := activeStart(conv, n, anchor, p.KeepTurns)
	ledger := ledgerIndex(conv, anchor, active)

	spans := make([]Span, 0, 5)
	if anchor > 0 {
		spans = append(spans, Span{Zone: ZoneAnchor, Start: 0, End: anchor})
	}
	start := anchor
	if ledger >= 0 {
		end := LedgerEnd(conv, ledger, active)
		spans = appendHistory(spans, start, ledger)
		spans = append(spans, Span{Zone: ZoneLedger, Start: ledger, End: end})
		start = end
	}
	spans = appendHistory(spans, start, active)
	if active < n {
		spans = append(spans, Span{Zone: ZoneActive, Start: active, End: n})
	}
	return spans
}

// anchorEnd extends the pinned head over the replies answering the tool calls the
// head itself carries, the same way a ledger claims its own. Without it a
// conversation whose head is an assistant ToolCall leaves the matching ToolResult
// as the first history block — an orphan block a later prune or fold would drop
// while its call stayed pinned in the anchor, which the provider rejects. A head
// that asked for nothing is returned unchanged.
func anchorEnd(conv []nacelle.Message, anchor int) int {
	if anchor <= 0 || anchor >= len(conv) {
		return anchor
	}
	calls := toolCallIDs(conv[anchor-1])
	end := anchor
	for end < len(conv) && answersAny(conv[end], calls) {
		end++
	}
	return end
}

// activeStart is where the verbatim window begins: the newest KeepTurns
// messages, pulled back off a ToolResult boundary and never into the anchor. The
// pull-back is skipped when it would land on the ledger: the ledger is not
// dropped with the cut, it carries the very call the result answers, so the pair
// stays valid with the result opening the active window instead of the ledger
// being swallowed into it.
func activeStart(conv []nacelle.Message, n, anchor, keep int) int {
	want := clamp(n-keep, anchor, n)
	if want <= anchor {
		return want
	}
	cut := max(AlignedCut(conv, want), anchor)
	if cut < want && IsLedger(conv[cut]) {
		return want
	}
	return cut
}

func ledgerIndex(conv []nacelle.Message, anchor, active int) int {
	for i := anchor; i < active; i++ {
		if IsLedger(conv[i]) {
			return i
		}
	}
	return -1
}
