package env

import (
	"net/url"
	"os"
)

const (
	DevServerOrigin       = "http://127.0.0.1:8288"
	DefaultAPIOrigin      = "https://api.inngest.com"
	DefaultEventAPIOrigin = "https://inn.gs"
)

// IsDev returns whether to use the dev server, by checking the presence of the INNGEST_DEV
// environment variable.
//
// To use the dev server, set INNGEST_DEV to any non-empty value OR the URL of the development
// server, eg:
//
//	INNGEST_DEV=1
//	INNGEST_DEV=http://192.168.1.254:8288
func IsDev() bool {
	return os.Getenv("INNGEST_DEV") != ""
}

// DevServerURL returns the URL for the Inngest dev server.  This uses the INNGEST_DEV
// environment variable, or defaults to 'http://127.0.0.1:8288' if unset.
func DevServerURL() string {
	if dev := os.Getenv("INNGEST_DEV"); dev != "" {
		if u, err := url.Parse(dev); err == nil && u.Host != "" {
			// Only return this if it's a valid URL.
			return dev
		}
	}
	return DevServerOrigin
}

// APIServerURL returns the URL used to access the Inngest API.  This uses the INNGEST_DEV
// environment variable, or defaults to 'https://api.inngest.com' (production) if unset.
func APIServerURL(override *string) string {
	if override != nil {
		return *override
	}
	if IsDev() {
		return DevServerURL()
	}
	return "https://api.inngest.com"
}
