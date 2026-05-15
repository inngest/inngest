package inngestgo

import (
	"fmt"
	"net/url"
	"os"
)

func serveOriginOverride(hOpts handlerOpts) *string {
	if hOpts.ServeOrigin != nil {
		return hOpts.ServeOrigin
	}
	if hOpts.URL != nil {
		return Ptr(fmt.Sprintf("%s://%s", hOpts.URL.Scheme, hOpts.URL.Host))
	}
	if v := os.Getenv("INNGEST_SERVE_HOST"); v != "" {
		return &v
	}
	return nil
}

func servePathOverride(hOpts handlerOpts) *string {
	if hOpts.ServePath != nil {
		return hOpts.ServePath
	}
	if hOpts.URL != nil {
		return Ptr(hOpts.URL.Path)
	}
	if v := os.Getenv("INNGEST_SERVE_PATH"); v != "" {
		return &v
	}
	return nil
}

func overrideURL(u *url.URL, hOpts handlerOpts) (*url.URL, error) {
	origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	if override := serveOriginOverride(hOpts); override != nil {
		origin = *override
	}

	path := u.Path
	if override := servePathOverride(hOpts); override != nil {
		path = *override
	}

	rawURL := fmt.Sprintf("%s%s", origin, path)
	if u.RawQuery != "" {
		rawURL += "?" + u.RawQuery
	}

	return url.Parse(rawURL)
}
