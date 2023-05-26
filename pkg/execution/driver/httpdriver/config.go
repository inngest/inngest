package httpdriver

import (
	"github.com/inngest/inngest/pkg/execution/driver"

	"github.com/inngest/inngest/pkg/config/registration"
)

func init() {
	registration.RegisterDriver(func() any { return &Config{} })
}

// Config represents driver configuration for use when configuring hosted
// services via config.cue
type Config struct {
	SigningKey string
	Timeout    int
}

// RuntimeName returns the runtime field that should invoke this driver.
func (Config) RuntimeName() string { return "http" }

// DriverName returns the name of this driver
func (Config) DriverName() string { return "http" }

func (c Config) NewDriver() (driver.Driver, error) {
	return DefaultExecutor, nil
}
