package config

import (
	"context"
	"encoding/json"
)

// Load loads the configu from the given locations in order.  If locs is empty,
// we use the default locations of "./inngest.(cue|json)" and "/etc/inngest.(cue|json)".
func Load(ctx context.Context, locs ...string) (*Config, error) {
	return loadAll(ctx, locs...)
}

func Default(ctx context.Context) (*Config, error) {
	return parse(nil)
}

// Config represents configuration for running the Inngest services.
type Config struct {
	// Log configures the logger used within Inngest services.
	Log Log `json:"log"`
	// EventAPI configures the event API service.
	EventAPI EventAPI
	// EventAPI configures the event stream, which connects events to the execution engine.
	EventStream EventStream
}

// Log configures the logger used within Inngest services.
type Log struct {
	// Level configures the log level.  Valid choices are:
	// "trace", "debug", "info", "warn", or "error".  The default
	// is "info".
	Level string `json:"level"`
	// Format configures the log format.  Currently, only "json"
	// is supported and is the default.
	Format string
}

// EventAPI configures the event API service.
type EventAPI struct {
	// Addr is the IP to bind to, eg. "0.0.0.0" or "127.0.0.1"
	Addr string
	// Port is the port to use, defaulting to 8288.
	Port string
}

// EventAPI configures the event stream, which connects events to the execution engine.
type EventStream struct {
	Service MessagingService
}

type MessagingService struct {
	Backend string
	raw     json.RawMessage
}

func (m *MessagingService) UnmarshalJSON(byt []byte) error {
	m.raw = byt
	type svc struct {
		Backend string
	}
	data := &svc{}
	err := json.Unmarshal(byt, data)
	m.Backend = data.Backend
	return err
}

type InMemoryMessaging struct {
	Topic string
}

type NATSMessaging struct {
	Topic     string
	ServerURL string
}
