package devserver

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/realtime"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/redis/rueidis"
	"go.uber.org/fx"
)

// redisClients are the Redis clients of the server.  without
// StartOpts.RedisURI, every client except realtimeSub has its own in-memory
// Redis server.
type redisClients struct {
	sharded   rueidis.Client
	unsharded rueidis.Client
	connect   rueidis.Client

	// realtime has one client to publish and one to subscribe, because a
	// Redis client cannot publish after it subscribes.
	realtimePub rueidis.Client
	realtimeSub rueidis.Client

	// inMemory are the in-memory Redis servers.  it is empty with
	// StartOpts.RedisURI.
	inMemory []*inMemoryRedis
}

// newRedisClients connects to StartOpts.RedisURI, or starts in-memory Redis
// servers.  the clients and servers close when the app stops.
func newRedisClients(opts StartOpts, log logger.Logger, lifecycle fx.Lifecycle) (*redisClients, error) {
	var (
		clients *redisClients
		err     error
	)
	if opts.RedisURI != "" {
		clients, err = externalRedisClients(opts.RedisURI, log)
	} else {
		clients, err = inMemoryRedisClients()
	}
	if err != nil {
		return nil, err
	}
	lifecycle.Append(fx.StopHook(clients.close))
	return clients, nil
}

func externalRedisClients(uri string, log logger.Logger) (*redisClients, error) {
	// Mask Redis URI credentials before logging
	loggedURI := ""
	if u, err := url.Parse(uri); err == nil {
		loggedURI = u.Redacted()
	}
	log.Info("using external redis", "url", loggedURI)

	c := &redisClients{}
	for _, client := range []*rueidis.Client{
		&c.sharded,
		&c.unsharded,
		&c.connect,
		&c.realtimePub,
		&c.realtimeSub,
	} {
		opt, err := redisOptions(uri)
		if err != nil {
			c.close()
			return nil, fmt.Errorf("could not create redis options: %w", err)
		}
		if *client, err = rueidis.NewClient(opt); err != nil {
			c.close()
			return nil, fmt.Errorf("error creating redis client: %w", err)
		}
	}
	return c, nil
}

func inMemoryRedisClients() (*redisClients, error) {
	c := &redisClients{}
	for _, client := range []*rueidis.Client{
		&c.sharded,
		&c.unsharded,
		&c.connect,
	} {
		server, err := c.startInMemory()
		if err != nil {
			c.close()
			return nil, err
		}
		if *client, err = server.client(); err != nil {
			c.close()
			return nil, err
		}
	}

	realtime, err := c.startInMemory()
	if err != nil {
		c.close()
		return nil, err
	}
	for _, client := range []*rueidis.Client{
		&c.realtimePub,
		&c.realtimeSub,
	} {
		if *client, err = realtime.client(); err != nil {
			c.close()
			return nil, err
		}
	}
	return c, nil
}

func (c *redisClients) startInMemory() (*inMemoryRedis, error) {
	server, err := startInMemoryRedis()
	if err != nil {
		return nil, err
	}
	c.inMemory = append(c.inMemory, server)
	return server, nil
}

// flush deletes every key in the in-memory Redis servers.
func (c *redisClients) flush() {
	for _, server := range c.inMemory {
		server.FlushAll()
	}
}

func (c *redisClients) close() {
	for _, client := range []rueidis.Client{
		c.sharded,
		c.unsharded,
		c.connect,
		c.realtimePub,
		c.realtimeSub,
	} {
		if client != nil {
			client.Close()
		}
	}
	for _, server := range c.inMemory {
		server.close()
	}
}

// redisOptions parses a Redis URI into client options.
func redisOptions(redisURI string) (rueidis.ClientOption, error) {
	opt, err := rueidis.ParseURL(redisURI)
	if err != nil {
		return rueidis.ClientOption{}, fmt.Errorf("error parsing redis uri: invalid format")
	}

	// Fix for Redis Sentinel authentication: rueidis.ParseURL correctly identifies
	// Sentinel configurations but fails to populate Sentinel credentials.
	// When a master_set is configured but Sentinel credentials are empty,
	// copy the main credentials to Sentinel authentication.
	if opt.Sentinel.MasterSet != "" && opt.Sentinel.Username == "" && opt.Username != "" {
		opt.Sentinel.Username = opt.Username
		opt.Sentinel.Password = opt.Password
	}

	// Set default overrides
	opt.DisableCache = true
	opt.BlockingPoolSize = consts.RedisBlockingPoolSize

	return opt, nil
}

// inMemoryRedis is a miniredis server.  miniredis does not move its clock on
// its own, so a goroutine moves it forward every DefaultTickDuration.
// without it, keys with a TTL never expire.
type inMemoryRedis struct {
	*miniredis.Miniredis
	stop chan struct{}
}

func startInMemoryRedis() (*inMemoryRedis, error) {
	server := miniredis.NewMiniRedis()
	if err := server.Start(); err != nil {
		return nil, fmt.Errorf("error starting in-memory redis: %w", err)
	}
	r := &inMemoryRedis{
		Miniredis: server,
		stop:      make(chan struct{}),
	}
	go r.fastForward()
	return r, nil
}

func (r *inMemoryRedis) fastForward() {
	ticker := time.NewTicker(DefaultTickDuration)
	defer ticker.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-ticker.C:
			r.FastForward(DefaultTickDuration)
		}
	}
}

func (r *inMemoryRedis) client() (rueidis.Client, error) {
	return rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
}

func (r *inMemoryRedis) close() {
	close(r.stop)
	r.Close()
}

// redisState are the run state, pause, and queue clients on Redis.
type redisState struct {
	fx.Out

	Unsharded  *redis_state.UnshardedClient
	Sharded    *redis_state.ShardedClient
	Pauses     pauses.Manager
	State      state.Manager
	RunService sv2.RunService
}

func newRedisState(ctx context.Context, clients *redisClients) (redisState, error) {
	unsharded := redis_state.NewUnshardedClient(clients.unsharded, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	sharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unsharded,
		FunctionRunStateClient: clients.sharded,
		StateDefaultKey:        redis_state.StateDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
		BatchClient:            clients.sharded,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
	})

	pauseMgr := pauses.NewPauseStoreManager(unsharded)
	sm, err := redis_state.New(ctx, redis_state.WithShardedClient(sharded), redis_state.WithPauseDeleter(pauseMgr))
	if err != nil {
		return redisState{}, err
	}

	return redisState{
		Unsharded:  unsharded,
		Sharded:    sharded,
		Pauses:     pauseMgr,
		State:      sm,
		RunService: redis_state.MustRunServiceV2(sm),
	}, nil
}

func newBroadcaster(clients *redisClients) realtime.Broadcaster {
	return realtime.NewRedisBroadcaster(clients.realtimePub, clients.realtimeSub)
}
