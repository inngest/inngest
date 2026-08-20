package expressions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func generatedFunctionFailedData() map[string]any {
	stack := strings.Repeat("at syntheticWorker (/srv/app/internal/worker.go:100:20)\n", 20)
	originalEvent := map[string]any{
		"id":   "01SYNTHETICORIGINALEVENT0000",
		"name": "app/synthetic.requested",
		"ts":   int64(1_725_000_000_000),
		"data": map[string]any{
			"job_id": "00000000-0000-4000-8000-000000000002",
			"amount": 42,
			"transport_context": map[string]any{
				"request_meta": map[string]any{
					"request_id": "00000000-0000-4000-8000-000000000003",
					"started_at": int64(1_725_000_000_000),
				},
			},
		},
		"user": map[string]any{},
	}

	return map[string]any{
		"event": map[string]any{
			"id":   "01SYNTHETICFUNCTIONFAILED000",
			"name": "inngest/function.failed",
			"ts":   int64(1_725_000_000_000),
			"data": map[string]any{
				"_inngest":    map[string]any{"status": "failed"},
				"function_id": "00000000-0000-4000-8000-000000000001",
				"run_id":      "01SYNTHETICRUN0000000000000",
				"error": map[string]any{
					"name":    "SyntheticError",
					"message": "representative anonymized failure context",
					"stack":   stack,
				},
				"event":  originalEvent,
				"events": []any{originalEvent},
				"result": map[string]any{
					"name":    "SyntheticError",
					"message": "representative anonymized failure context",
					"stack":   stack,
				},
			},
		},
		"user": map[string]any{
			"external_id": "synthetic-user",
		},
	}
}

func TestLiftedExpressionEvaluateSharedDataConcurrently(t *testing.T) {
	ctx := context.Background()
	data := NewData(generatedFunctionFailedData())
	tests := []struct {
		expression string
		expected   bool
	}{
		{expression: `event.data.error.name == "SyntheticError"`, expected: true},
		{expression: `event.data.error.name == "OtherError"`, expected: false},
	}
	evaluators := make([]BooleanEvaluator, len(tests))
	for i, test := range tests {
		var err error
		evaluators[i], err = NewBooleanEvaluator(ctx, test.expression)
		require.NoError(t, err)
	}

	const workers = 64
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			test := tests[worker%len(tests)]
			evaluator := evaluators[worker%len(evaluators)]
			for i := 0; i < 100; i++ {
				actual, err := evaluator.Evaluate(ctx, data)
				if err != nil {
					errs <- err
					return
				}
				if actual != test.expected {
					errs <- fmt.Errorf("expression %q: got %t, want %t", test.expression, actual, test.expected)
					return
				}
			}
		}(worker)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	_, exists := data.Get(ctx, []string{"vars"})
	require.False(t, exists, "evaluation mutated the shared input")
}

func BenchmarkLiftedExpressionEvaluate(b *testing.B) {
	ctx := context.Background()
	evaluator, err := NewBooleanEvaluator(ctx, `event.data.error.name == "SyntheticError" && event.data._inngest.status == "failed"`)
	if err != nil {
		b.Fatal(err)
	}
	input := generatedFunctionFailedData()
	encoded, err := json.Marshal(input)
	if err != nil {
		b.Fatal(err)
	}
	data := NewData(input)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matched, err := evaluator.Evaluate(ctx, data)
		if err != nil {
			b.Fatal(err)
		}
		if !matched {
			b.Fatal("expression did not match generated event")
		}
	}
	b.ReportMetric(float64(len(encoded)), "payload_B")
}
