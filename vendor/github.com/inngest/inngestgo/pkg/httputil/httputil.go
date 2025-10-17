package httputil

import "net/http"

func GetScheme(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		if proto == "https" || proto == "http" {
			return proto
		}
	}

	if r.TLS != nil {
		return "https"
	}

	return "http"
}
