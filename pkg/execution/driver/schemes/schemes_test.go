package schemes

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	Register("registered", func(u *url.URL) string {
		return "driver-" + u.Host
	})
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		delete(registered, "registered")
	})

	tests := []struct {
		name       string
		uri        string
		driver     string
		registered bool
	}{
		{
			name:       "registered scheme",
			uri:        "registered://a?fnId=x",
			driver:     "driver-a",
			registered: true,
		},
		{
			name: "unregistered scheme",
			uri:  "other://a?fnId=x",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.uri)
			require.NoError(t, err)
			driver, ok := Driver(u)
			require.Equal(t, tt.registered, ok)
			require.Equal(t, tt.driver, driver)
			require.Equal(t, tt.registered, Registered(u.Scheme))
		})
	}

	require.Panics(t, func() {
		Register("registered", func(u *url.URL) string {
			return "other"
		})
	})
}
