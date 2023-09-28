package inngest

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
)

func NewEventBatchConfig(conf map[string]any) (*EventBatchConfig, error) {
	if conf == nil {
		return nil, nil
	}

	data, err := json.Marshal(conf)
	if err != nil {
		return nil, err
	}

	config := &EventBatchConfig{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to decode batch config: %v", err)
	}

	if config.MaxSize <= 0 {
		config.MaxSize = consts.DefaultBatchSize
	}

	dur, err := time.ParseDuration(config.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time duration: %v", err)
	}
	if dur > consts.MaxBatchTimeout {
		config.Timeout = "60s"
	}

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
	// batch of 1 should not be considered a batch
	return c.MaxSize > 1 && c.Timeout != ""
}

func (c EventBatchConfig) IsValid() error {
	if c.MaxSize < 2 {
		return fmt.Errorf("batch size cannot be smaller than 2: %d", c.MaxSize)
	}

	if _, err := time.ParseDuration(c.Timeout); err != nil {
		return fmt.Errorf("invalid timeout string: %v", err)
	}

	return nil
}
