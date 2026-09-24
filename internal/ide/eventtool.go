package ide

// Change is one file an editing call changed, in the shape an editor marks it
// with: the line span it touched in the new file, how many lines it added and
// removed, and the tool that did it.
//
// First and Last are 1-based and inclusive. A change that only removed lines
// has no line of its own in the new file, and is reported at the one line the
// removal left behind, so a mark always lands somewhere real.
type Change struct {
	Path    string
	Tool    string
	First   int
	Last    int
	Added   int
	Removed int
	// Diff carries the unified diff when one was produced. It is optional in
	// the protocol and empty today: the session's own diff renderer draws into
	// a terminal rather than producing text another process can use.
	Diff string
}

// ToolStart reports a tool call that has begun. Path is set for the file tools
// whose input names one, and empty for everything else.
func ToolStart(id, name, path string) Event {
	fields := map[string]any{"id": id, "name": name, "status": "start"}
	if path != "" {
		fields["path"] = path
	}
	return Event{V: Protocol, T: "tool", Fields: fields}
}

// ToolDone reports a tool call that has finished, and whether it worked.
func ToolDone(id, name string, ok bool, path string) Event {
	fields := map[string]any{"id": id, "name": name, "status": "done", "ok": ok}
	if path != "" {
		fields["path"] = path
	}
	return Event{V: Protocol, T: "tool", Fields: fields}
}

// Edit reports a file that changed.
func Edit(id string, c Change) Event {
	fields := map[string]any{
		"id": id, "path": c.Path, "tool": c.Tool,
		"first": c.First, "last": c.Last, "added": c.Added, "removed": c.Removed,
	}
	if c.Diff != "" {
		fields["diff"] = c.Diff
	}
	return Event{V: Protocol, T: "edit", Fields: fields}
}

// Approval reports a tool call waiting for the user's yes or no. Input is the
// tool's own JSON, verbatim, so the editor shows exactly what is about to run
// rather than a summary somebody else wrote.
func Approval(id, tool, input string) Event {
	return Event{V: Protocol, T: "approval", Fields: map[string]any{
		"id": id, "tool": tool, "input": input,
	}}
}
