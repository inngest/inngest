package redis

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"testing"
)

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

	for i := 0; i < 50; i++ {
		type l1 struct{}
		ctx = context.WithValue(ctx, l1{}, "test")
	}

	return ctx
}

func BenchmarkDirectClient(b *testing.B) {
	ctx := createLargeContext()
	rc := setupMiniredis(b)
	defer rc.Close()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_ = rc.Do(ctx, rc.B().Set().Key("test").Value("test").Build())
	}
}

func BenchmarkInstrumentedClient(b *testing.B) {
	ctx := createLargeContext()
	rc := wrapWithObservability(setupMiniredis(b), InstrumentedClientOpts{"test", "test"})
	defer rc.Close()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		// wrapping context adds ~1µs
		_ = rc.Do(WithScriptName(WithScope(ctx, ScopePauses), "test/script"), rc.B().Set().Key("test").Value("test").Build())
	}
}
