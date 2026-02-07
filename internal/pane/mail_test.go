package pane

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewMailPane(t *testing.T) {
	p := NewMailPane()
	if p.ID() != PaneMail {
		t.Errorf("ID() = %d, want %d", p.ID(), PaneMail)
	}
	if p.Title() != "Mail" {
		t.Errorf("Title() = %q, want %q", p.Title(), "Mail")
	}
	if p.ShortTitle() != "\U0001F4E7" {
		t.Errorf("ShortTitle() = %q, want %q", p.ShortTitle(), "\U0001F4E7")
	}
	if p.Badge() != 0 {
		t.Errorf("Badge() = %d, want 0 for empty pane", p.Badge())
	}
}

func TestMailPaneBadge(t *testing.T) {
	p := NewMailPane()
	p.messages = []MailInfo{
		{ID: "m1", Subject: "Hello", Read: false},
		{ID: "m2", Subject: "World", Read: true},
		{ID: "m3", Subject: "Test", Read: false},
	}
	if got := p.Badge(); got != 2 {
		t.Errorf("Badge() = %d, want 2 (2 unread)", got)
	}
}

func TestMailPaneUpdateWithMailData(t *testing.T) {
	p := NewMailPane()
	p.SetSize(80, 24)

	messages := []MailInfo{
		{ID: "m1", From: "mayor/", Subject: "Assignment", Read: false, Timestamp: time.Now()},
		{ID: "m2", From: "kestral_tui/witness", Subject: "Health check", Read: true, Timestamp: time.Now()},
	}

	updated, _ := p.Update(MailUpdateMsg{Messages: messages})
	p = updated.(*MailPane)

	if len(p.messages) != 2 {
		t.Fatalf("messages count = %d, want 2", len(p.messages))
	}
	if p.Badge() != 1 {
		t.Errorf("Badge() = %d, want 1 (1 unread)", p.Badge())
	}
}

func TestMailPaneView(t *testing.T) {
	p := NewMailPane()
	p.SetSize(80, 24)

	messages := []MailInfo{
		{ID: "m1", From: "mayor/", Subject: "New assignment", Read: false, Timestamp: time.Now()},
		{ID: "m2", From: "kestral_tui/witness", Subject: "Health report", Read: true, Timestamp: time.Now()},
	}
	p.Update(MailUpdateMsg{Messages: messages})

	view := p.View()

	if !strings.Contains(view, "MAIL") {
		t.Error("View should contain 'MAIL' header")
	}

	if !strings.Contains(view, "New assignment") {
		t.Error("View should contain message subject 'New assignment'")
	}
	if !strings.Contains(view, "Health report") {
		t.Error("View should contain message subject 'Health report'")
	}

	if !strings.Contains(view, "j/k=scroll") {
		t.Error("View should contain scroll help footer")
	}
}

func TestMailPaneViewEmpty(t *testing.T) {
	p := NewMailPane()
	p.SetSize(80, 24)

	view := p.View()
	if !strings.Contains(view, "No messages") {
		t.Error("Empty pane should show 'No messages'")
	}
}

func TestMailPaneViewError(t *testing.T) {
	p := NewMailPane()
	p.SetSize(80, 24)

	p.Update(MailUpdateMsg{Err: &testError{}})

	view := p.View()
	if !strings.Contains(view, "Error") {
		t.Error("Error state should show error message")
	}
}

func TestMailPaneViewZeroSize(t *testing.T) {
	p := NewMailPane()
	view := p.View()
	if view != "" {
		t.Errorf("View with zero size should be empty, got %q", view)
	}
}

func TestMailPaneScrolling(t *testing.T) {
	p := NewMailPane()
	p.SetSize(80, 5) // Tiny viewport

	var msgs []MailInfo
	for i := 0; i < 20; i++ {
		msgs = append(msgs, MailInfo{
			ID:        "m" + string(rune('a'+i)),
			From:      "test/sender",
			Subject:   "Message " + string(rune('a'+i)),
			Read:      i%2 == 0,
			Timestamp: time.Now(),
		})
	}
	p.Update(MailUpdateMsg{Messages: msgs})

	if p.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", p.cursor)
	}

	// Move down
	for i := 0; i < 5; i++ {
		p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	if p.cursor != 5 {
		t.Errorf("cursor after 5x down = %d, want 5", p.cursor)
	}

	// Move up
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.cursor != 4 {
		t.Errorf("cursor after up = %d, want 4", p.cursor)
	}

	// Can't go below 0
	p.cursor = 0
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.cursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", p.cursor)
	}
}

func TestMailPaneMessageView(t *testing.T) {
	p := NewMailPane()
	p.SetSize(80, 24)

	messages := []MailInfo{
		{
			ID:        "m1",
			From:      "mayor/",
			To:        "kestral_tui/jasper",
			Subject:   "New assignment",
			Body:      "Please implement the mail pane.\nThanks!",
			Read:      false,
			Timestamp: time.Now(),
		},
	}
	p.Update(MailUpdateMsg{Messages: messages})

	// Press enter to view message
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if p.view != mailViewMessage {
		t.Fatalf("view should be mailViewMessage after enter, got %d", p.view)
	}

	view := p.View()
	if !strings.Contains(view, "MESSAGE") {
		t.Error("Message view should contain 'MESSAGE' header")
	}
	if !strings.Contains(view, "mayor/") {
		t.Error("Message view should contain From address")
	}
	if !strings.Contains(view, "New assignment") {
		t.Error("Message view should contain subject")
	}
	if !strings.Contains(view, "implement the mail pane") {
		t.Error("Message view should contain body text")
	}

	// Press esc to go back
	p.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if p.view != mailViewInbox {
		t.Errorf("view should be mailViewInbox after esc, got %d", p.view)
	}
}

func TestMailPaneSetSize(t *testing.T) {
	p := NewMailPane()
	p.SetSize(120, 40)
	if p.width != 120 || p.height != 40 {
		t.Errorf("SetSize(120, 40) -> width=%d, height=%d", p.width, p.height)
	}
}

func TestShortAddr(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{"mayor/", ""},
		{"kestral_tui/polecats/jasper", "jasper"},
		{"kestral_tui/witness", "witness"},
		{"simple", "simple"},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got := shortAddr(tt.addr)
			if got != tt.want {
				t.Errorf("shortAddr(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
		want  int // expected number of lines
	}{
		{"short", "hello", 80, 1},
		{"long", "this is a very long line that should be wrapped at the specified width boundary", 20, 5},
		{"multiline", "line one\nline two\nline three", 80, 3},
		{"empty", "", 80, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapText(tt.text, tt.width)
			if len(got) != tt.want {
				t.Errorf("wrapText(%q, %d) = %d lines, want %d; lines: %v",
					tt.text, tt.width, len(got), tt.want, got)
			}
		})
	}
}
