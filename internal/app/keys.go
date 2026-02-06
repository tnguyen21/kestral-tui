package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application.
// Designed for mobile SSH friendliness: number keys for navigation,
// vim keys for scrolling, and tab for cycling panes.
type KeyMap struct {
	Quit     key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Pane1    key.Binding
	Pane2    key.Binding
	Pane3    key.Binding
	Pane4    key.Binding
	Pane5    key.Binding
	Pane6    key.Binding
	Pane7    key.Binding
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	Back     key.Binding
	Refresh  key.Binding
	Help     key.Binding
}

// DefaultKeyMap returns the default set of keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev pane"),
		),
		Pane1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "pane 1"),
		),
		Pane2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "pane 2"),
		),
		Pane3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "pane 3"),
		),
		Pane4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "pane 4"),
		),
		Pane5: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "pane 5"),
		),
		Pane6: key.NewBinding(
			key.WithKeys("6"),
			key.WithHelp("6", "pane 6"),
		),
		Pane7: key.NewBinding(
			key.WithKeys("7"),
			key.WithHelp("7", "pane 7"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}
