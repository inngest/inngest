package broker

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	defaultNatsURL = "nats://127.0.0.1:4222"
)

type NatsConnOpt struct {
	// Name of this client
	Name string
	// Comma delimited URLs of NATS instances
	URLS string
	// JetStream signals that the connection should be using the JetStream APIs instead of the raw
	// conn itself
	JetStream bool
	// Consumers specifics the list of existing consumers to initialize
	//   Key: stream name
	//   Value: list of consumer names
	//
	// This will be combined as key-value for each consumer in the map, therefore if there results
	// with duplicates, the last key-value string will likely take precendence
	Consumers map[string][]string
	Opts      []nats.Option
}

// NOTE: probably should implement some kind of interface
//
// NatsConnector represents a valid nats connection that can be used for publishing and consuming
// messages.
type NatsConnector struct {
	// connWg is used for waiting connections to be drained before closing the connection off.
	// this is required to make sure data in buffer are flushed without losing them
	connWg *sync.WaitGroup
	// conn represents a successfully connected NATS connection
	conn *nats.Conn
	// js represents the jetstream API on top of the NATS connection
	js jetstream.JetStream
	// consumers stores the list of consumers used for this connection.
	// this is only available when jetstream is enabled.
	//
	// the key is a combination of stream name + consumer name (e.g. trace + trace-consumer = "trace-trace-consumer")
	consumers map[string]jetstream.Consumer
	// The size of the buffer allowed for pending messages on async publish
	BufferSize int
}

func NewNATSConnector(ctx context.Context, opts NatsConnOpt) (*NatsConnector, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("error retrieving hostname: %w", err)
	}
	name := fmt.Sprintf("%s-%s", opts.Name, host)

	urls := defaultNatsURL
	if len(opts.URLS) > 0 {
		urls = opts.URLS
	}

	wg := sync.WaitGroup{}

	// merge provided options
	nopts := []nats.Option{
		nats.Name(name),
		nats.ClosedHandler(func(c *nats.Conn) {
			wg.Done()
		}),
	}
	nopts = append(nopts, opts.Opts...)

	conn, err := nats.Connect(urls, nopts...)
	if err != nil {
		return nil, fmt.Errorf("error connecting to NATS: %w", err)
	}
	wg.Add(1)

	logger.StdlibLogger(ctx).Info("established connection with NATS server",
		"urls", opts.URLS,
		"name", name,
	)

	c := &NatsConnector{
		connWg: &wg,
		conn:   conn,
	}

	if opts.JetStream {
		pending := 10_000
		if p, err := strconv.Atoi(os.Getenv("NATS_JS_MAX_PENDING")); err == nil && p > 0 {
			pending = p
		}
		c.BufferSize = pending

		// NOTE: should there be some default jetstream options?
		js, err := jetstream.New(conn,
			jetstream.WithPublishAsyncMaxPending(pending),
		)
		if err != nil {
			return nil, fmt.Errorf("error initializing jetstream API: %w", err)
		}
		c.js = js

		// initialize the consumers
		if opts.Consumers != nil {
			c.consumers = map[string]jetstream.Consumer{}

			for stream, consumers := range opts.Consumers {
				for _, consumer := range consumers {
					cons, err := js.Consumer(ctx, stream, consumer)
					if err != nil {
						return nil, fmt.Errorf("error initializing consumer: %w", err)
					}

					key := fmt.Sprintf("%s-%s", stream, consumer)
					c.consumers[key] = cons
				}
			}
		}
	}

	return c, nil
}

// Publish submits the provided data. It'll use jetstream if it's available, or default
// to the core NATS connection.
func (c *NatsConnector) Publish(ctx context.Context, sub string, data []byte) error {
	if c.js != nil {
		_, err := c.js.Publish(ctx, sub, data)
		return err
	}
	return c.conn.Publish(sub, data)
}

func (c *NatsConnector) JSConn() (jetstream.JetStream, error) {
	if c.js == nil {
		return nil, fmt.Errorf("jetstream connection not available")
	}
	return c.js, nil
}

// Consumer returns the specified consumer based on stream name and consumer name, if it's initialized
func (c *NatsConnector) Consumer(ctx context.Context, stream, consumer string) (jetstream.Consumer, error) {
	if c.consumers == nil {
		return nil, fmt.Errorf("consumers are not initialized")
	}

	key := fmt.Sprintf("%s-%s", stream, consumer)
	if cons, ok := c.consumers[key]; ok {
		return cons, nil
	}

	return nil, fmt.Errorf("no consumer named '%s' initialized for stream: '%s'", consumer, stream)
}

// Shutdown drains the connection to flush data in buffer before fully closing the
// connection
func (c *NatsConnector) Shutdown(ctx context.Context) error {
	if c.conn == nil {
		// nothing to do
		return nil
	}

	if err := c.conn.Drain(); err != nil {
		return err
	}
	c.connWg.Wait() // wait for the drain to complete

	return nil
}
