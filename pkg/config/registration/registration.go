package registration

import (
	"context"

	"github.com/inngest/inngest-cli/pkg/execution/driver"
	"github.com/inngest/inngest-cli/pkg/execution/queue"
	"github.com/inngest/inngest-cli/pkg/execution/state"
)

var (
	// registeredDrivers stores all registered driver configurations.
	registeredDrivers = map[string]any{}

	registeredQueues = map[string]any{}

	registeredStates = map[string]any{}
)

func RegisteredDrivers() map[string]any {
	return registeredDrivers
}

func RegisteredQueues() map[string]any {
	return registeredQueues
}

func RegisteredStates() map[string]any {
	return registeredStates
}

// RegisterDriver registers a driver's configuration for use with
// self-hosted services.
func RegisterDriver(c DriverConfig) {
	// Overwrite any previous drivers.
	registeredDrivers[c.DriverName()] = c
}

// RegisterQueue registers a queue for use within self hosted services/
func RegisterQueue(c QueueConfig) {
	registeredQueues[c.QueueName()] = c
}

// RegisterState registers a state manager for use within self hosted services/
func RegisterState(c StateConfig) {
	registeredStates[c.StateName()] = c
}

// DriverConfig is an interface used to determine driver config structs.
type DriverConfig interface {
	NewDriver() (driver.Driver, error)

	// DriverName returns the name of the specific driver.
	DriverName() string
	// RuntimeName returns the name of the runtime used within the
	// driver implemetation and step configuration.
	RuntimeName() string
}

type QueueConfig interface {
	QueueName() string
	Queue() (queue.Queue, error)
	Producer() (queue.Producer, error)
	Consumer() (queue.Consumer, error)
}

type StateConfig interface {
	StateName() string
	Manager(context.Context) (state.Manager, error)
}
