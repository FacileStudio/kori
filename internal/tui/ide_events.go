package tui

import "github.com/FacileStudio/nacelle"

// The editor is told the shape of the run it is watching: that a model turn
// started, and that the run ended and why. Both are read from where the session
// already knows them, so the editor and the terminal cannot disagree about what
// happened.

// ideTurn reports a model turn starting. nacelle owns the tool loop, so the
// terminal never sees a per-iteration boundary directly; what it does see is
// KindTurn, which is one model call ending. The first streaming event after
// that is proof the next call is under way, and a run that ends instead never
// sends one — so no turn is invented for a call that never happened.
func (m *Model) ideTurn(kind nacelle.Kind) {
	if m.ide.surface == nil {
		return
	}
	switch kind {
	case nacelle.KindTurn:
		m.ide.open = false
		return
	case nacelle.KindDone, nacelle.KindToolResult:
		return
	}
	if m.ide.open {
		return
	}
	m.ide.open = true
	m.ide.turns++
	m.ide.surface.Turn(m.ide.turns)
}

// ideDone reports the run over, with the reason the protocol names. A run
// abandoned by esc is cancelled, one that spent its iteration budget is
// max_iterations, one a provider error ended is error, and anything else is the
// model having finished what it was asked.
func (m *Model) ideDone() {
	if m.ide.surface == nil {
		return
	}
	m.ide.surface.Done(editorReason(m.run.stop, m.ide.failed), m.run.usage.Cost)
	m.ide.open = false
}

// editorReason names why a run ended, in the four words the protocol carries.
// The failure flag is checked first because a run a provider error ended keeps
// whatever stop the last turn left, and reporting that as a finished answer
// would tell the editor a broken run was a complete one.
func editorReason(stop nacelle.Stop, failed bool) string {
	switch {
	case failed:
		return "error"
	case stop == abandoned:
		return "cancelled"
	case stop == nacelle.StopIterations:
		return "max_iterations"
	default:
		return "end_turn"
	}
}
