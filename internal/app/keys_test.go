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
		{"Pane8", km.Pane8, []string{"8"}},
		{"Pane9", km.Pane9, []string{"9"}},
		{"Pane0", km.Pane0, []string{"0"}},
		{"PanePicker", km.PanePicker, []string{" "}},
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

func TestPaneKeysOneToNine(t *testing.T) {
	km := DefaultKeyMap()
	panes := []key.Binding{
		km.Pane1, km.Pane2, km.Pane3, km.Pane4,
		km.Pane5, km.Pane6, km.Pane7, km.Pane8, km.Pane9,
	}
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

func TestPaneKeyZeroMapsToTen(t *testing.T) {
	km := DefaultKeyMap()
	keys := km.Pane0.Keys()
	if len(keys) != 1 || keys[0] != "0" {
		t.Errorf("Pane0 key = %v, want [\"0\"]", keys)
	}
}
