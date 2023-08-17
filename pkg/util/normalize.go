package util

import (
	"fmt"
	"net"
	"net/url"
)

// NormalizeAppURL normalizes localhost and 127.0.0.1 as the same string.  This
// ensures that we don't add duplicate apps.
func NormalizeAppURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}

	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		return u
	}

	switch host {
	case "localhost", "127.0.0.1", "0.0.0.0":
		parsed.Host = fmt.Sprintf("localhost:%s", port)
		return parsed.String()
	default:
		return u
	}
}
