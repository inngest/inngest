package metrics

import (
	"testing"
)

func TestAggregateNsPercentiles(t *testing.T) {
	// 1ms, 2ms, ..., 100ms
	xs := make([]int64, 100)
	for i := range xs {
		xs[i] = int64((i + 1) * 1_000_000)
	}
	got := aggregateNs(xs)
	if got.Count != 100 {
		t.Errorf("count: got %d, want 100", got.Count)
	}
	// Index-based selection: p50 -> idx 49 -> 50ms; p95 -> idx 94 -> 95ms;
	// p99 -> idx 98 -> 99ms.
	if got.P50 != 50 {
		t.Errorf("p50: got %v, want 50", got.P50)
	}
	if got.P95 != 95 {
		t.Errorf("p95: got %v, want 95", got.P95)
	}
	if got.P99 != 99 {
		t.Errorf("p99: got %v, want 99", got.P99)
	}
}

func TestAggregateNsEmpty(t *testing.T) {
	got := aggregateNs(nil)
	if got.Count != 0 {
		t.Errorf("empty count: got %d, want 0", got.Count)
	}
}

func TestAggregateNsSorts(t *testing.T) {
	xs := []int64{50_000_000, 10_000_000, 30_000_000, 20_000_000, 40_000_000}
	got := aggregateNs(xs)
	if got.P50 != 30 {
		t.Errorf("p50: got %v, want 30", got.P50)
	}
}
