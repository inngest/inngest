package update

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	inncli "github.com/inngest/inngest/pkg/cli"
	isatty "github.com/mattn/go-isatty"
)

// EnabledFor reports whether the update notifier should run for the named
// subcommand. To enable the notifier for a new command, add it here.
func EnabledFor(name string) bool {
	switch name {
	case "dev", "help", "version", "":
		return true
	}
	return false
}

// Notify renders the update notice to w (typically os.Stderr) if a newer
// version is cached and every skip-gate passes. Always safe to call.
func Notify(w io.Writer, currentVersion string) {
	if disabled(currentVersion) {
		return
	}
	latest, url := Latest()
	if !IsNewer(currentVersion, latest) {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, render(currentVersion, latest, url, Detect()))
	fmt.Fprintln(w)
}

// disabled reports whether the update notifier is suppressed by env, TTY,
// or dev-build state. Shared by Check and Notify so opting out also
// silences the background network request — not just the rendering.
func disabled(currentVersion string) bool {
	if currentVersion == "" || currentVersion == "dev" {
		return true
	}
	for _, k := range []string{
		"INNGEST_NO_UPDATE_NOTIFIER",
		"NO_UPDATE_NOTIFIER",
		"DO_NOT_TRACK",
		"CI",
	} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	// Stderr-only output — skip when stderr isn't a terminal so we don't
	// pollute redirected error logs (and so we never fetch when we'd never
	// render).
	return !isatty.IsTerminal(os.Stderr.Fd())
}

func render(current, latest, url string, m Method) string {
	header := lipgloss.NewStyle().Bold(true).Foreground(inncli.Orange).
		Render("Update available")
	arrow := lipgloss.NewStyle().Foreground(inncli.Feint).Render("→")
	versions := fmt.Sprintf("%s %s %s", current, arrow, latest)

	runLabel := lipgloss.NewStyle().Foreground(inncli.Feint).Render("Run:")
	cmd := lipgloss.NewStyle().Bold(true).Render(UpgradeCommand(m))

	body := fmt.Sprintf("%s  %s\n%s %s", header, versions, runLabel, cmd)
	if url != "" {
		body += "\n" + lipgloss.NewStyle().Foreground(inncli.Feint).Render(url)
	}
	return inncli.UpdateNoticeStyle.Render(body)
}
