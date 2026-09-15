package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/FacileStudio/nacelle"
)

type spentDelegation struct {
	usage nacelle.Usage
}

func watchDelegations() tea.Cmd {
	return func() tea.Msg {
		return spentDelegation{usage: <-delegations}
	}
}

func (m *Model) recordDelegation(spent spentDelegation) tea.Cmd {
	m.run.usage = m.run.usage.Add(spent.usage)
	return watchDelegations()
}

func (m *Model) routeTask(message tea.Msg) (tea.Cmd, bool) {
	switch msg := message.(type) {
	case spentDelegation:
		return m.recordDelegation(msg), true
	case detachedResult:
		return m.recordDetached(msg), true
	case subagentUpdate:
		return m.recordUpdate(msg), true
	case taskTitled:
		return m.recordTitle(msg), true
	case taskUpdate:
		return m.recordTasks(msg), true
	}
	return nil, false
}
