package inngest

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/inngest/inngest/pkg/expressions"
	"time"

	"github.com/inngest/inngest/pkg/syscode"
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

	if _, err = time.ParseDuration(config.Timeout); err != nil {
		return nil, fmt.Errorf("failed to parse time duration: %v", err)
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
	Key *string `json:"key,omitempty"`

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

func (c EventBatchConfig) IsValid(ctx context.Context) error {
	if c.MaxSize < 2 {
		return syscode.Error{
			Code:    syscode.CodeBatchSizeInvalid,
			Message: fmt.Sprintf("batch size cannot be smaller than 2: %d", c.MaxSize),
		}
	}

	dur, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return syscode.Error{
			Code:    syscode.CodeBatchTimeoutInvalid,
			Message: fmt.Sprintf("invalid timeout string: %s", c.Timeout),
		}
	}

	if dur < time.Second {
		return syscode.Error{
			Code:    syscode.CodeBatchTimeoutInvalid,
			Message: "batch timeout should be more than 1s",
		}
	}

	if c.Key != nil {
		// Ensure the expression is valid if present.
		if exprErr := expressions.Validate(ctx, *c.Key); exprErr != nil {
			return syscode.Error{
				Code:    syscode.CodeBatchKeyExpressionInvalid,
				Message: fmt.Sprintf("batch key expression is invalid: %s", exprErr),
			}
		}
	}

	return nil
}
