//go:build !windows

package worker

import (
	"os"
	"syscall"
)

func terminateProcess(p *os.Process) {
	if p != nil {
		_ = p.Signal(syscall.SIGTERM)
	}
}
