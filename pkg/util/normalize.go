package util

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// NormalizeAppURL normalizes localhost and 127.0.0.1 as the same string.  This
// ensures that we don't add duplicate apps.
func NormalizeAppURL(u string, forceHTTPS bool) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	parsed = stripDeployID(*parsed)

	if forceHTTPS {
		if parsed.Scheme != "https" {
			parsed.Scheme = "https"
		}
	}

	if strings.Contains(parsed.Host, ":") {
		host, port, err := net.SplitHostPort(parsed.Host)
		if err != nil {
			return parsed.String(), err
		}

		// this shouldn't be valid: https://api.example.com:80/api/inngest
		if parsed.Scheme == "https" && port != "" {
			parsed.Host = host
			return parsed.String(), nil
		}

		switch host {
		case "localhost", "127.0.0.1", "0.0.0.0":
			parsed.Host = fmt.Sprintf("localhost:%s", port)
			return parsed.String(), nil
		default:
			return parsed.String(), nil
		}
	}

	return parsed.String(), nil
}

func stripDeployID(u url.URL) *url.URL {
	qp := u.Query()
	qp.Del("deployId")
	u.RawQuery = qp.Encode()
	return &u
}
