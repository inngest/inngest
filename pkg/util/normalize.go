package util

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// NormalizeAppURL normalizes localhost and 127.0.0.1 as the same string.  This
// ensures that we don't add duplicate apps.
func NormalizeAppURL(u string, forceHTTPS bool) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}

	parsed = stripDeployID(*parsed)

	if forceHTTPS {
		isWebSocket := strings.HasPrefix(parsed.Scheme, "ws")
		parsed.Scheme = "https"
		if isWebSocket {
			parsed.Scheme = "wss"
		}
	}

	if strings.Contains(parsed.Host, ":") {
		host, port, err := net.SplitHostPort(parsed.Host)
		if err != nil {
			return parsed.String()
		}

		switch host {
		case "localhost", "127.0.0.1", "0.0.0.0":
			parsed.Host = fmt.Sprintf("localhost:%s", port)
			return parsed.String()
		default:
			return parsed.String()
		}
	}

	return parsed.String()
}

func stripDeployID(u url.URL) *url.URL {
	qp := u.Query()
	qp.Del("deployId")
	u.RawQuery = qp.Encode()
	return &u
}
