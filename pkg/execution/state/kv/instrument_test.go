package kv

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/domain"
	"github.com/stretchr/testify/require"
)

// fakeStore returns canned results and records the options it was given.
type fakeStore struct {
	getEntry Entry
	getErr   error
	setErr   error
	delErr   error
	setOpts  Options
	delOpts  Options
}

func (f *fakeStore) Get(ctx context.Context, k Key) (Entry, error) {
	if f.getErr != nil {
		return Entry{}, f.getErr
	}
	e := f.getEntry
	e.Key = k
	return e, nil
}

func (f *fakeStore) Set(ctx context.Context, e Entry, opts ...Option) (Entry, error) {
	f.setOpts = ApplyOptions(opts...)
	if f.setErr != nil {
		return Entry{}, f.setErr
	}
	e.Version = 7
	return e, nil
}

func (f *fakeStore) Delete(ctx context.Context, k Key, opts ...Option) error {
	f.delOpts = ApplyOptions(opts...)
	return f.delErr
}

type recorder struct {
	ops []Operation
}

func (r *recorder) OnOperation(ctx context.Context, op Operation) {
	r.ops = append(r.ops, op)
}

func (r *recorder) last(t *testing.T) Operation {
	t.Helper()
	require.NotEmpty(t, r.ops)
	return r.ops[len(r.ops)-1]
}

func TestInstrumentGet(t *testing.T) {
	ctx := context.Background()
	key := testKey()
	value := json.RawMessage(`{"a":1}`)

	cases := []struct {
		name   string
		err    error
		result Result
		bytes  int64
	}{
		{name: "ok", result: ResultOK, bytes: int64(len(value))},
		{name: "not found", err: ErrNotFound, result: ResultNotFound},
		{name: "error", err: errors.New("boom"), result: ResultError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			inner := &fakeStore{
				getEntry: Entry{
					Value:   value,
					Version: 3,
				},
				getErr: c.err,
			}
			rec := &recorder{}
			s := Instrument(inner, rec)

			e, err := s.Get(ctx, key)
			if c.err == nil {
				require.NoError(t, err)
				require.Equal(t, value, e.Value)
				require.Equal(t, int64(3), e.Version)
			} else {
				require.ErrorIs(t, err, c.err)
			}

			op := rec.last(t)
			require.Equal(t, Operation{
				Tenant:    key.Tenant,
				Namespace: key.Namespace,
				Op:        OpGet,
				Result:    c.result,
				Bytes:     c.bytes,
			}, op)
			require.Len(t, rec.ops, 1)
		})
	}
}

func TestInstrumentSet(t *testing.T) {
	ctx := context.Background()
	key := testKey()
	value := json.RawMessage(`{"a":1,"b":[1,2,3]}`)

	cases := []struct {
		name   string
		err    error
		result Result
		bytes  int64
	}{
		{name: "ok", result: ResultOK, bytes: int64(len(value))},
		{name: "conflict", err: ErrConflict, result: ResultConflict},
		{name: "validation", err: ErrInvalidKey, result: ResultError},
		{name: "error", err: errors.New("boom"), result: ResultError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			inner := &fakeStore{setErr: c.err}
			rec := &recorder{}
			s := Instrument(inner, rec)

			out, err := s.Set(ctx, Entry{Key: key, Value: value}, WithExpected(2), WithTTL(10))
			if c.err == nil {
				require.NoError(t, err)
				require.Equal(t, int64(7), out.Version)
			} else {
				require.ErrorIs(t, err, c.err)
			}

			require.NotNil(t, inner.setOpts.Expected)
			require.Equal(t, int64(2), *inner.setOpts.Expected)
			require.EqualValues(t, 10, inner.setOpts.TTL)

			op := rec.last(t)
			require.Equal(t, Operation{
				Tenant:    key.Tenant,
				Namespace: key.Namespace,
				Op:        OpSet,
				Result:    c.result,
				Bytes:     c.bytes,
			}, op)
		})
	}
}

func TestInstrumentDelete(t *testing.T) {
	ctx := context.Background()
	key := testKey()

	cases := []struct {
		name   string
		err    error
		result Result
	}{
		{name: "ok", result: ResultOK},
		{name: "conflict", err: ErrConflict, result: ResultConflict},
		{name: "error", err: errors.New("boom"), result: ResultError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			inner := &fakeStore{delErr: c.err}
			rec := &recorder{}
			s := Instrument(inner, rec)

			err := s.Delete(ctx, key, WithExpected(5))
			if c.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, c.err)
			}
			require.NotNil(t, inner.delOpts.Expected)
			require.Equal(t, int64(5), *inner.delOpts.Expected)

			op := rec.last(t)
			require.Equal(t, Operation{
				Tenant:    key.Tenant,
				Namespace: key.Namespace,
				Op:        OpDelete,
				Result:    c.result,
			}, op)
		})
	}
}

func TestInstrumentNilHook(t *testing.T) {
	ctx := context.Background()
	key := testKey()
	inner := &fakeStore{
		getEntry: Entry{Value: json.RawMessage(`1`)},
	}
	s := Instrument(inner, nil)

	_, err := s.Get(ctx, key)
	require.NoError(t, err)
	_, err = s.Set(ctx, Entry{Key: key, Value: json.RawMessage(`2`)})
	require.NoError(t, err)
	require.NoError(t, s.Delete(ctx, key))
}

func TestInstrumentHookFunc(t *testing.T) {
	ctx := context.Background()
	var got Operation
	s := Instrument(&fakeStore{}, HookFunc(func(ctx context.Context, op Operation) {
		got = op
	}))
	require.NoError(t, s.Delete(ctx, testKey()))
	require.Equal(t, OpDelete, got.Op)
	require.Equal(t, ResultOK, got.Result)
}

func TestResultOf(t *testing.T) {
	require.Equal(t, ResultOK, resultOf(nil))
	require.Equal(t, ResultNotFound, resultOf(ErrNotFound))
	require.Equal(t, ResultConflict, resultOf(ErrConflict))
	require.Equal(t, ResultError, resultOf(ErrInvalidNamespace))
	require.Equal(t, ResultError, resultOf(errors.New("x")))
}

func TestInstrumentTenantPassthrough(t *testing.T) {
	ctx := context.Background()
	tenant := domain.NewTenant(uuid.New(), uuid.New())
	key := Key{
		Tenant:    tenant,
		Namespace: "ns",
		Key:       "k",
	}
	rec := &recorder{}
	s := Instrument(&fakeStore{}, rec)
	require.NoError(t, s.Delete(ctx, key))
	require.Equal(t, tenant, rec.last(t).Tenant)
	require.Equal(t, "ns", rec.last(t).Namespace)
}
