package registration

import (
	"fmt"
)

var (
	// registeredDrivers stores all registered driver configurations.
	registeredDrivers = map[string]interface{}{}
)

func RegisteredDrivers() map[string]interface{} {
	return registeredDrivers
}

func RegisterDriverConfig(c DriverConfig) error {
	if _, ok := registeredDrivers[c.DriverName()]; ok {
		return fmt.Errorf("driver already registered: %s", c.DriverName())
	}
	registeredDrivers[c.DriverName()] = c
	return nil
}

// DriverConfig is an interface used to determine driver config structs.
type DriverConfig interface {
	// json.Unmarshaler

	// DriverName returns the name of the specific driver.
	DriverName() string
	// RuntimeName returns the name of the runtime used within the
	// driver implemetation and step configuration.
	RuntimeName() string
}
