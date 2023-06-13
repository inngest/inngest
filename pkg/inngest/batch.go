package inngest

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
