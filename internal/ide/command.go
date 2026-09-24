package ide

// Command is one object an editor sends a session. Unknown types and unknown
// fields are ignored, which is what keeps a newer editor working against an
// older session and the other way round.
type Command struct {
	V      int    `json:"v"`
	T      string `json:"t"`
	Root   string `json:"root"`
	PID    int    `json:"pid"`
	Text   string `json:"text"`
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Branch string `json:"branch"`
	ID     string `json:"id"`
	Allow  bool   `json:"allow"`
}
