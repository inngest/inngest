package config

import (
	"context"
	"encoding/json"
	"fmt"
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
	Log Log
	// EventAPI configures the event API service.
	EventAPI EventAPI
	//
	Execution Execution
	// EventAPI configures the event stream, which connects events to the execution engine.
	EventStream EventStream
	// Queue configures the backing queue, used to enqueue function steps
	// for execution.
	Queue Queue
	// State configures the execution state store.
	State State
}

// Log configures the logger used within Inngest services.
type Log struct {
	// Level configures the log level.  Valid choices are:
	// "trace", "debug", "info", "warn", or "error".  The default
	// is "info".
	Level string
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
	// MaxSize represents the max size of events ingested, in bytes.
	MaxSize int
}

// EventAPI configures the event stream, which connects events to the execution engine.
type EventStream struct {
	Service MessagingService
}

type Queue struct {
	Service QueueService
}

type State struct {
	Service StateService
}

type Execution struct {
	// Drivers represents all drivers enabled.
	Drivers map[string]DriverConfig
}

func (e *Execution) UnmarshalJSON(byt []byte) error {
	type drivers struct {
		Drivers map[string]unmarshalDriver
	}
	names := &drivers{}
	if err := json.Unmarshal(byt, names); err != nil {
		return err
	}

	e.Drivers = map[string]DriverConfig{}

	// TODO: Move to registering driver config in init, vs
	// hard coding.
	for runtime, driver := range names.Drivers {
		var def interface{}

		switch driver.Name {
		case "docker":
			def = &DockerDriver{}
		case "mock":
			def = &MockDriver{}
		case "http":
			def = &HTTPDriver{}
		default:
			return fmt.Errorf("unknown driver name: %s", driver.Name)
		}

		if err := json.Unmarshal(driver.Raw, def); err != nil {
			return err
		}

		e.Drivers[runtime] = def.(DriverConfig)
	}

	return nil
}
