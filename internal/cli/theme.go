package cli

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#67E8F9"))
	brandStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC"))
	bannerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#22D3EE"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#67E8F9"))
	accentStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A7F3D0"))
	greenStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#34D399"))
	yellowStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FBBF24"))
	redStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FB7185"))
	whiteStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F8FAFC"))
	boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#475569")).Padding(0, 1)
	panelStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#334155")).Padding(0, 1)
)

const (
	symSelect = "❯"
	symActive = "●"
	symOn     = "●"
	symOff    = "○"
	symOK     = "✓"
	symFail   = "✕"
	symWarn   = "⚠"
	symLoad   = "◌"
	symPlus   = "＋"
	symBrand  = "◈"
	symTree   = "├─"
	symTreeEnd = "└─"
)
