package dnscache

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestDNSCache(t *testing.T) {
	ctx := context.Background()
	l := logger.StdlibLogger(ctx)

	cachedResolver := New(WithLogger(l))

	c := http.Client{
		Transport: &http.Transport{
			DialContext: cachedResolver.Dialer(),
		},
	}

	// inngest and vercel use SNI
	addrs := []string{"https://www.example.com", "https://www.inngest.com", "https://vercel.com"}

	for _, addr := range addrs {
		host := parseHost(addr)
		testName := fmt.Sprintf("not cached: %s", host)

		t.Run(testName, func(t *testing.T) {
			require.False(t, cachedResolver.isCached(host))
		})
	}

	for _, host := range addrs {
		testName := fmt.Sprintf("host: %s", host)

		t.Run(testName, func(t *testing.T) {
			resp, err := c.Get(host)
			require.NoError(t, err)
			require.EqualValues(t, 200, resp.StatusCode)
		})
	}

	// These shouldn't incur lookups, as we just looked them up.
	for _, addr := range addrs {
		host := parseHost(addr)
		testName := fmt.Sprintf("cached: %s", host)

		t.Run(testName, func(t *testing.T) {
			require.True(t, cachedResolver.isCached(host))
		})
	}
}

func parseHost(urlstr string) string {
	parsed, err := url.Parse(urlstr)
	if err != nil {
		return ""
	}
	return parsed.Host
}
