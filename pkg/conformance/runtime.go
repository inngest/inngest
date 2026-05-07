package conformance

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// RuntimeConfig is the normalized execution contract used by runners.
//
// The CLI and file-based config can both be partial, so runners should consume
// this concrete structure instead of repeating URL/default logic.
type RuntimeConfig struct {
	Transport      Transport  `json:"transport"`
	SDKURL         *url.URL   `json:"sdk_url,omitempty"`
	IntrospectURL  *url.URL   `json:"introspect_url,omitempty"`
	DevURL         *url.URL   `json:"dev_url,omitempty"`
	APIURL         *url.URL   `json:"api_url,omitempty"`
	EventURL       *url.URL   `json:"event_url,omitempty"`
	EventKey       string     `json:"event_key,omitempty"`
	SigningKey     string     `json:"signing_key,omitempty"`
	Timeout        time.Duration `json:"timeout"`
}

// Runtime resolves the user-facing config into concrete URLs and defaults.
//
// The current implementation defaults the transport to serve because Phase 2
// only ships a real serve runner. Connect remains in the registry and the
// report model, but not in the executable runtime path yet.
func (c Config) Runtime() (RuntimeConfig, error) {
	rt := RuntimeConfig{
		Transport: c.Transport,
		Timeout:   c.TimeoutOrDefault(60 * time.Second),
		EventKey:  c.Dev.EventKey,
		SigningKey: c.Dev.SigningKey,
	}

	if rt.Transport == "" {
		rt.Transport = TransportServe
	}

	var err error
	if c.SDK.URL != "" {
		rt.SDKURL, err = url.Parse(c.SDK.URL)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid sdk.url %q: %w", c.SDK.URL, err)
		}
	}

	if c.Dev.URL != "" {
		rt.DevURL, err = url.Parse(c.Dev.URL)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid dev.url %q: %w", c.Dev.URL, err)
		}
	}

	if c.Dev.APIURL != "" {
		rt.APIURL, err = url.Parse(c.Dev.APIURL)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid dev.api_url %q: %w", c.Dev.APIURL, err)
		}
	} else if rt.DevURL != nil {
		rt.APIURL = cloneURL(rt.DevURL)
	}

	if c.Dev.EventURL != "" {
		rt.EventURL, err = url.Parse(c.Dev.EventURL)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid dev.event_url %q: %w", c.Dev.EventURL, err)
		}
	} else if rt.DevURL != nil {
		rt.EventURL = cloneURL(rt.DevURL)
	}

	if rt.EventKey == "" {
		rt.EventKey = "test"
	}

	if c.SDK.IntrospectPath != "" {
		if rt.SDKURL == nil {
			return RuntimeConfig{}, fmt.Errorf("sdk.introspect_path requires sdk.url to also be set")
		}

		rt.IntrospectURL = cloneURL(rt.SDKURL)
		rt.IntrospectURL.Path = c.SDK.IntrospectPath
	} else if rt.SDKURL != nil {
		rt.IntrospectURL = cloneURL(rt.SDKURL)
		rt.IntrospectURL.Path = deriveIntrospectPath(rt.SDKURL.Path)
	}

	return rt, nil
}

func cloneURL(in *url.URL) *url.URL {
	if in == nil {
		return nil
	}

	out := *in
	return &out
}

func deriveIntrospectPath(servePath string) string {
	trimmed := strings.TrimSpace(servePath)
	if trimmed == "" || trimmed == "/" {
		return "/api/introspect"
	}

	if strings.HasSuffix(trimmed, "/inngest") {
		return strings.TrimSuffix(trimmed, "/inngest") + "/introspect"
	}

	return "/api/introspect"
}
