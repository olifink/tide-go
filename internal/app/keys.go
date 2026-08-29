package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings across the application.
type KeyMap struct {
	Files        key.Binding
	NewFile      key.Binding
	ToggleEdit   key.Binding
	Save         key.Binding
	Build        key.Binding
	Run          key.Binding
	ShellCommand key.Binding
	Gemini       key.Binding
	Quit         key.Binding
	Tab          key.Binding
	Backtab      key.Binding
	Escape       key.Binding
	FocusConsole key.Binding
	Fullscreen   key.Binding
}

// DefaultKeyMap returns the standard key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Files: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("^F", "Files"),
		),
		NewFile: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("^N", "New"),
		),
		ToggleEdit: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("^E", "Edit"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("^S", "Save"),
		),
		Build: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("^B", "Build"),
		),
		Run: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("^R", "Run"),
		),
		ShellCommand: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("^X", "Shell"),
		),
		Gemini: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("^G", "Gemini"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+q"),
			key.WithHelp("^Q", "Quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "Next Pane"),
		),
		Backtab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("Shift+Tab", "Prev Pane"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "Focus Editor / View"),
		),
		FocusConsole: key.NewBinding(
			key.WithKeys("shift+esc"),
			key.WithHelp("Shift+Esc", "Focus Console"),
		),
		Fullscreen: key.NewBinding(
			key.WithKeys("ctrl+z", "f11"),
			key.WithHelp("^Z", "Fullscreen"),
		),
	}
}
