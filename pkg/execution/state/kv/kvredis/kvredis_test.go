package kvredis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/domain"
	"github.com/inngest/inngest/pkg/execution/state/kv"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func newStore(t testing.TB, prefix string) (*Store, *miniredis.Miniredis) {
	t.Helper()
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	return New(rc, prefix), r
}

func testKey() kv.Key {
	return kv.Key{
		Tenant:    domain.NewTenant(uuid.New(), uuid.New()),
		Namespace: "steps",
		Key:       "step-1",
	}
}

func TestStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, "kv:")
	key := testKey()
	value := json.RawMessage(`{"a":1}`)

	_, err := s.Get(ctx, key)
	require.ErrorIs(t, err, kv.ErrNotFound)

	out, err := s.Set(ctx, kv.Entry{Key: key, Value: value})
	require.NoError(t, err)
	require.Equal(t, key, out.Key)
	require.Equal(t, value, out.Value)
	require.Equal(t, int64(1), out.Version)

	raw := r.HGet("kv:"+key.String(), fieldData)
	require.Equal(t, string(value), raw)
	require.Equal(t, "1", r.HGet("kv:"+key.String(), fieldVersion))

	got, err := s.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, key, got.Key)
	require.Equal(t, value, got.Value)
	require.Equal(t, int64(1), got.Version)

	out, err = s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`2`)})
	require.NoError(t, err)
	require.Equal(t, int64(2), out.Version)

	require.NoError(t, s.Delete(ctx, key))
	_, err = s.Get(ctx, key)
	require.ErrorIs(t, err, kv.ErrNotFound)

	// deleting a missing key is fine.
	require.NoError(t, s.Delete(ctx, key))
}

func TestStoreEmptyValue(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t, "")
	key := testKey()

	_, err := s.Set(ctx, kv.Entry{Key: key})
	require.NoError(t, err)

	got, err := s.Get(ctx, key)
	require.NoError(t, err)
	require.Empty(t, got.Value)
	require.Equal(t, int64(1), got.Version)
}

func TestStoreCreateOnly(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t, "")
	key := testKey()

	out, err := s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`1`)}, kv.WithExpected(0))
	require.NoError(t, err)
	require.Equal(t, int64(1), out.Version)

	_, err = s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`2`)}, kv.WithExpected(0))
	require.ErrorIs(t, err, kv.ErrConflict)

	got, err := s.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, json.RawMessage(`1`), got.Value)
	require.Equal(t, int64(1), got.Version)
}

func TestStoreExpectedVersion(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t, "")
	key := testKey()

	// a missing key is at version 0, so expecting anything else conflicts.
	_, err := s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`1`)}, kv.WithExpected(1))
	require.ErrorIs(t, err, kv.ErrConflict)

	out, err := s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`1`)})
	require.NoError(t, err)
	require.Equal(t, int64(1), out.Version)

	out, err = s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`2`)}, kv.WithExpected(1))
	require.NoError(t, err)
	require.Equal(t, int64(2), out.Version)

	_, err = s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`3`)}, kv.WithExpected(1))
	require.ErrorIs(t, err, kv.ErrConflict)

	got, err := s.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, json.RawMessage(`2`), got.Value)
	require.Equal(t, int64(2), got.Version)
}

func TestStoreTTL(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, "")
	key := testKey()

	_, err := s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`1`)}, kv.WithTTL(time.Minute))
	require.NoError(t, err)
	require.Equal(t, time.Minute, r.TTL(key.String()))

	r.FastForward(30 * time.Second)
	_, err = s.Get(ctx, key)
	require.NoError(t, err)

	r.FastForward(31 * time.Second)
	_, err = s.Get(ctx, key)
	require.ErrorIs(t, err, kv.ErrNotFound)

	// a set without a ttl removes an earlier ttl.
	_, err = s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`1`)}, kv.WithTTL(time.Minute))
	require.NoError(t, err)
	_, err = s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`2`)})
	require.NoError(t, err)
	require.Zero(t, r.TTL(key.String()))
	r.FastForward(2 * time.Minute)
	_, err = s.Get(ctx, key)
	require.NoError(t, err)
}

func TestStoreConditionalDelete(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t, "")
	key := testKey()

	// a missing key is at version 0.
	require.ErrorIs(t, s.Delete(ctx, key, kv.WithExpected(1)), kv.ErrConflict)
	require.NoError(t, s.Delete(ctx, key, kv.WithExpected(0)))

	_, err := s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`1`)})
	require.NoError(t, err)
	_, err = s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`2`)})
	require.NoError(t, err)

	require.ErrorIs(t, s.Delete(ctx, key, kv.WithExpected(1)), kv.ErrConflict)
	_, err = s.Get(ctx, key)
	require.NoError(t, err)

	require.NoError(t, s.Delete(ctx, key, kv.WithExpected(2)))
	_, err = s.Get(ctx, key)
	require.ErrorIs(t, err, kv.ErrNotFound)
}

func TestStoreValidatesBeforeRedis(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, "")
	bad := kv.Key{
		Tenant:    domain.NewTenant(uuid.New(), uuid.New()),
		Namespace: "a/b",
		Key:       "k",
	}

	_, err := s.Get(ctx, bad)
	require.ErrorIs(t, err, kv.ErrInvalidNamespace)
	_, err = s.Set(ctx, kv.Entry{Key: bad, Value: json.RawMessage(`1`)})
	require.ErrorIs(t, err, kv.ErrInvalidNamespace)
	require.ErrorIs(t, s.Delete(ctx, bad), kv.ErrInvalidNamespace)

	require.Empty(t, r.Keys())
}

func TestRedisKeyAllocs(t *testing.T) {
	s := &Store{prefix: "kv:"}
	key := testKey()
	require.Equal(t, "kv:"+key.String(), s.redisKey(key))
	require.Equal(t, 1.0, testing.AllocsPerRun(100, func() {
		_ = s.redisKey(key)
	}))
}

func BenchmarkStoreGet(b *testing.B) {
	ctx := context.Background()
	s, _ := newStore(b, "kv:")
	key := testKey()
	_, err := s.Set(ctx, kv.Entry{Key: key, Value: json.RawMessage(`{"a":1}`)})
	require.NoError(b, err)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.Get(ctx, key); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStoreSet(b *testing.B) {
	ctx := context.Background()
	s, _ := newStore(b, "kv:")
	e := kv.Entry{Key: testKey(), Value: json.RawMessage(`{"a":1}`)}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.Set(ctx, e); err != nil {
			b.Fatal(err)
		}
	}
}

type recordingHook struct {
	ops []kv.Operation
}

func (h *recordingHook) OnOperation(ctx context.Context, op kv.Operation) {
	h.ops = append(h.ops, op)
}

func TestStoreInstrumented(t *testing.T) {
	ctx := context.Background()
	inner, _ := newStore(t, "kv:")
	hook := &recordingHook{}
	s := kv.Instrument(inner, hook)
	key := testKey()
	value := json.RawMessage(`{"a":1,"b":2}`)

	_, err := s.Get(ctx, key)
	require.ErrorIs(t, err, kv.ErrNotFound)

	_, err = s.Set(ctx, kv.Entry{Key: key, Value: value}, kv.WithExpected(0))
	require.NoError(t, err)

	_, err = s.Set(ctx, kv.Entry{Key: key, Value: value}, kv.WithExpected(0))
	require.ErrorIs(t, err, kv.ErrConflict)

	got, err := s.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, value, got.Value)

	require.NoError(t, s.Delete(ctx, key, kv.WithExpected(1)))

	bad := kv.Key{
		Tenant:    key.Tenant,
		Namespace: "a/b",
		Key:       "k",
	}
	_, err = s.Get(ctx, bad)
	require.ErrorIs(t, err, kv.ErrInvalidNamespace)

	n := int64(len(value))
	want := []kv.Operation{
		{Tenant: key.Tenant, Namespace: key.Namespace, Op: kv.OpGet, Result: kv.ResultNotFound},
		{Tenant: key.Tenant, Namespace: key.Namespace, Op: kv.OpSet, Result: kv.ResultOK, Bytes: n},
		{Tenant: key.Tenant, Namespace: key.Namespace, Op: kv.OpSet, Result: kv.ResultConflict},
		{Tenant: key.Tenant, Namespace: key.Namespace, Op: kv.OpGet, Result: kv.ResultOK, Bytes: n},
		{Tenant: key.Tenant, Namespace: key.Namespace, Op: kv.OpDelete, Result: kv.ResultOK},
		{Tenant: key.Tenant, Namespace: "a/b", Op: kv.OpGet, Result: kv.ResultError},
	}
	require.Equal(t, want, hook.ops)
}
