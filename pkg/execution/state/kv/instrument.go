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
	record(ctx, s.hook, start, k, OpGet, err, n)
	return e, err
}

func (s *instrumented) Set(ctx context.Context, e Entry, opts ...Option) (Entry, error) {
	start := time.Now()
	out, err := s.inner.Set(ctx, e, opts...)
	var n int64
	if err == nil {
		n = int64(len(e.Value))
	}
	record(ctx, s.hook, start, e.Key, OpSet, err, n)
	return out, err
}

func (s *instrumented) Delete(ctx context.Context, k Key, opts ...Option) error {
	start := time.Now()
	err := s.inner.Delete(ctx, k, opts...)
	record(ctx, s.hook, start, k, OpDelete, err, 0)
	return err
}

func (s *instrumented) Extend(ctx context.Context, k Key, d time.Duration) error {
	start := time.Now()
	err := s.inner.Extend(ctx, k, d)
	record(ctx, s.hook, start, k, OpExtend, err, 0)
	return err
}

// InstrumentList wraps s the same way Instrument wraps a Store.
func InstrumentList(s ListStore, hook Hook) ListStore {
	return &instrumentedList{
		inner: s,
		hook:  hook,
	}
}

type instrumentedList struct {
	inner ListStore
	hook  Hook
}

func (s *instrumentedList) Push(ctx context.Context, k Key, it Item, opts ...Option) (bool, error) {
	start := time.Now()
	added, err := s.inner.Push(ctx, k, it, opts...)
	res := resultOf(err)
	var n int64
	switch {
	case res == ResultOK && added:
		n = int64(len(it.Value))
	case res == ResultOK:
		res = ResultDuplicate
	}
	recordResult(ctx, s.hook, start, k, OpPush, res, n)
	return added, err
}

func (s *instrumentedList) Peek(ctx context.Context, k Key, limit int) ([]Item, error) {
	start := time.Now()
	items, err := s.inner.Peek(ctx, k, limit)
	var n int64
	for _, it := range items {
		n += int64(len(it.Value))
	}
	record(ctx, s.hook, start, k, OpPeek, err, n)
	return items, err
}

func (s *instrumentedList) Remove(ctx context.Context, k Key, ids []string, opts ...Option) error {
	start := time.Now()
	err := s.inner.Remove(ctx, k, ids, opts...)
	record(ctx, s.hook, start, k, OpRemove, err, 0)
	return err
}

func (s *instrumentedList) Clear(ctx context.Context, k Key) error {
	start := time.Now()
	err := s.inner.Clear(ctx, k)
	record(ctx, s.hook, start, k, OpClear, err, 0)
	return err
}

// bytesDir returns the direction tag for the bytes of a call, or "" for a
// call that moves no value bytes.
func bytesDir(op Op) string {
	switch op {
	case OpGet, OpPeek:
		return "read"
	case OpSet, OpPush:
		return "written"
	}
	return ""
}

func record(ctx context.Context, hook Hook, start time.Time, k Key, op Op, err error, n int64) {
	recordResult(ctx, hook, start, k, op, resultOf(err), n)
}

func recordResult(ctx context.Context, hook Hook, start time.Time, k Key, op Op, res Result, n int64) {
	metrics.HistogramKVOperationDuration(ctx, time.Since(start).Milliseconds(), metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"op":        string(op),
			"namespace": k.Namespace,
			"result":    string(res),
		},
	})
	if dir := bytesDir(op); res == ResultOK && dir != "" {
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
	if hook == nil {
		return
	}
	hook.OnOperation(ctx, Operation{
		Tenant:    k.Tenant,
		Namespace: k.Namespace,
		Op:        op,
		Result:    res,
		Bytes:     n,
	})
}
