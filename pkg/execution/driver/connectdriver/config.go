package connectdriver

import (
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/execution/driver"
)

func init() {
	registration.RegisterDriver(func() any { return &Config{} })
}

// Config represents driver configuration for use when configuring hosted
// services via config.cue
type Config struct{}

// RuntimeName returns the runtime field that should invoke this driver.
func (Config) RuntimeName() string { return "connect" }

// DriverName returns the name of this driver
func (Config) DriverName() string { return "connect" }

func (c Config) NewDriver(opts ...registration.NewDriverOpts) (driver.DriverV1, error) {
	e := &executor{}
	if len(opts) > 0 {
		e.forwarder = opts[0].ConnectForwarder
		e.tracer = opts[0].ConditionalTracer
	}

	return e, nil
}
