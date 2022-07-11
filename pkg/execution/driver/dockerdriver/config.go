package dockerdriver

import (
	"github.com/inngest/inngest-cli/pkg/config/registration"
)

func init() {
	registration.RegisterDriverConfig(&Config{})
}

// Config represents driver configuration for use when configuring hosted
// services via config.cue
type Config struct {
	Host *string
}

// RuntimeName returns the runtime field that should invoke this driver.
func (Config) RuntimeName() string { return "docker" }

// DriverName returns the name of this driver
func (Config) DriverName() string { return "docker" }
