package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest-cli/pkg/config/registration"
)

// Load loads the configu from the given locations in order.  If locs is empty,
// we use the default locations of "./inngest.(cue|json)" and "/etc/inngest.(cue|json)".
func Load(ctx context.Context, locs ...string) (*Config, error) {
	return loadAll(ctx, locs...)
}

func Default(ctx context.Context) (*Config, error) {
	return Parse(nil)
}

// Config represents configuration for running the Inngest services.
type Config struct {
	// Log configures the logger used within Inngest services.
	Log Log
	// EventAPI configures the event API service.
	EventAPI EventAPI
	// CoreAPI configures the core API service.
	CoreAPI CoreAPI
	// Execution configures the executor, which invokes actions and steps.
	Execution Execution
	// EventAPI configures the event stream, which connects events to the execution engine.
	EventStream EventStream
	// Queue configures the backing queue, used to enqueue function steps
	// for execution.
	Queue Queue
	// State configures the execution state store.
	State State
	// DataStore configures the persisted data for the system
	DataStore DataStore
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
	Port int
	// MaxSize represents the max size of events ingested, in bytes.
	MaxSize int
}

type CoreAPI struct {
	// Addr is the IP to bind to, eg. "0.0.0.0" or "127.0.0.1"
	Addr string
	// Port is the port to use, defaulting to 8288.
	Port int
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

type DataStore struct {
	Service DataStoreService
}

type Execution struct {
	// Drivers represents all drivers enabled.
	Drivers   map[string]registration.DriverConfig
	LogOutput bool `json:"logOutput"`
}

func (e *Execution) UnmarshalJSON(byt []byte) error {
	type drivers struct {
		Drivers   map[string]unmarshalDriver
		LogOutput bool
	}
	names := &drivers{}
	if err := json.Unmarshal(byt, names); err != nil {
		return err
	}

	e.Drivers = map[string]registration.DriverConfig{}
	e.LogOutput = names.LogOutput

	for runtime, driver := range names.Drivers {
		f, ok := registration.RegisteredDrivers()[driver.Name]
		if !ok {
			return fmt.Errorf("unknown driver: %s", driver.Name)
		}

		iface := f()
		if err := json.Unmarshal(driver.Raw, iface); err != nil {
			return err
		}
		res, _ := iface.(registration.DriverConfig)
		if runtime != res.RuntimeName() {
			// Ensure the driver can run the given runtime.
			return fmt.Errorf("driver %s is not valid for runtime %s", driver.Name, runtime)
		}

		e.Drivers[runtime] = res
	}

	return nil
}

// unmarshalDriver is used to help unmarshal drivers into their
// concrete structs.
type unmarshalDriver struct {
	Name string
	Raw  json.RawMessage
}

func (u *unmarshalDriver) UnmarshalJSON(b []byte) error {
	type driver struct {
		Name string
	}
	d := &driver{}
	if err := json.Unmarshal(b, d); err != nil {
		return err
	}
	u.Name = d.Name
	u.Raw = b
	return nil
}
