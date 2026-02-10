package pane

import (
	"testing"
	"time"
)

func TestTruncateWithEllipsis(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncated", "hello world", 5, "hell…"},
		{"maxLen 1", "hello", 1, "…"},
		{"maxLen 0", "hello", 0, ""},
		{"negative maxLen", "hello", -1, ""},
		{"empty string", "", 5, ""},
		{"unicode", "日本語テスト", 4, "日本語…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateWithEllipsis(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("TruncateWithEllipsis(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"one minute", 1 * time.Minute, "1m ago"},
		{"hours", 2 * time.Hour, "2h ago"},
		{"days", 3 * 24 * time.Hour, "3d ago"},
		{"zero", 0, "just now"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAge(tt.d)
			if got != tt.want {
				t.Errorf("FormatAge(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestStatusColor(t *testing.T) {
	statuses := []string{
		"OPEN", "PENDING", "IN_PROGRESS",
		"COMPLETED", "CLOSED", "BLOCKED",
		"ESCALATED", "UNKNOWN",
	}
	for _, s := range statuses {
		t.Run(s, func(t *testing.T) {
			style := StatusColor(s)
			// Verify it returns a usable style (renders without panic).
			_ = style.Render("test")
		})
	}
}

func TestPaneIDValues(t *testing.T) {
	// Ensure the iota enum values are sequential starting from 0.
	if PaneDashboard != 0 {
		t.Errorf("PaneDashboard = %d, want 0", PaneDashboard)
	}
	if PaneAgents != 1 {
		t.Errorf("PaneAgents = %d, want 1", PaneAgents)
	}
	if PaneRefinery != 2 {
		t.Errorf("PaneRefinery = %d, want 2", PaneRefinery)
	}
	if PanePRs != 3 {
		t.Errorf("PanePRs = %d, want 3", PanePRs)
	}
	if PaneHistory != 7 {
		t.Errorf("PaneHistory = %d, want 7", PaneHistory)
	}
	if PaneLogs != 9 {
		t.Errorf("PaneLogs = %d, want 9", PaneLogs)
	}
}
