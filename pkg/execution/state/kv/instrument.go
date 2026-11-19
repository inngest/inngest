package kv

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

const pkgName = "execution.state.kv"

// Instrument wraps s so every call records its duration, result, and bytes
// as metrics without tenant tags, then reports the call to hook.  a nil
// hook records the metrics only.
func Instrument(s Store, hook Hook) Store {
	return &instrumented{
		inner: s,
		hook:  hook,
	}
}

type instrumented struct {
	inner Store
	hook  Hook
}

func (s *instrumented) Get(ctx context.Context, k Key) (Entry, error) {
	start := time.Now()
	e, err := s.inner.Get(ctx, k)
	var n int64
	if err == nil {
		n = int64(len(e.Value))
	}
	s.record(ctx, start, k, OpGet, err, n)
	return e, err
}

func (s *instrumented) Set(ctx context.Context, e Entry, opts ...Option) (Entry, error) {
	start := time.Now()
	out, err := s.inner.Set(ctx, e, opts...)
	var n int64
	if err == nil {
		n = int64(len(e.Value))
	}
	s.record(ctx, start, e.Key, OpSet, err, n)
	return out, err
}

func (s *instrumented) Delete(ctx context.Context, k Key, opts ...Option) error {
	start := time.Now()
	err := s.inner.Delete(ctx, k, opts...)
	s.record(ctx, start, k, OpDelete, err, 0)
	return err
}

func (s *instrumented) record(ctx context.Context, start time.Time, k Key, op Op, err error, n int64) {
	res := resultOf(err)
	metrics.HistogramKVOperationDuration(ctx, time.Since(start).Milliseconds(), metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"op":        string(op),
			"namespace": k.Namespace,
			"result":    string(res),
		},
	})
	if res == ResultOK && op != OpDelete {
		dir := "read"
		if op == OpSet {
			dir = "written"
		}
		metrics.HistogramKVValueSize(ctx, n, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"op":        string(op),
				"namespace": k.Namespace,
			},
		})
		metrics.IncrKVBytesCounter(ctx, n, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"op":        dir,
				"namespace": k.Namespace,
			},
		})
	}
	if s.hook == nil {
		return
	}
	s.hook.OnOperation(ctx, Operation{
		Tenant:    k.Tenant,
		Namespace: k.Namespace,
		Op:        op,
		Result:    res,
		Bytes:     n,
	})
}
