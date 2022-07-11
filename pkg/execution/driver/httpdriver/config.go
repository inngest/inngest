package httpdriver

import (
	"github.com/inngest/inngest-cli/pkg/execution/driver"

	"github.com/inngest/inngest-cli/pkg/config/registration"
)

func init() {
	registration.RegisterDriverConfig(&Config{})
}

// Config represents driver configuration for use when configuring hosted
// services via config.cue
type Config struct{}

// RuntimeName returns the runtime field that should invoke this driver.
func (Config) RuntimeName() string { return "http" }

// DriverName returns the name of this driver
func (Config) DriverName() string { return "http" }

func (Config) UnmarshalJSON(b []byte) error { return nil }

func (c Config) NewDriver() (driver.Driver, error) {
	return DefaultExecutor, nil
}
