package auth

import (
	"errors"
	"os/exec"
	"runtime"
)

func OpenBrowser(rawURL string) error {
	if err := validateOAuthURLString(rawURL); err != nil {
		return err
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", rawURL)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		command = exec.Command("xdg-open", rawURL)
	}
	if err := command.Start(); err != nil {
		return err
	}
	if command.Process == nil {
		return errors.New("browser process did not start")
	}
	return command.Process.Release()
}
