package app

import "testing"

func TestGetLayoutMode(t *testing.T) {
	tests := []struct {
		width int
		want  LayoutMode
	}{
		{0, LayoutNarrow},
		{20, LayoutNarrow},
		{39, LayoutNarrow},
		{40, LayoutMedium},
		{60, LayoutMedium},
		{79, LayoutMedium},
		{80, LayoutWide},
		{120, LayoutWide},
		{200, LayoutWide},
	}
	for _, tt := range tests {
		got := GetLayoutMode(tt.width)
		if got != tt.want {
			t.Errorf("GetLayoutMode(%d) = %d, want %d", tt.width, got, tt.want)
		}
	}
}

func TestTabBarHeight(t *testing.T) {
	if got := TabBarHeight(); got != 1 {
		t.Errorf("TabBarHeight() = %d, want 1", got)
	}
}

func TestStatusBarHeight(t *testing.T) {
	if got := StatusBarHeight(); got != 1 {
		t.Errorf("StatusBarHeight() = %d, want 1", got)
	}
}

func TestContentHeight(t *testing.T) {
	tests := []struct {
		total int
		want  int
	}{
		{24, 22},
		{2, 0},
		{1, 0},
		{0, 0},
		{80, 78},
	}
	for _, tt := range tests {
		got := ContentHeight(tt.total)
		if got != tt.want {
			t.Errorf("ContentHeight(%d) = %d, want %d", tt.total, got, tt.want)
		}
	}
}
