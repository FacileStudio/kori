package compaction

import (
	"encoding/json"

	"github.com/FacileStudio/nacelle"
)

// callMessage is an assistant turn asking for one tool.
func callMessage(id, name string) nacelle.Message {
	return nacelle.Message{
		Role: nacelle.RoleAssistant,
		Parts: []nacelle.Part{
			nacelle.ToolCall{ID: id, Name: name, Input: json.RawMessage(`{}`), Finished: true},
		},
	}
}

// resultMessage is the user turn answering one tool call.
func resultMessage(id, name, result string) nacelle.Message {
	return nacelle.Message{
		Role: nacelle.RoleUser,
		Parts: []nacelle.Part{
			nacelle.ToolResult{ID: id, Name: name, Result: result},
		},
	}
}
