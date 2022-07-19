package inmemoryqueue

import (
	"context"
	"sync"
	"time"

	"github.com/inngest/inngest-cli/pkg/config/registration"
	"github.com/inngest/inngest-cli/pkg/execution/queue"
)

func init() {
	registration.RegisterQueue(func() any { return &Config{} })
}

type Config struct {
	l sync.Mutex

	// mem stores a pointer to memory, acting as a singleton per Config
	// instance created.
	//
	// We need to create "local" singletons per config struct in order to
	// parallelize and unit test these correctly;  with a global singleton
	// we may have unexpected data within our in-memory queue during parallel
	// tests.
	mem *mem
}

func (c *Config) QueueName() string { return "inmemory" }

func (c *Config) Queue() (queue.Queue, error) {
	c.l.Lock()
	defer c.l.Unlock()

	if c.mem == nil {
		c.mem = &mem{
			q: make(chan queue.Item),
		}
	}

	return c.mem, nil
}

func (c *Config) Producer() (queue.Producer, error) {
	return c.Queue()
}

func (c *Config) Consumer() (queue.Consumer, error) {
	return c.Queue()
}

// MemoryQueue is a simplistic, **non production ready** queue for processing steps
// of functions, keepign the queue in-memory with zero persistence.  It is used
// to simulate a production environment for local testing.
type MemoryQueue interface {
	queue.Queue
	// Channel returns a channel which receives available jobs on the queue.
	// This is helpful during testing.
	Channel() chan queue.Item
}

type mem struct {
	q chan queue.Item
}

func (m *mem) Enqueue(ctx context.Context, item queue.Item, at time.Time) error {
	go func() {
		<-time.After(time.Until(at))
		m.q <- item
	}()
	return nil
}

func (m *mem) Channel() chan queue.Item {
	return m.q
}

func (m *mem) Run(ctx context.Context, f func(context.Context, queue.Item) error) error {
	for {
		select {
		case <-ctx.Done():
			// We are shutting down.
			return nil
		case item := <-m.q:
			if err := f(ctx, item); err != nil {
				return err
			}
		}

	}
}
