package auth

import (
	"errors"
	"os/exec"
	"runtime"
)

func OpenBrowser(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", url)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	if err := command.Start(); err != nil {
		return err
	}
	if command.Process == nil {
		return errors.New("browser process did not start")
	}
	return command.Process.Release()
}
