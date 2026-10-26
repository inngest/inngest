package constraintapi

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestCheckSemaphoreUnused(t *testing.T) {
	addr := os.Getenv("SEMAPHORE_TEST_REDIS_ADDR")
	if addr == "" {
		addr = miniredis.RunT(t).Addr()
	}
	host, _, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", host, "test must use a disposable local database")
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{addr}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	checker := NewRedisSemaphoreManager(rc).(UnusedSemaphoreChecker)
	ctx := context.Background()
	for _, value := range []string{"missing", "0", "1", "5", "-1", "invalid"} {
		t.Run(value, func(t *testing.T) {
			account := uuid.New()
			key := semaphoreUsageKey(account, "test", "bucket")
			if value != "missing" {
				require.NoError(t, rc.Do(ctx, rc.B().Set().Key(key).Value(value).Build()).Error())
			}
			err := checker.CheckSemaphoreUnused(ctx, account, "test", "bucket")
			if value == "missing" || value == "0" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			if value == "1" || value == "5" || value == "-1" {
				require.ErrorIs(t, err, ErrSemaphoreUsageNotEmpty)
			}
			after, err := rc.Do(ctx, rc.B().Get().Key(key).Build()).ToString()
			if value == "missing" {
				require.True(t, rueidis.IsRedisNil(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, value, after, "checking must never change shared usage")
			}
		})
	}
	account := uuid.New()
	require.NoError(t, checker.CheckSemaphoreUnused(ctx, account, "test", "later-holder"))
	key := semaphoreUsageKey(account, "test", "later-holder")
	require.NoError(t, rc.Do(ctx, rc.B().Set().Key(key).Value("1").Build()).Error())
	require.ErrorIs(t, checker.CheckSemaphoreUnused(ctx, account, "test", "later-holder"), ErrSemaphoreUsageNotEmpty)
	usage, err := rc.Do(ctx, rc.B().Get().Key(key).Build()).AsInt64()
	require.NoError(t, err)
	require.EqualValues(t, 1, usage, "retry must not release a later holder")
	rc.Close()
	require.Error(t, checker.CheckSemaphoreUnused(ctx, account, "test", "later-holder"), "dependency errors must not look like zero usage")
}
