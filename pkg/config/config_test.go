package config

import (
	"testing"

	"github.com/inngest/inngest-cli/pkg/config/registration"
	"github.com/inngest/inngest-cli/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest-cli/pkg/execution/driver/httpdriver"
	"github.com/inngest/inngest-cli/pkg/execution/queue/inmemoryqueue"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
	"github.com/stretchr/testify/require"
)

func defaultConfig() *Config {
	// The default config is created via Cue's default parameters.
	base := &Config{
		Log: Log{
			Level:  "info",
			Format: "json",
		},
		EventAPI: EventAPI{
			Addr:    "0.0.0.0",
			Port:    8288,
			MaxSize: 524288,
		},
		Execution: Execution{
			Drivers: map[string]registration.DriverConfig{
				"docker": &dockerdriver.Config{},
				"http":   &httpdriver.Config{},
			},
		},
		EventStream: EventStream{
			Service: MessagingService{
				Backend:  "inmemory",
				Concrete: &InMemoryMessaging{Topic: "events"},
			},
		},
		Queue: Queue{
			Service: QueueService{
				Backend:  "inmemory",
				Concrete: &inmemoryqueue.Config{},
			},
		},
		State: State{
			Service: StateService{
				Backend:  "inmemory",
				Concrete: &inmemory.Config{},
			},
		},
	}

	return base
}

func TestParse(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		config func() *Config
		err    error
		// post is a func that can run after unmarshalling, to add any custom predicates.
		post func(t *testing.T, c *Config)
	}{
		{
			name:   "none",
			config: defaultConfig,
		},
		{
			name:   "empty json",
			input:  []byte(`{}`),
			config: defaultConfig,
		},
		{
			name: "overriding log level as cue, without imports",
			input: []byte(`{
	log: { level: "warn" }
}
`),
			config: func() *Config {
				c := defaultConfig()
				c.Log.Level = "warn"
				return c
			},
		},
		{
			name: "overriding log level, with package imports",
			input: []byte(`package main

import (
	config "inngest.com/defs/config"
)

config.#Config & {
	log: { level: "warn" }
}
`),
			config: func() *Config {
				c := defaultConfig()
				c.Log.Level = "warn"
				return c
			},
		},
		{
			name: "nats config, as cue",
			input: []byte(`package main

import (
	config "inngest.com/defs/config"
)

config.#Config & {
  eventstream: {
    service: {
      backend: "nats"
      topic: "nats-events"
      serverURL: "http://127.0.0.1:4222"
    }
  }
}
`),
			config: func() *Config {
				c := defaultConfig()
				// Valid JSON
				c.EventStream.Service.Backend = "nats"
				c.EventStream.Service.Concrete = &NATSMessaging{
					Topic:     "nats-events",
					ServerURL: "http://127.0.0.1:4222",
				}
				return c
			},

			post: func(t *testing.T, c *Config) {
				nats, ok := c.EventStream.Service.Concrete.(*NATSMessaging)
				require.True(t, ok)
				require.EqualValues(t, &NATSMessaging{
					Topic:     "nats-events",
					ServerURL: "http://127.0.0.1:4222",
				}, nats)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := parse(test.input)
			require.Equal(t, test.err, err)
			require.EqualValues(t, test.config(), config)

			if test.post != nil {
				test.post(t, config)
			}
		})
	}
}
