package httpdriver

import (
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/exechttp"
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

func (c Config) NewDriver(opts ...registration.NewDriverOpts) (driver.DriverV1, error) {
	var skey []byte
	requireLocalSigningKey := false
	client := exechttp.RequestExecutor(defaultClient)
	if len(opts) > 0 {
		if opts[0].LocalSigningKey != nil {
			skey = []byte(*opts[0].LocalSigningKey)
		}

		if opts[0].RequireLocalSigningKey {
			requireLocalSigningKey = true
		}

		if opts[0].HTTPClient != nil {
			client = opts[0].HTTPClient
		}
	}

	return &executor{
		Client:                 client,
		localSigningKey:        skey,
		requireLocalSigningKey: requireLocalSigningKey,
	}, nil
}
