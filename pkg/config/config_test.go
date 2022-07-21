package config

import (
	"os"
	"testing"

	"github.com/inngest/inngest-cli/pkg/config/registration"
	inmemorydatastore "github.com/inngest/inngest-cli/pkg/coredata/inmemory"
	"github.com/inngest/inngest-cli/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest-cli/pkg/execution/driver/httpdriver"
	"github.com/inngest/inngest-cli/pkg/execution/queue/inmemoryqueue"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
	"github.com/inngest/inngest-cli/pkg/execution/state/redis_state"
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
		CoreAPI: CoreAPI{
			Addr: "0.0.0.0",
			Port: 8300,
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
		DataStore: DataStore{
			Service: DataStoreService{
				Backend:  "inmemory",
				Concrete: &inmemorydatastore.Config{},
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
		{
			name: "redis state config, with env vars",
			input: []byte(`package main

import (
	config "inngest.com/defs/config"
)

config.#Config & {
  state: {
    service: {
      backend: "redis"
      host: "${TEST_ENV}"
    }
  }
}
`),
			config: func() *Config {
				c := defaultConfig()
				// Valid JSON
				c.State.Service.Backend = "redis"
				c.State.Service.Concrete = &redis_state.Config{
					Host:      "test-env",
					Port:      6379,
					KeyPrefix: "inngest:state",
				}
				return c
			},

			post: func(t *testing.T, c *Config) {
				redis, ok := c.State.Service.Concrete.(*redis_state.Config)
				require.True(t, ok)
				require.EqualValues(t, &redis_state.Config{
					Host:      "test-env",
					Port:      6379,
					KeyPrefix: "inngest:state",
				}, redis)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			os.Clearenv()
			os.Setenv("TEST_ENV", "test-env")

			config, err := Parse(test.input)
			require.Equal(t, test.err, err)
			require.EqualValues(t, test.config(), config)

			if test.post != nil {
				test.post(t, config)
			}
		})
	}
}
