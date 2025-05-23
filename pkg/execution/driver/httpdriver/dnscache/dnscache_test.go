package dnscache

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestDNSCache(t *testing.T) {
	ctx := context.Background()
	l := logger.StdlibLogger(ctx)

	ttl := 2 * time.Second

	cachedResolver := New(
		WithCacheTTL(ttl),
		WithLogger(l),
	)

	c := http.Client{
		Transport: &http.Transport{
			DialContext: cachedResolver.Dialer(),
		},
	}

	// inngest and vercel use SNI
	hosts := []string{"www.example.com", "www.inngest.com", "vercel.com"}

	for _, host := range hosts {
		t.Run(host, func(t *testing.T) {
			t.Run("non cached", func(t *testing.T) {
				require.False(t, cachedResolver.isCached(host))
			})

			t.Run("request", func(t *testing.T) {
				resp, err := c.Get(fmt.Sprintf("https://%s", host))
				require.NoError(t, err)
				require.EqualValues(t, 200, resp.StatusCode)
			})

			t.Run("cached", func(t *testing.T) {
				require.True(t, cachedResolver.isCached(host))
			})

			<-time.After(ttl)

			t.Run("cache expired", func(t *testing.T) {
				require.False(t, cachedResolver.isCached(host))
			})
		})
	}
}
