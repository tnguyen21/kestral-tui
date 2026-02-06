package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		name    string
		binding key.Binding
		keys    []string
	}{
		{"Quit", km.Quit, []string{"q", "ctrl+c"}},
		{"Tab", km.Tab, []string{"tab"}},
		{"ShiftTab", km.ShiftTab, []string{"shift+tab"}},
		{"Pane1", km.Pane1, []string{"1"}},
		{"Pane2", km.Pane2, []string{"2"}},
		{"Pane3", km.Pane3, []string{"3"}},
		{"Pane4", km.Pane4, []string{"4"}},
		{"Pane5", km.Pane5, []string{"5"}},
		{"Pane6", km.Pane6, []string{"6"}},
		{"Pane7", km.Pane7, []string{"7"}},
		{"Up", km.Up, []string{"k", "up"}},
		{"Down", km.Down, []string{"j", "down"}},
		{"Select", km.Select, []string{"enter"}},
		{"Back", km.Back, []string{"esc"}},
		{"Refresh", km.Refresh, []string{"r"}},
		{"Help", km.Help, []string{"?"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKeys := tt.binding.Keys()
			if len(gotKeys) != len(tt.keys) {
				t.Fatalf("%s: got %d keys, want %d", tt.name, len(gotKeys), len(tt.keys))
			}
			for i, k := range tt.keys {
				if gotKeys[i] != k {
					t.Errorf("%s: key[%d] = %q, want %q", tt.name, i, gotKeys[i], k)
				}
			}
		})
	}
}

func TestPaneKeysOneToSeven(t *testing.T) {
	km := DefaultKeyMap()
	panes := []key.Binding{km.Pane1, km.Pane2, km.Pane3, km.Pane4, km.Pane5, km.Pane6, km.Pane7}
	for i, b := range panes {
		keys := b.Keys()
		if len(keys) != 1 {
			t.Fatalf("Pane%d: expected 1 key, got %d", i+1, len(keys))
		}
		expected := string(rune('1' + i))
		if keys[0] != expected {
			t.Errorf("Pane%d: key = %q, want %q", i+1, keys[0], expected)
		}
	}
}
