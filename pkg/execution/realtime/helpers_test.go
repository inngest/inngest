package realtime

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// newTestBroadcaster creates a broadcaster backed by an in-memory miniredis instance,
// suitable for use in tests. It uses the same Redis code path as production.
func newTestBroadcaster(t *testing.T) Broadcaster {
	t.Helper()
	r := miniredis.RunT(t)
	pubc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	subc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	return NewRedisBroadcaster(pubc, subc)
}

// newTestBroadcasterWithOpts is like newTestBroadcaster but accepts BroadcasterOpts.
func newTestBroadcasterWithOpts(t *testing.T, opts BroadcasterOpts) Broadcaster {
	t.Helper()
	r := miniredis.RunT(t)
	pubc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	subc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	return NewRedisBroadcaster(pubc, subc, opts)
}

// subCount returns the number of active subscriptions on a broadcaster.
func subCount(b Broadcaster) int {
	bc := b.(*broadcaster)
	bc.l.RLock()
	defer bc.l.RUnlock()
	return len(bc.subs)
}
