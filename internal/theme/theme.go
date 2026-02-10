package theme

import "github.com/charmbracelet/lipgloss"

// Ayu color palette — AdaptiveColor for light/dark terminal support.
var (
	ColorPass   = lipgloss.AdaptiveColor{Light: "#86b300", Dark: "#c2d94c"}
	ColorWarn   = lipgloss.AdaptiveColor{Light: "#f2ae49", Dark: "#ffb454"}
	ColorFail   = lipgloss.AdaptiveColor{Light: "#f07171", Dark: "#f07178"}
	ColorMuted  = lipgloss.AdaptiveColor{Light: "#828c99", Dark: "#6c7680"}
	ColorAccent = lipgloss.AdaptiveColor{Light: "#399ee6", Dark: "#59c2ff"}
)

// Semantic text styles.
var (
	PassStyle   = lipgloss.NewStyle().Foreground(ColorPass)
	WarnStyle   = lipgloss.NewStyle().Foreground(ColorWarn)
	FailStyle   = lipgloss.NewStyle().Foreground(ColorFail)
	MutedStyle  = lipgloss.NewStyle().Foreground(ColorMuted)
	AccentStyle = lipgloss.NewStyle().Foreground(ColorAccent)
)

// Tab bar styles.
var (
	TabActiveStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true).
			Padding(0, 1)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 1)
)

// StatusBarStyle for the bottom status bar.
var StatusBarStyle = lipgloss.NewStyle().
	Background(lipgloss.AdaptiveColor{Light: "#e7e8e9", Dark: "#1f2430"}).
	Foreground(ColorMuted).
	Padding(0, 1)

// Header bar styles (compact pane indicator replacing full tab bar).
var (
	HeaderBarStyle = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#e7e8e9", Dark: "#1f2430"}).
			Foreground(ColorAccent).
			Bold(true).
			Padding(0, 1)

	HeaderHintStyle = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#e7e8e9", Dark: "#1f2430"}).
			Foreground(ColorMuted).
			Padding(0, 1)
)

// Pane picker overlay styles.
var (
	PickerTitleStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true).
				Padding(0, 1)

	PickerRowStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1)

	PickerActiveRowStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true).
				Padding(0, 1)

	PickerCursorStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)
)

// Pane styles.
var (
	PaneHeaderStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	PaneBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorMuted)
)
