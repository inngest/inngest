package mockdriver

import (
	"context"
	"sync"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/execution/driver"
	"github.com/inngest/inngest-cli/pkg/execution/state"
)

const RuntimeName = "mock"

type Mock struct {
	RuntimeName string

	// Responses stores the responses that a driver should return.
	Responses map[string]driver.Response
	// Errors stores which steps should return with a driver error, as if
	// the step wasn't executed.
	Errors map[string]error

	// Executed stores which actions were "executed"
	Executed map[string]inngest.ActionVersion

	lock sync.RWMutex
}

// RuntimeType fulfiils the inngest.Runtime interface.
func (m *Mock) RuntimeType() string {
	if m.RuntimeName == "" {
		return RuntimeName
	}
	// Allow mocking other arbitrary runtime names.
	return m.RuntimeName
}

func (m *Mock) Execute(ctx context.Context, state state.State, action inngest.ActionVersion, step inngest.Step) (*driver.Response, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.Executed == nil {
		m.Executed = map[string]inngest.ActionVersion{}
	}

	m.Executed[step.ID] = action

	response := m.Responses[step.ID]
	err := m.Errors[step.ID]
	return &response, err
}

func (m *Mock) ExecutedLen() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.Executed)
}
