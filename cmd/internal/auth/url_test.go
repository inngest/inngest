package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateOAuthURL(t *testing.T) {
	tests := map[string]struct {
		url     string
		wantErr bool
	}{
		"HTTPS":              {url: "https://app.inngest.com/oauth/device?request=value"},
		"IPv4 loopback HTTP": {url: "http://127.0.0.1:5173/oauth/device"},
		"IPv6 loopback HTTP": {url: "http://[::1]:5173/oauth/device"},
		"localhost HTTP":     {url: "http://localhost:5173/oauth/device"},
		"remote HTTP":        {url: "http://example.com/oauth/device", wantErr: true},
		"file URL":           {url: "file:///tmp/device", wantErr: true},
		"user info":          {url: "https://user@example.com/oauth/device", wantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateOAuthURLString(test.url)
			require.Equal(t, test.wantErr, err != nil)
		})
	}
}
