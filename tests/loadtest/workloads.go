package loadtest

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
)

// sdkWorkloadClient adapts an inngestgo.Client to the WorkloadClient interface.
type sdkWorkloadClient struct {
	client inngestgo.Client
}

func newWorkloadClient(client inngestgo.Client) WorkloadClient {
	return &sdkWorkloadClient{client: client}
}

func (c *sdkWorkloadClient) CreateSimpleFunction(id string, eventName string, handler func(eventData map[string]any) error) error {
	_, err := inngestgo.CreateFunction(
		c.client,
		inngestgo.FunctionOpts{ID: id},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			return nil, handler(input.Event.Data)
		},
	)
	return err
}

func (c *sdkWorkloadClient) CreateBatchFunction(id string, eventName string, batchSize int, batchTimeout time.Duration, handler func(eventsData []map[string]any) error) error {
	_, err := inngestgo.CreateFunction(
		c.client,
		inngestgo.FunctionOpts{
			ID:          id,
			BatchEvents: &inngestgo.ConfigBatchEvents{MaxSize: batchSize, Timeout: batchTimeout},
		},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			var allData []map[string]any
			for _, evt := range input.Events {
				allData = append(allData, evt.Data)
			}
			return nil, handler(allData)
		},
	)
	return err
}

func (c *sdkWorkloadClient) CreateDebounceFunction(id string, eventName string, period time.Duration, key string, handler func(eventData map[string]any) error) error {
	_, err := inngestgo.CreateFunction(
		c.client,
		inngestgo.FunctionOpts{
			ID:       id,
			Debounce: &inngestgo.ConfigDebounce{Period: period, Key: key},
		},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			return nil, handler(input.Event.Data)
		},
	)
	return err
}

func (c *sdkWorkloadClient) CreateConcurrencyFunction(id string, eventName string, limit int, key string, handler func(eventData map[string]any) error) error {
	concurrency := []inngestgo.ConfigStepConcurrency{
		{Limit: limit},
	}
	if key != "" {
		concurrency[0].Key = &key
	}
	_, err := inngestgo.CreateFunction(
		c.client,
		inngestgo.FunctionOpts{
			ID:          id,
			Concurrency: concurrency,
		},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			return nil, handler(input.Event.Data)
		},
	)
	return err
}

func (c *sdkWorkloadClient) CreateThrottleFunction(id string, eventName string, limit uint, period time.Duration, handler func(eventData map[string]any) error) error {
	_, err := inngestgo.CreateFunction(
		c.client,
		inngestgo.FunctionOpts{
			ID:       id,
			Throttle: &inngestgo.ConfigThrottle{Limit: limit, Period: period},
		},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			return nil, handler(input.Event.Data)
		},
	)
	return err
}

func (c *sdkWorkloadClient) CreateMultiStepFunction(id string, eventName string, steps int, handler func(stepIndex int, eventData map[string]any) error) error {
	_, err := inngestgo.CreateFunction(
		c.client,
		inngestgo.FunctionOpts{ID: id},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			for i := range steps {
				stepIdx := i
				_, err := step.Run(ctx, fmt.Sprintf("step-%d", stepIdx), func(ctx context.Context) (any, error) {
					return nil, handler(stepIdx, input.Event.Data)
				})
				if err != nil {
					return nil, err
				}
			}
			return nil, nil
		},
	)
	return err
}

// --- Predefined workload factories ---

// SimpleWorkload creates a minimal function that returns immediately.
func SimpleWorkload() Workload {
	return Workload{
		Name: "simple",
		SetupFn: func(client WorkloadClient, collector *Collector) (string, error) {
			eventName := "loadtest/simple"
			err := client.CreateSimpleFunction("loadtest-simple", eventName, func(data map[string]any) error {
				now := time.Now()
				if id, ok := data["loadtest_id"].(string); ok {
					collector.RecordFirstHit(id, now)
					collector.RecordCompletion(id, now)
				}
				return nil
			})
			return eventName, err
		},
	}
}

// BatchWorkload creates a function that collects events into batches.
func BatchWorkload(size int, timeout time.Duration) Workload {
	return Workload{
		Name: fmt.Sprintf("batch-%d", size),
		SetupFn: func(client WorkloadClient, collector *Collector) (string, error) {
			eventName := "loadtest/batch"
			err := client.CreateBatchFunction("loadtest-batch", eventName, size, timeout, func(eventsData []map[string]any) error {
				now := time.Now()
				var ids []string
				for _, data := range eventsData {
					if id, ok := data["loadtest_id"].(string); ok {
						ids = append(ids, id)
					}
				}
				collector.RecordBatchCompletion(ids, now, now)
				return nil
			})
			return eventName, err
		},
		ExpectedCompletions: func(total int) int {
			return total // Each event still gets a completion recorded via batch handler.
		},
	}
}

// DebounceWorkload creates a function with debounce configured.
// Note: with debounce, only the last event in each debounce window actually triggers
// execution, so many events may never complete. The collector tracks only what runs.
func DebounceWorkload(period time.Duration, key string) Workload {
	return Workload{
		Name: "debounce",
		SetupFn: func(client WorkloadClient, collector *Collector) (string, error) {
			eventName := "loadtest/debounce"
			err := client.CreateDebounceFunction("loadtest-debounce", eventName, period, key, func(data map[string]any) error {
				now := time.Now()
				if id, ok := data["loadtest_id"].(string); ok {
					collector.RecordFirstHit(id, now)
					collector.RecordCompletion(id, now)
				}
				return nil
			})
			return eventName, err
		},
		ExpectedCompletions: func(total int) int {
			// Debounce collapses events, so only ~1 completion per debounce window.
			// Use 1 as a minimum expected; the actual count depends on timing.
			return 1
		},
	}
}

// ConcurrencyWorkload creates a function with a concurrency limit.
func ConcurrencyWorkload(limit int) Workload {
	return Workload{
		Name: fmt.Sprintf("concurrency-%d", limit),
		SetupFn: func(client WorkloadClient, collector *Collector) (string, error) {
			eventName := "loadtest/concurrency"
			err := client.CreateConcurrencyFunction("loadtest-concurrency", eventName, limit, "", func(data map[string]any) error {
				now := time.Now()
				if id, ok := data["loadtest_id"].(string); ok {
					collector.RecordFirstHit(id, now)
					// Simulate some work to make concurrency limiting observable.
					time.Sleep(10 * time.Millisecond)
					collector.RecordCompletion(id, time.Now())
				}
				return nil
			})
			return eventName, err
		},
	}
}

// ThrottleWorkload creates a function with throttle (soft rate limit).
func ThrottleWorkload(limit uint, period time.Duration) Workload {
	return Workload{
		Name: fmt.Sprintf("throttle-%d", limit),
		SetupFn: func(client WorkloadClient, collector *Collector) (string, error) {
			eventName := "loadtest/throttle"
			err := client.CreateThrottleFunction("loadtest-throttle", eventName, limit, period, func(data map[string]any) error {
				now := time.Now()
				if id, ok := data["loadtest_id"].(string); ok {
					collector.RecordFirstHit(id, now)
					collector.RecordCompletion(id, now)
				}
				return nil
			})
			return eventName, err
		},
	}
}

// MultiStepWorkload creates a function with N sequential steps.
func MultiStepWorkload(steps int) Workload {
	return Workload{
		Name: fmt.Sprintf("multistep-%d", steps),
		SetupFn: func(client WorkloadClient, collector *Collector) (string, error) {
			eventName := "loadtest/multistep"
			err := client.CreateMultiStepFunction("loadtest-multistep", eventName, steps, func(stepIdx int, data map[string]any) error {
				now := time.Now()
				if id, ok := data["loadtest_id"].(string); ok {
					if stepIdx == 0 {
						collector.RecordFirstHit(id, now)
					}
					if stepIdx == steps-1 {
						collector.RecordCompletion(id, now)
					}
				}
				return nil
			})
			return eventName, err
		},
	}
}
