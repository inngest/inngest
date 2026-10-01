package auth

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

func validateOAuthURLString(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return errors.New("URL is invalid")
	}
	return validateOAuthURL(parsed)
}

func validateOAuthURL(parsed *url.URL) error {
	if parsed == nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return errors.New("URL must include a host and no user info")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		return nil
	case "http":
		if isLoopbackHost(parsed.Hostname()) {
			return nil
		}
		return errors.New("HTTP is allowed only for loopback addresses")
	default:
		return errors.New("URL must use HTTPS")
	}
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
