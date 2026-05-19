package tui

// Shortcut describes one keybinding in the aipim TUI.
type Shortcut struct {
	Keys        string `json:"keys"`
	Description string `json:"description"`
}

// Shortcuts returns the canonical list of TUI keybindings.
// Used by the TUI help overlay and by `aipim shortcuts` so the docs never drift.
func Shortcuts() []Shortcut {
	return []Shortcut{
		{Keys: "enter", Description: "launch the highlighted profile"},
		{Keys: "n", Description: "create a new profile"},
		{Keys: "e", Description: "edit the highlighted profile"},
		{Keys: "d", Description: "delete the highlighted profile"},
		{Keys: "s", Description: "open settings (agents, default agent)"},
		{Keys: "up / down", Description: "move the selection"},
		{Keys: "?", Description: "toggle the help overlay"},
		{Keys: "q / ctrl+c", Description: "quit"},
	}
}
