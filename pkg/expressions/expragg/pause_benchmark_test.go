package expragg

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	// Uncomment for profiling
	// "os"
	// "runtime"
	// "runtime/pprof"
	// "runtime/trace"

	"github.com/google/uuid"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/expressions/exprenv"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

// BenchmarkPauseFast benchmarks fast expressions (equality only) with different version values
func BenchmarkPauseFast(b *testing.B) {
	runPauseBenchmark(b, pauseBenchmarkOpts{
		useSameVersion:     false,
		useMixedExpression: false,
	})
}

// BenchmarkPauseMixed benchmarks mixed expressions (equality + comparison + string operations)
func BenchmarkPauseMixed(b *testing.B) {
	runPauseBenchmark(b, pauseBenchmarkOpts{
		useSameVersion:     false,
		useMixedExpression: true,
	})
}

// BenchmarkPauseSameVersionFast benchmarks fast expressions with the same version value
func BenchmarkPauseSameVersionFast(b *testing.B) {
	runPauseBenchmark(b, pauseBenchmarkOpts{
		useSameVersion:     true,
		useMixedExpression: false,
	})
}

type pauseBenchmarkOpts struct {
	useSameVersion     bool
	useMixedExpression bool
}

func runPauseBenchmark(b *testing.B, opts pauseBenchmarkOpts) {
	ctx := context.Background()
	parser := expr.NewTreeParser(exprenv.CompilerSingleton())

	numExpressions := b.N

	// Create the aggregate evaluator
	ae := expr.NewAggregateEvaluator(expr.AggregateEvaluatorOpts[*state.Pause]{
		Parser:      parser,
		Eval:        expressions.ExprEvaluator,
		Concurrency: 1000,
		KV:          &mockKV{},
		Log:         logger.From(ctx).SLog(),
	})
	defer ae.Close()

	// Add expressions
	b.Logf("Adding %d expressions (useSameVersion=%v, useMixedExpression=%v)...",
		numExpressions, opts.useSameVersion, opts.useMixedExpression)
	for i := 1; i <= numExpressions; i++ {
		var version int
		if opts.useSameVersion {
			version = 1
		} else {
			version = (i % 10) + 1
		}

		var expression string
		if opts.useMixedExpression {
			expression = fmt.Sprintf(`"sub_%d" == async.data.subscriptionId && "2025-10-28T23:21:37.821Z" <= async.data.createdAt && "v%d" == async.data.version`, i, version)
		} else {
			expression = fmt.Sprintf(`"sub_%d" == async.data.subscriptionId && "v%d" == async.data.version`, i, version)
		}

		pause := &state.Pause{
			ID:         uuid.New(),
			Expression: &expression,
		}

		_, err := ae.Add(ctx, pause)
		if err != nil {
			b.Fatalf("Failed to add evaluable %d: %v", i, err)
		}
	}

	// Print how many evaluables the aggregate evaluator has
	b.Logf("Aggregate evaluator has %d total evaluables (Fast: %d, Mixed: %d, Slow: %d)",
		ae.Len(), ae.FastLen(), ae.MixedLen(), ae.SlowLen())

	// Pre-generate test data sets with sequential subscription IDs
	testDataSets := make([]map[string]any, numExpressions)
	for i := 1; i <= numExpressions; i++ {
		subscriptionId := fmt.Sprintf("sub_%d", i)
		var versionNum int
		if opts.useSameVersion {
			versionNum = 1
		} else {
			versionNum = (i % 10) + 1
		}
		version := fmt.Sprintf("v%d", versionNum)
		testDataSets[i-1] = map[string]any{
			"async": map[string]any{
				"data": map[string]any{
					"attempts":       0,
					"createdAt":      "2025-10-28T23:21:37.821Z",
					"subscriptionId": subscriptionId,
					"version":        version,
				},
				"id":   "01K8PSJM4N3V520F0STRTCASWS",
				"name": "my-event",
				"ts":   1761701613717,
			},
		}
	}

	done := make(chan struct{})
	defer close(done)

	// Continually add more pauses during benchmark to stress locks contention further.
	// go func() {
	// 	addCounter := numExpressions
	// 	for {
	// 		select {
	// 		case <-done:
	// 			return
	// 		default:
	// 			addCounter++
	// 			var version int
	// 			if opts.useSameVersion {
	// 				version = 1
	// 			} else {
	// 				version = (addCounter % 10) + 1
	// 			}
	// 			var expression string
	// 			if opts.useMixedExpression {
	// 				expression = fmt.Sprintf(`"sub_%d" == async.data.subscriptionId && "2025-10-28T23:21:37.821Z" < async.data.createdAt && "v%d" == async.data.version`, addCounter, version)
	// 			} else {
	// 				expression = fmt.Sprintf(`"sub_%d" == async.data.subscriptionId && "v%d" == async.data.version`, addCounter, version)
	// 			}
	// 			pause := &state.Pause{
	// 				ID:         uuid.New(),
	// 				Expression: &expression,
	// 			}
	// 			ae.Add(ctx, pause)
	// 		}
	// 	}
	// }()

	b.ResetTimer()

	// === CPU PROFILING ===
	// f, err := os.Create("pause_cpu.prof")
	// if err != nil {
	// 	b.Fatal(err)
	// }
	// defer f.Close()
	// if err := pprof.StartCPUProfile(f); err != nil {
	// 	b.Fatal(err)
	// }
	// defer pprof.StopCPUProfile()

	// === MUTEX PROFILING ===
	// runtime.SetMutexProfileFraction(1) // Enable mutex profiling (1 = sample every event)
	// defer func() {
	// 	mutexf, err := os.Create("pause_mutex.prof")
	// 	if err != nil {
	// 		b.Fatal(err)
	// 	}
	// 	defer mutexf.Close()
	// 	if err := pprof.Lookup("mutex").WriteTo(mutexf, 0); err != nil {
	// 		b.Fatal(err)
	// 	}
	// }()

	// === MEMORY PROFILING ===
	// defer func() {
	// 	mf, err := os.Create("pause_mem.prof")
	// 	if err != nil {
	// 		b.Fatal(err)
	// 	}
	// 	defer mf.Close()
	// 	runtime.GC() // Force GC before capturing heap profile
	// 	if err := pprof.WriteHeapProfile(mf); err != nil {
	// 		b.Fatal(err)
	// 	}
	// }()

	// === TRACING ===
	// tf, err := os.Create("pause_trace.out")
	// if err != nil {
	// 	b.Fatal(err)
	// }
	// defer tf.Close()
	// if err := trace.Start(tf); err != nil {
	// 	b.Fatal(err)
	// }
	// defer trace.Stop()

	// Simulate concurrent events with sequential subscription IDs
	var counter int64
	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine gets sequential test data
		for pb.Next() {
			index := int(atomic.AddInt64(&counter, 1) % int64(numExpressions))
			localTestData := testDataSets[index]
			vals, _, _ := ae.Evaluate(ctx, localTestData)
			require.Len(b, vals, 1)
			for _, val := range vals {
				_ = ae.Remove(ctx, val)
			}
		}
	})
}

// mockKV is a simple in-memory KV store for testing
type mockKV struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*state.Pause
}

func (m *mockKV) Get(evalID uuid.UUID) (*state.Pause, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.data == nil {
		return nil, nil
	}
	return m.data[evalID], nil
}

func (m *mockKV) Set(eval *state.Pause) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[uuid.UUID]*state.Pause)
	}
	m.data[eval.ID] = eval
	return nil
}

func (m *mockKV) Remove(evalID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data != nil {
		delete(m.data, evalID)
	}
	return nil
}

func (m *mockKV) Len() int32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int32(len(m.data))
}
