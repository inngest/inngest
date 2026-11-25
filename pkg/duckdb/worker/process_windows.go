//go:build windows

package worker

import "os"

func terminateProcess(p *os.Process) {
	if p != nil {
		_ = p.Kill()
	}
}
