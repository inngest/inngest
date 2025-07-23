package util

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
)

// ParsePort parses a port string that can be either a plain port number
// or a TCP URL (e.g., "tcp://192.168.194.165:8288")
func ParsePort(ps string) (int, error) {
	if ps == "" {
		return 0, fmt.Errorf("port cannot be empty")
	}

	// Handle both plain port numbers and TCP URLs
	if u, err := url.Parse(ps); err == nil && u.Scheme != "" {
		// Extract port from URL
		_, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			return 0, fmt.Errorf("failed to parse port from URL %q: %w", ps, err)
		}
		num, err := strconv.Atoi(port)
		if err != nil {
			return 0, fmt.Errorf("invalid port number %q in URL %q: %w", port, ps, err)
		}
		return num, nil
	} else {
		// Try parsing as plain port number
		num, err := strconv.Atoi(ps)
		if err != nil {
			return 0, fmt.Errorf("invalid port %q: %w", ps, err)
		}
		return num, nil
	}
}
