package update

import (
	"os"
	"path/filepath"
	"strings"
)

// Method identifies how the running CLI was installed.
type Method string

const (
	MethodNPM      Method = "npm"
	MethodHomebrew Method = "homebrew"
	MethodBash     Method = "bash"
	MethodBinary   Method = "binary"
)

// EnvInstallMethod lets installers declare the channel explicitly,
// bypassing path-based heuristics. Homebrew formula and install.sh set this.
const EnvInstallMethod = "INNGEST_INSTALL_METHOD"

// Detect returns the install method, preferring the env override and
// falling back to a path heuristic on os.Executable().
func Detect() Method {
	if m := normalize(os.Getenv(EnvInstallMethod)); m != "" {
		return m
	}
	exe, err := os.Executable()
	if err != nil {
		return MethodBinary
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return classifyPath(exe)
}

func normalize(v string) Method {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "npm", "npx":
		return MethodNPM
	case "homebrew", "brew":
		return MethodHomebrew
	case "bash", "sh", "install.sh", "curl":
		return MethodBash
	case "binary":
		return MethodBinary
	}
	return ""
}

func classifyPath(p string) Method {
	p = filepath.ToSlash(p)
	switch {
	case strings.Contains(p, "/node_modules/"),
		strings.Contains(p, "/_npx/"):
		return MethodNPM
	case strings.Contains(p, "/Cellar/"),
		strings.HasPrefix(p, "/opt/homebrew/"),
		strings.HasPrefix(p, "/home/linuxbrew/"):
		return MethodHomebrew
	case p == "/usr/local/bin/inngest", p == "/usr/bin/inngest":
		return MethodBash
	}
	return MethodBinary
}

// UpgradeCommand returns the recommended upgrade command for the method.
func UpgradeCommand(m Method) string {
	switch m {
	case MethodNPM:
		return "npm install -g inngest-cli@latest"
	case MethodHomebrew:
		return "brew upgrade inngest"
		// The default is the bash command. The user will always see a link
		// to the GitHub releases page if they want to download the binary manually
	case MethodBash:
	default:
		return "curl -fsSL https://cli.inngest.com/install.sh | sh"
	}
}
