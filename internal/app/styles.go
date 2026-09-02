package app

import "github.com/charmbracelet/lipgloss"

// Palette defines the Dracula/Nord inspired color scheme.
var (
	ColorBg       = lipgloss.Color("#1E1E2E")
	ColorFg       = lipgloss.Color("#CDD6F4")
	ColorSubtext  = lipgloss.Color("#A6ADC8")
	ColorSurface  = lipgloss.Color("#313244")
	ColorOverlay  = lipgloss.Color("#45475A")
	ColorPrimary  = lipgloss.Color("#89B4FA") // Blue/Cyan
	ColorAccent   = lipgloss.Color("#CBA6F7") // Mauve/Purple
	ColorSuccess  = lipgloss.Color("#A6E3A1") // Green
	ColorWarning  = lipgloss.Color("#F9E2AF") // Yellow/Peach
	ColorDanger   = lipgloss.Color("#F38BA8") // Red
	ColorSapphire = lipgloss.Color("#74C7EC")
)

// Header Styles
var (
	HeaderLogoStyle = lipgloss.NewStyle().
			Bold(true).
			Background(ColorAccent).
			Foreground(ColorBg).
			Padding(0, 1)

	HeaderDirStyle = lipgloss.NewStyle().
			Foreground(ColorSapphire).
			Bold(true).
			Padding(0, 1)

	HeaderGitStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true).
			Padding(0, 1)

	HeaderGitDirtyStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true).
			Padding(0, 1)

	HeaderFileStyle = lipgloss.NewStyle().
			Foreground(ColorFg).
			Padding(0, 1)

	HeaderModStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWarning).
			Background(ColorSurface).
			Padding(0, 1)

	HeaderBar = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorFg).
			Height(1)
)

// Panel Border Styles
var (
	ActiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorAccent)

	InactiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorOverlay)

	PanelTitleActive = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent)

	PanelTitleInactive = lipgloss.NewStyle().
				Foreground(ColorSubtext)
)

// Footer Bar Styles
var (
	FooterKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Background(ColorPrimary).
			Foreground(ColorBg).
			Padding(0, 0)

	FooterDescStyle = lipgloss.NewStyle().
			Foreground(ColorFg).
			Padding(0, 1)

	FooterContainer = lipgloss.NewStyle().
			Background(ColorSurface).
			Height(1)
)
