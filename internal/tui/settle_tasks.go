// Package tui — the reminder a finished run leaves for the next one: an
// unfinished plan, said out loud and put in the conversation so the model sees it
// too. It lives apart from settle.go because the file cap is real and this is a
// different question from ending a run.
package tui

import (
	"fmt"

	"github.com/FacileStudio/nacelle"
)

// taskReminder adds a reminder to the conversation when the run ended normally
// and tasks remain unfinished, so the model sees them on the next turn and
// either updates their status or adjusts the plan.
//
// It runs after closeTurn so the conversation has the finished turn committed
// before the reminder is inserted. The reminder is a user message with a
// bracket-prefixed prefix, clearly the client speaking and not the person.
func (m *Model) taskReminder() {
	if m.run.stop != nacelle.StopEnd {
		return
	}
	if !m.tasksUnfinished() {
		return
	}
	m.say(fromTool, "\u2606 tasks: "+m.tasksSummary())
	m.conversation = append(m.conversation, nacelle.UserText(m.tasksSummary()))
}

// tasksUnfinished returns true when the plan has steps that are not completed.
func (m *Model) tasksUnfinished() bool {
	for _, item := range m.tasks {
		if item.Status != statusDone {
			return true
		}
	}
	return false
}

// tasksSummary returns a short description of the plan's state for reminders.
func (m *Model) tasksSummary() string {
	total := len(m.tasks)
	if total == 0 {
		return ""
	}
	done := 0
	for _, item := range m.tasks {
		if item.Status == statusDone {
			done++
		}
	}
	switch done {
	case total:
		return ""
	case 0:
		return fmt.Sprintf("tasks: %d steps, none done — keep the plan current as you go", total)
	default:
		return fmt.Sprintf("tasks: %d/%d steps complete — update the plan before continuing", done, total)
	}
}
