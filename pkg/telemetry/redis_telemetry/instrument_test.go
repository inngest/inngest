package redis_telemetry

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const largeContextSize = 50

func setupMiniredis(b *testing.B) rueidis.Client {
	r := miniredis.RunT(b)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(b, err)

	return rc
}

func createLargeContext() context.Context {
	ctx := context.Background()

	for i := 0; i < largeContextSize; i++ {
		type l1 struct{}
		ctx = context.WithValue(ctx, l1{}, "test")
	}

	return ctx
}

// ~21,000ns
func BenchmarkDirectClient(b *testing.B) {
	ctx := createLargeContext()
	rc := setupMiniredis(b)
	defer rc.Close()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_ = rc.Do(ctx, rc.B().Set().Key("test").Value("test").Build())
	}
}

// 4,000ns
func BenchmarkParallelDirectClient(b *testing.B) {
	ctx := createLargeContext()
	rc := setupMiniredis(b)
	defer rc.Close()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = rc.Do(ctx, rc.B().Set().Key("test").Value("test").Build())
		}
	})
}

// ~24,000ns with largeContextSize = 50
func BenchmarkInstrumentedClient(b *testing.B) {
	ctx := createLargeContext()
	rc := InstrumentRedisClient(context.Background(), setupMiniredis(b), InstrumentedClientOpts{"test", "test", 0, 0})
	defer rc.Close()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		// wrapping context adds ~1µs
		_ = rc.Do(WithOpName(WithScriptName(WithScope(ctx, ScopePauses), "test/script"), "testop"), rc.B().Set().Key("test").Value("test").Build())
	}
}

// ~8,000ns with largeContextSize = 50
func BenchmarkParallelInstrumentedClient(b *testing.B) {
	ctx := createLargeContext()
	rc := InstrumentRedisClient(context.Background(), setupMiniredis(b), InstrumentedClientOpts{"test", "test", 0, 0})
	defer rc.Close()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// wrapping context adds ~1µs
			_ = rc.Do(WithOpName(WithScriptName(WithScope(ctx, ScopePauses), "test/script"), "testop"), rc.B().Set().Key("test").Value("test").Build())
		}
	})
}

// 100ns
func BenchmarkContextEnrichFull(b *testing.B) {
	ctx := createLargeContext()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_ = WithOpName(WithScriptName(WithScope(ctx, ScopePauses), "test/script"), "testop")
	}
}

// 30ns
func BenchmarkContextEnrichSingle(b *testing.B) {
	ctx := createLargeContext()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_ = WithOpName(ctx, "testop")
	}
}

// 30ns
func BenchmarkNow(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_ = time.Now()
	}
}

// 0.3ns
func BenchmarkSliceAccess(b *testing.B) {
	sl := []string{"ZADD", "key", "score", "member"}
	getSl := func() []string {
		return sl
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		cmd := ""
		if len(getSl()) > 0 {
			cmd = getSl()[0]
		}
		_ = cmd
	}
}

// 0.3ns
func BenchmarkFunc(b *testing.B) {
	testFunc := func() {}

	for n := 0; n < b.N; n++ {
		testFunc()
	}
}

// 1.5ns
func BenchmarkDefer(b *testing.B) {
	testFunc := func() {
		defer func() {
			doSomething := true
			_ = doSomething
		}()
	}

	for n := 0; n < b.N; n++ {
		testFunc()
	}
}

// 200ns
func BenchmarkDeferGoroutine(b *testing.B) {
	testFunc := func() {
		defer func() {
			go func() {
				doSomething := true
				_ = doSomething
			}()
		}()
	}

	for n := 0; n < b.N; n++ {
		testFunc()
	}
}

// 190ns
func BenchmarkGoroutine(b *testing.B) {
	for n := 0; n < b.N; n++ {
		go func() {

		}()
	}
}

// 2ns
func BenchmarkDurationSub(b *testing.B) {
	start := time.Now()
	end := start.Add(time.Second)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_ = end.Sub(start)
	}
}
