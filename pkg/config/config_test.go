package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func defaultConfig() *Config {
	base := &Config{
		Log: Log{
			Level:  "info",
			Format: "json",
		},
		EventAPI: EventAPI{
			Addr: "0.0.0.0",
			Port: "8288",
		},
		EventStream: EventStream{
			Service: MessagingService{
				Backend:  "inmemory",
				raw:      json.RawMessage(`{"backend":"inmemory","topic":"events"}`),
				concrete: &InMemoryMessaging{Topic: "events"},
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
  eventStream: {
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
				c.EventStream.Service.raw = []byte(`{"backend":"nats","topic":"nats-events","serverURL":"http://127.0.0.1:4222"}`)
				c.EventStream.Service.Backend = "nats"
				c.EventStream.Service.concrete = &NATSMessaging{
					Topic:     "nats-events",
					ServerURL: "http://127.0.0.1:4222",
				}
				return c
			},

			post: func(t *testing.T, c *Config) {
				nats, err := c.EventStream.Service.NATS()
				require.NoError(t, err)
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
