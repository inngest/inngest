package registration

import (
	"github.com/inngest/inngest-cli/pkg/execution/driver"
)

var (
	// registeredDrivers stores all registered driver configurations.
	registeredDrivers = map[string]interface{}{}
)

func RegisteredDrivers() map[string]interface{} {
	return registeredDrivers
}

func RegisterDriverConfig(c DriverConfig) {
	// Overwrite any previous drivers.
	registeredDrivers[c.DriverName()] = c
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
