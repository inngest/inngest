package cli

import (
	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

var (
	Color   = compat.AdaptiveColor{Light: lipgloss.Color("#111222"), Dark: lipgloss.Color("#FAFAFA")}
	Primary = lipgloss.Color("#4636f5")
	Green   = lipgloss.Color("#9dcc3a")
	Red     = lipgloss.Color("#ff0000")
	White   = lipgloss.Color("#ffffff")
	Black   = lipgloss.Color("#000000")
	Orange  = lipgloss.Color("#D3A347")
	Feint   = compat.AdaptiveColor{Light: lipgloss.Color("#333333"), Dark: lipgloss.Color("#888888")}
	Iris    = lipgloss.Color("#5D5FEF")
	Fuschia = lipgloss.Color("#EF5DA8")

	TextStyle    = lipgloss.NewStyle().Foreground(Color)
	FeintStyle   = TextStyle.Foreground(Feint)
	BoldStyle    = TextStyle.Bold(true)
	WarningStyle = TextStyle.Foreground(Orange)
)

// RenderError returns a formatted error string.
func RenderError(msg string) string {
	// Error applies styles to an error message
	err := lipgloss.NewStyle().Background(Red).Foreground(White).Bold(true).Padding(0, 1).Render("Error")
	content := lipgloss.NewStyle().Bold(true).Padding(0, 1).Render(msg)
	return err + content
}

// RenderWarning returns a formatted warning string.
func RenderWarning(msg string) string {
	// Error applies styles to an error message
	err := lipgloss.NewStyle().Foreground(Orange).Bold(true).Render("Warning: ")
	content := lipgloss.NewStyle().Bold(true).Foreground(Orange).Padding(0, 1).Render(msg)
	return err + content
}
