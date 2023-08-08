//go:build ratelimit_test

package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/redis/rueidis"

	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/storetest"
)

const (
	redisTestDB     = 1
	redisTestPrefix = "throttled:rueidis:"
)

func getClient() (rueidis.Client, error) {
	return rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{":6379"},
	})
}

func TestRedisStore(t *testing.T) {
	c, st := setupRedis(t, 0)
	defer c.Close()
	defer clearRedis(c)

	clearRedis(c)
	storetest.TestGCRAStoreCtx(t, st)
	storetest.TestGCRAStoreTTLCtx(t, st)
}

func BenchmarkRedisStore(b *testing.B) {
	c, st := setupRedis(b, 0)
	defer c.Close()
	defer clearRedis(c)

	storetest.BenchmarkGCRAStoreCtx(b, st)
}

func clearRedis(c rueidis.Client) error {
	ctx := context.Background()
	keys, err := c.Do(ctx, c.B().Keys().Pattern(redisTestPrefix+"*").Build()).AsStrSlice()
	if err != nil {
		return err
	}

	if err = c.Do(ctx, c.B().Del().Key(keys...).Build()).Error(); err != nil {
		return err
	}

	return nil
}

func setupRedis(tb testing.TB, ttl time.Duration) (rueidis.Client, throttled.GCRAStoreCtx) {
	ctx := context.Background()
	c, err := getClient()
	if err != nil {
		tb.Fatal(err)
	}

	if err := c.Do(ctx, c.B().Ping().Build()).Error(); err != nil {
		c.Close()
		tb.Skip("redis server not available on localhost port 6379")
	}

	if err := c.Do(ctx, c.B().Select().Index(redisTestDB).Build()).Error(); err != nil {
		c.Close()
		tb.Fatal(err)
	}

	st := ratelimit.New(context.Background(), c, redisTestPrefix)

	return c, st.(throttled.GCRAStoreCtx)
}
