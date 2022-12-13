package inmemoryqueue

import (
	"sync"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
)

func init() {
	registration.RegisterQueue(func() any { return &Config{} })
}

func New() queue.Queue {
	r := miniredis.NewMiniRedis()
	_ = r.Start()

	rc := redis.NewClient(&redis.Options{
		Addr:     r.Addr(),
		PoolSize: 100,
	})
	go func() {
		for range time.Tick(time.Second) {
			r.FastForward(time.Second)
		}
	}()
	return redis_state.NewQueue(rc, redis_state.WithNumWorkers(100))
}

type Config struct {
	l  sync.Mutex
	rq queue.Queue
}

func (c *Config) QueueName() string { return "inmemory" }

func (c *Config) Queue() (queue.Queue, error) {
	c.l.Lock()
	defer c.l.Unlock()

	if c.rq != nil {
		return c.rq, nil
	}
	c.rq = New()
	return c.rq, nil
}

func (c *Config) Producer() (queue.Producer, error) {
	return c.Queue()
}

func (c *Config) Consumer() (queue.Consumer, error) {
	return c.Queue()
}
