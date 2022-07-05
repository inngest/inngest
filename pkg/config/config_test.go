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
				Backend: "inmemory",
				raw:     json.RawMessage(`{"backend":"inmemory","topic":"events"}`),
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
	}{
		{
			name:   "none",
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
			name:   "empty json",
			input:  []byte(`{}`),
			config: defaultConfig,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := parse(test.input)
			require.Equal(t, test.err, err)
			require.EqualValues(t, test.config(), config)
		})
	}
}
