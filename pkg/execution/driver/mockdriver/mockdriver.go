package mockdriver

import (
	"context"
	"sync"

	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
)

func init() {
	registration.RegisterDriver(func() any { return &Config{} })
}

const RuntimeName = "mock"

type Mock struct {
	// DynamicResponses allows users to specify a function which allows
	// steps to return different data on each execution invocation.
	DynamicResponses func(context.Context, sv2.Metadata, queue.Item, inngest.Edge, inngest.Step, int, int) map[string]state.DriverResponse

	// Responses stores the responses that a driver should return.
	Responses map[string]state.DriverResponse

	// Errors stores which steps should return with a driver error, as if
	// the step wasn't executed.
	Errors map[string]error

	RuntimeName string

	// Executed stores which actions were "executed"
	Executed map[string]inngest.Step

	lock sync.RWMutex
}

// Name fulfiils the inngest.Runtime interface.
func (m *Mock) Name() string {
	if m.RuntimeName == "" {
		return RuntimeName
	}
	// Allow mocking other arbitrary runtime names.
	return m.RuntimeName
}

func (m *Mock) Execute(ctx context.Context, sl sv2.StateLoader, s sv2.Metadata, item queue.Item, edge inngest.Edge, step inngest.Step, idx, attempt int) (*state.DriverResponse, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.Executed == nil {
		m.Executed = map[string]inngest.Step{}
	}
	m.Executed[step.ID] = step

	resp := m.Responses
	if m.DynamicResponses != nil {
		resp = m.DynamicResponses(ctx, s, item, edge, step, idx, attempt)
	}
	response := resp[step.ID]
	err := m.Errors[step.ID]

	return &response, err
}

func (m *Mock) ExecutedLen() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.Executed)
}

// Config represents driver configuration for use when configuring hosted
// services via config.cue
type Config struct {
	l         sync.Mutex
	Responses map[string]state.DriverResponse
	// DynamicResponses allows users to specify a function which allows
	// steps to return different data on each execution invocation.
	DynamicResponses func(context.Context, sv2.Metadata, queue.Item, inngest.Edge, inngest.Step, int, int) map[string]state.DriverResponse
	// driver stores the driver once, as a singleton per config instance.
	driver driver.DriverV1
	Driver string
}

// RuntimeName returns the runtime field that should invoke this driver.
func (c *Config) RuntimeName() string { return c.Driver }

// DriverName returns the name of this driver
func (*Config) DriverName() string { return RuntimeName }

func (c *Config) NewDriver(opts ...registration.NewDriverOpts) (driver.DriverV1, error) {
	c.l.Lock()
	defer c.l.Unlock()

	if c.driver == nil {
		c.driver = &Mock{
			Responses:        c.Responses,
			DynamicResponses: c.DynamicResponses,
			RuntimeName:      c.Driver,
		}
	}
	return c.driver, nil
}
