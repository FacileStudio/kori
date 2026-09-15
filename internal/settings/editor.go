package settings

// Editor holds the settings for external prompt editing. Editor is the
// path to the editor executable; PromptEditKey names the input field to
// pre-fill when empty, meaning every message uses the full editor.
type Editor struct {
	Editor        string `json:"editor" yaml:"editor"`
	PromptEditKey string `json:"prompt_edit_key" yaml:"prompt_edit_key"`
}
