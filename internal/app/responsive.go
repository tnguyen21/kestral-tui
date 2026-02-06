package app

// LayoutMode represents the display width category for responsive layout.
type LayoutMode int

const (
	LayoutNarrow LayoutMode = iota // <40 chars: icons only in tab bar
	LayoutMedium                   // 40-79: short titles
	LayoutWide                     // 80+: full titles
)

// GetLayoutMode returns the appropriate layout mode for the given terminal width.
func GetLayoutMode(width int) LayoutMode {
	switch {
	case width < 40:
		return LayoutNarrow
	case width < 80:
		return LayoutMedium
	default:
		return LayoutWide
	}
}

// TabBarHeight returns the height of the tab bar (always 1 row).
func TabBarHeight() int {
	return 1
}

// StatusBarHeight returns the height of the status bar (always 1 row).
func StatusBarHeight() int {
	return 1
}

// ContentHeight returns the available height for content after subtracting
// the tab bar and status bar from the total terminal height.
func ContentHeight(totalHeight int) int {
	h := totalHeight - TabBarHeight() - StatusBarHeight()
	if h < 0 {
		return 0
	}
	return h
}
