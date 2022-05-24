package cli

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/inngest/inngest-cli/inngest/state"
)

func EnvString() string {
	prod := state.IsProd()

	var env string
	prefix := lipgloss.NewStyle().Bold(true).Padding(0, 1, 0, 0).Render("Environment:")

	if prod {
		env = lipgloss.NewStyle().
			Background(Primary).
			Foreground(White).
			Bold(true).
			Padding(0, 1).
			Render("Production")
	} else {
		env = lipgloss.NewStyle().
			Background(White).
			Foreground(Black).
			Bold(true).
			Padding(0, 1).
			Render("Test")
	}

	return prefix + env + "\n"
}
