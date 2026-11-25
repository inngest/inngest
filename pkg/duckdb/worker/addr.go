package worker

import (
	"net"
	"strconv"
	"strings"
)

// ParseAddress splits a worker endpoint into a (network, address) pair for
// net.Listen / net.Dialer.DialContext:
//   - "tcp://host:port"  -> ("tcp", "host:port")
//   - "unix:///path"     -> ("unix", "/path")
//   - "host:port" / ":p" -> ("tcp", addr) when net.SplitHostPort succeeds and
//     the port is numeric (so Windows drive paths like C:\... stay "unix")
//   - anything else      -> ("unix", addr)
func ParseAddress(addr string) (network, address string) {
	if rest, ok := strings.CutPrefix(addr, "tcp://"); ok {
		return "tcp", rest
	}
	if rest, ok := strings.CutPrefix(addr, "unix://"); ok {
		return "unix", rest
	}
	if _, port, err := net.SplitHostPort(addr); err == nil && port != "" {
		if _, err := strconv.Atoi(port); err == nil {
			return "tcp", addr
		}
	}
	return "unix", addr
}
