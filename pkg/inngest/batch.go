package inngest

import "fmt"

func NewEventBatchConfig(data map[string]any) (*EventBatchConfig, error) {
	if data == nil {
		return nil, nil
	}

	config := &EventBatchConfig{}

	size, ok := data["maxSize"].(int)
	if !ok {
		return nil, fmt.Errorf("unexpected type for MaxSize: %v", size)
	}
	config.MaxSize = size

	timeout, ok := data["timeout"].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected type for Timeout: %v", timeout)
	}
	config.Timeout = timeout

	// TODO: validate timeout expression

	return config, nil
}

// EventBatchConfig represents how a function would expect
// a list of events to look like for consumption
//
// A batch of events will be invoked if one of the following
// is fulfilled
// - The batch is full
// - The time to wait is up
type EventBatchConfig struct {
	// MaxSize is the maximum number of events that can be
	// included in a batch
	MaxSize int `json:"maxSize"`

	// Timeout is the maximum number of time the batch will
	// wait before being consumed.
	Timeout string `json:"timeout"`
}

func (c EventBatchConfig) IsEnabled() bool {
	return c.MaxSize != 0 && c.Timeout != ""
}
