package dockerdriver

import (
	docker "github.com/fsouza/go-dockerclient"
	"github.com/inngest/inngest-cli/pkg/config/registration"
	"github.com/inngest/inngest-cli/pkg/execution/driver"
)

func init() {
	registration.RegisterDriver(func() any { return &Config{} })
}

// Config represents driver configuration for use when configuring hosted
// services via config.cue
type Config struct {
	Host *string
}

func (c Config) NewDriver() (driver.Driver, error) {
	var (
		client *docker.Client
		err    error
	)

	if c.Host == nil {
		if client, err = docker.NewClientFromEnv(); err != nil {
			return nil, err
		}
	} else {
		if client, err = docker.NewClient(*c.Host); err != nil {
			return nil, err
		}
	}

	if err := client.Ping(); err != nil {
		return nil, err
	}

	return &dockerExec{
		client: client,
	}, nil
}

// RuntimeName returns the runtime field that should invoke this driver.
func (Config) RuntimeName() string { return "docker" }

// DriverName returns the name of this driver
func (Config) DriverName() string { return "docker" }
