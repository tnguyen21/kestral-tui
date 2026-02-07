package pane

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/data"
)

func TestNewPRsPane(t *testing.T) {
	p := NewPRsPane()
	if p.ID() != PanePRs {
		t.Errorf("ID() = %d, want %d", p.ID(), PanePRs)
	}
	if p.Title() != "PRs" {
		t.Errorf("Title() = %q, want %q", p.Title(), "PRs")
	}
	if p.ShortTitle() != "📋" {
		t.Errorf("ShortTitle() = %q, want %q", p.ShortTitle(), "📋")
	}
	if p.Badge() != 0 {
		t.Errorf("Badge() = %d, want 0 for empty pane", p.Badge())
	}
}

func TestPRsPaneBadge(t *testing.T) {
	p := NewPRsPane()
	p.prs = []data.PRInfo{
		{Number: 1, Title: "Fix bug"},
		{Number: 2, Title: "Add feature"},
	}
	if got := p.Badge(); got != 2 {
		t.Errorf("Badge() = %d, want 2", got)
	}
}

func TestPRsPaneUpdateWithData(t *testing.T) {
	p := NewPRsPane()
	p.SetSize(80, 24)

	prs := []data.PRInfo{
		{
			Number:         42,
			Title:          "Fix authentication bug",
			Author:         data.PRAuthor{Login: "alice"},
			HeadRefName:    "fix/auth-bug",
			CreatedAt:      "2026-02-06T12:00:00Z",
			ReviewDecision: "APPROVED",
			Mergeable:      "MERGEABLE",
			Additions:      10,
			Deletions:      3,
			ChangedFiles:   2,
			StatusChecks: []data.PRStatusCheck{
				{Name: "ci/test", Status: "COMPLETED", Conclusion: "SUCCESS"},
			},
		},
		{
			Number:         43,
			Title:          "Add new feature",
			Author:         data.PRAuthor{Login: "bob"},
			HeadRefName:    "feat/new-thing",
			CreatedAt:      "2026-02-07T01:00:00Z",
			IsDraft:        true,
			ReviewDecision: "",
			Mergeable:      "UNKNOWN",
		},
	}

	updated, _ := p.Update(PRUpdateMsg{PRs: prs})
	p = updated.(*PRsPane)

	if len(p.prs) != 2 {
		t.Fatalf("prs count = %d, want 2", len(p.prs))
	}
	if p.Badge() != 2 {
		t.Errorf("Badge() = %d, want 2", p.Badge())
	}
}

func TestPRsPaneViewList(t *testing.T) {
	p := NewPRsPane()
	p.SetSize(100, 24)

	prs := []data.PRInfo{
		{
			Number:         42,
			Title:          "Fix authentication bug",
			Author:         data.PRAuthor{Login: "alice"},
			HeadRefName:    "fix/auth-bug",
			CreatedAt:      "2026-02-06T12:00:00Z",
			ReviewDecision: "APPROVED",
			Mergeable:      "MERGEABLE",
			Additions:      10,
			Deletions:      3,
			ChangedFiles:   2,
			StatusChecks: []data.PRStatusCheck{
				{Name: "ci/test", Status: "COMPLETED", Conclusion: "SUCCESS"},
			},
		},
	}
	p.Update(PRUpdateMsg{PRs: prs})

	view := p.View()

	if !strings.Contains(view, "PRs") {
		t.Error("View should contain 'PRs' header")
	}
	if !strings.Contains(view, "#42") {
		t.Error("View should contain PR number '#42'")
	}
	if !strings.Contains(view, "Fix authentication bug") {
		t.Error("View should contain PR title")
	}
	if !strings.Contains(view, "alice") {
		t.Error("View should contain author name")
	}
	if !strings.Contains(view, "approved") {
		t.Error("View should contain review state 'approved'")
	}
	if !strings.Contains(view, "mergeable") {
		t.Error("View should contain merge state 'mergeable'")
	}
}

func TestPRsPaneViewEmpty(t *testing.T) {
	p := NewPRsPane()
	p.SetSize(80, 24)

	view := p.View()
	if !strings.Contains(view, "No open PRs") {
		t.Error("Empty pane should show 'No open PRs'")
	}
}

func TestPRsPaneViewError(t *testing.T) {
	p := NewPRsPane()
	p.SetSize(80, 24)

	p.Update(PRUpdateMsg{Err: errTest})

	view := p.View()
	if !strings.Contains(view, "Error") {
		t.Error("Error state should show error message")
	}
}

func TestPRsPaneViewZeroSize(t *testing.T) {
	p := NewPRsPane()
	view := p.View()
	if view != "" {
		t.Errorf("View with zero size should be empty, got %q", view)
	}
}

func TestPRsPaneDetailView(t *testing.T) {
	p := NewPRsPane()
	p.SetSize(80, 30)

	prs := []data.PRInfo{
		{
			Number:         42,
			Title:          "Fix authentication bug",
			Author:         data.PRAuthor{Login: "alice"},
			HeadRefName:    "fix/auth-bug",
			CreatedAt:      "2026-02-06T12:00:00Z",
			ReviewDecision: "APPROVED",
			Mergeable:      "MERGEABLE",
			Additions:      10,
			Deletions:      3,
			ChangedFiles:   2,
			StatusChecks: []data.PRStatusCheck{
				{Name: "ci/test", Status: "COMPLETED", Conclusion: "SUCCESS"},
				{Name: "ci/lint", Status: "COMPLETED", Conclusion: "FAILURE"},
			},
		},
	}
	p.Update(PRUpdateMsg{PRs: prs})

	// Enter detail view
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !p.detail {
		t.Fatal("Expected detail mode after enter key")
	}

	view := p.View()
	if !strings.Contains(view, "PR #42") {
		t.Error("Detail view should contain 'PR #42' header")
	}
	if !strings.Contains(view, "Fix authentication bug") {
		t.Error("Detail view should contain PR title")
	}
	if !strings.Contains(view, "alice") {
		t.Error("Detail view should contain author")
	}
	if !strings.Contains(view, "fix/auth-bug") {
		t.Error("Detail view should contain branch name")
	}
	if !strings.Contains(view, "ci/test") {
		t.Error("Detail view should contain check name")
	}
	if !strings.Contains(view, "ci/lint") {
		t.Error("Detail view should contain second check name")
	}

	// Exit detail view
	p.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if p.detail {
		t.Error("Expected list mode after esc key")
	}
}

func TestPRsPaneScrolling(t *testing.T) {
	p := NewPRsPane()
	p.SetSize(80, 6) // Very constrained

	var prs []data.PRInfo
	for i := 0; i < 10; i++ {
		prs = append(prs, data.PRInfo{
			Number:    i + 1,
			Title:     "PR title",
			Author:    data.PRAuthor{Login: "user"},
			CreatedAt: "2026-02-06T12:00:00Z",
		})
	}
	p.Update(PRUpdateMsg{PRs: prs})

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
}

func TestPRStatusIcon(t *testing.T) {
	tests := []struct {
		name string
		pr   data.PRInfo
	}{
		{
			"draft",
			data.PRInfo{IsDraft: true},
		},
		{
			"all green",
			data.PRInfo{
				ReviewDecision: "APPROVED",
				Mergeable:      "MERGEABLE",
				StatusChecks: []data.PRStatusCheck{
					{Conclusion: "SUCCESS"},
				},
			},
		},
		{
			"conflicting",
			data.PRInfo{Mergeable: "CONFLICTING"},
		},
		{
			"changes requested",
			data.PRInfo{ReviewDecision: "CHANGES_REQUESTED"},
		},
		{
			"checks failing",
			data.PRInfo{
				StatusChecks: []data.PRStatusCheck{
					{Conclusion: "FAILURE"},
				},
			},
		},
		{
			"checks pending",
			data.PRInfo{
				StatusChecks: []data.PRStatusCheck{
					{Conclusion: ""},
				},
			},
		},
		{
			"no checks no reviews",
			data.PRInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic and returns something
			icon := prStatusIcon(tt.pr)
			if icon == "" {
				t.Error("prStatusIcon should return a non-empty icon")
			}
		})
	}
}

func TestPRAllChecksPassed(t *testing.T) {
	tests := []struct {
		name   string
		checks []data.PRStatusCheck
		want   bool
	}{
		{"no checks", nil, false},
		{"all success", []data.PRStatusCheck{{Conclusion: "SUCCESS"}}, true},
		{"mixed", []data.PRStatusCheck{{Conclusion: "SUCCESS"}, {Conclusion: "FAILURE"}}, false},
		{"neutral counts as pass", []data.PRStatusCheck{{Conclusion: "NEUTRAL"}}, true},
		{"skipped counts as pass", []data.PRStatusCheck{{Conclusion: "SKIPPED"}}, true},
		{"pending", []data.PRStatusCheck{{Conclusion: ""}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := data.PRInfo{StatusChecks: tt.checks}
			got := prAllChecksPassed(pr)
			if got != tt.want {
				t.Errorf("prAllChecksPassed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPRReviewLabel(t *testing.T) {
	tests := []struct {
		decision string
		contains string
	}{
		{"APPROVED", "approved"},
		{"CHANGES_REQUESTED", "changes requested"},
		{"REVIEW_REQUIRED", "review required"},
		{"", "no reviews"},
	}

	for _, tt := range tests {
		t.Run(tt.decision, func(t *testing.T) {
			label := prReviewLabel(tt.decision)
			if !strings.Contains(label, tt.contains) {
				t.Errorf("prReviewLabel(%q) = %q, want to contain %q", tt.decision, label, tt.contains)
			}
		})
	}
}

func TestPRMergeLabel(t *testing.T) {
	tests := []struct {
		mergeable string
		contains  string
	}{
		{"MERGEABLE", "mergeable"},
		{"CONFLICTING", "conflicts"},
		{"UNKNOWN", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.mergeable, func(t *testing.T) {
			pr := data.PRInfo{Mergeable: tt.mergeable}
			label := prMergeLabel(pr)
			if !strings.Contains(label, tt.contains) {
				t.Errorf("prMergeLabel(%q) = %q, want to contain %q", tt.mergeable, label, tt.contains)
			}
		})
	}
}

func TestPRAge(t *testing.T) {
	// Invalid date
	got := prAge("not-a-date")
	if got != "?" {
		t.Errorf("prAge(invalid) = %q, want %q", got, "?")
	}

	// Very old date should return days ago
	got = prAge("2020-01-01T00:00:00Z")
	if !strings.Contains(got, "d ago") {
		t.Errorf("prAge(old date) = %q, want to contain 'd ago'", got)
	}
}

func TestPRsPaneSetSize(t *testing.T) {
	p := NewPRsPane()
	p.SetSize(120, 40)
	if p.width != 120 || p.height != 40 {
		t.Errorf("SetSize(120, 40) -> width=%d, height=%d", p.width, p.height)
	}
}
