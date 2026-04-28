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

	parsed = stripInternalQueryParams(*parsed)

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

		// Strip default ports so that URLs with and without the default
		// port are treated as identical (e.g. http://host/path and
		// http://host:80/path).
		isDefaultPort := (port == "80" && (parsed.Scheme == "http" || parsed.Scheme == "ws")) ||
			(port == "443" && (parsed.Scheme == "https" || parsed.Scheme == "wss"))
		if isDefaultPort {
			port = ""
		}

		switch host {
		case "localhost", "127.0.0.1", "0.0.0.0":
			if port == "" {
				parsed.Host = "localhost"
			} else {
				parsed.Host = fmt.Sprintf("localhost:%s", port)
			}
			return parsed.String()
		default:
			if port == "" {
				parsed.Host = host
			}
			return parsed.String()
		}
	}

	return parsed.String()
}

func stripInternalQueryParams(u url.URL) *url.URL {
	qp := u.Query()
	qp.Del("deployId")
	qp.Del("probe")
	u.RawQuery = qp.Encode()
	return &u
}
