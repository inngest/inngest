package kvredis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/inngest/inngest/pkg/execution/state/kv"
	"github.com/inngest/inngest/pkg/execution/state/kv/kvtest"
	"github.com/stretchr/testify/require"
)

const listPrefix = "kv:"

func testItem(n int) kv.Item {
	return kv.Item{
		ID:    fmt.Sprintf("item-%d", n),
		Value: json.RawMessage(fmt.Sprintf(`{"n":%d}`, n)),
	}
}

// listFields returns every field of the list hash, or nil when the hash is
// missing.  a test compares two results to check that a call wrote nothing.
func listFields(t *testing.T, r *miniredis.Miniredis, key string) map[string]string {
	t.Helper()
	if !r.Exists(key) {
		return nil
	}
	names, err := r.HKeys(key)
	require.NoError(t, err)
	out := map[string]string{}
	for _, name := range names {
		out[name] = r.HGet(key, name)
	}
	return out
}

// push calls Push and returns only the error.  a test that checks whether
// the item was added calls Push itself.
func push(s kv.ListStore, k kv.Key, it kv.Item, opts ...kv.Option) error {
	_, err := s.Push(context.Background(), k, it, opts...)
	return err
}

func TestListContract(t *testing.T) {
	kvtest.RunListStore(t, func(t *testing.T) kv.ListStore {
		s, _ := newStore(t, listPrefix)
		return s
	})
}

func listIDs(t *testing.T, s *Store, k kv.Key) []string {
	t.Helper()
	got, err := s.Peek(context.Background(), k, 0)
	require.NoError(t, err)
	out := []string{}
	for _, it := range got {
		out = append(out, it.ID)
	}
	return out
}

func TestListPushPeek(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, listPrefix)
	k := testKey()
	rk := s.listKey(k)

	// more than 9 items, so the order is by number and not by text.
	var pushed []kv.Item
	for i := range 12 {
		it := testItem(i)
		require.NoError(t, push(s, k, it, kv.WithTTL(time.Hour)))
		pushed = append(pushed, it)
	}

	all, err := s.Peek(ctx, k, 0)
	require.NoError(t, err)
	require.Equal(t, pushed, all)

	first, err := s.Peek(ctx, k, 2)
	require.NoError(t, err)
	require.Equal(t, all[:2], first)

	neg, err := s.Peek(ctx, k, -1)
	require.NoError(t, err)
	require.Equal(t, all, neg)

	more, err := s.Peek(ctx, k, 20)
	require.NoError(t, err)
	require.Equal(t, all, more)

	// a peek does not write.
	before := listFields(t, r, rk)
	r.FastForward(time.Minute)
	_, err = s.Peek(ctx, k, 0)
	require.NoError(t, err)
	require.Equal(t, before, listFields(t, r, rk))
	require.Equal(t, time.Hour-time.Minute, r.TTL(rk))
}

func TestListValueBytes(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t, listPrefix)
	k := testKey()

	// the stored bytes are the pushed bytes, with white space, HTML
	// characters, and ':' as they were.  an empty value is stored too.
	items := []kv.Item{
		{
			ID:    "a",
			Value: json.RawMessage("{ \"html\": \"<a href=\\\"x:y\\\">&</a>\",\n \"n\": 1 }"),
		},
		{
			ID:    "b",
			Value: json.RawMessage{},
		},
		{
			ID:    "c",
			Value: json.RawMessage{0, 255, ':', 0},
		},
	}
	for _, it := range items {
		require.NoError(t, push(s, k, it))
	}
	got, err := s.Peek(ctx, k, 0)
	require.NoError(t, err)
	require.Equal(t, items, got)
}

func TestListPushDuplicate(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, listPrefix)
	k := testKey()
	rk := s.listKey(k)

	it := testItem(1)
	require.NoError(t, push(s, k, it))
	require.NoError(t, push(s, k, testItem(2)))
	before := listFields(t, r, rk)

	added, err := s.Push(ctx, k, it)
	require.NoError(t, err)
	require.False(t, added)
	require.Equal(t, before, listFields(t, r, rk))

	// the same ID with other bytes is still a duplicate.
	dup := it
	dup.Value = json.RawMessage(`{"other":true}`)
	added, err = s.Push(ctx, k, dup)
	require.NoError(t, err)
	require.False(t, added)
	require.Equal(t, before, listFields(t, r, rk))

	got, err := s.Peek(ctx, k, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, it, got[0])
}

func TestListMaxItems(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, listPrefix)
	k := testKey()
	rk := s.listKey(k)
	limit := kv.WithMaxItems(3)

	for i := range 3 {
		require.NoError(t, push(s, k, testItem(i), limit))
	}
	before := listFields(t, r, rk)

	require.ErrorIs(t, push(s, k, testItem(99), limit), kv.ErrFull)
	require.Equal(t, before, listFields(t, r, rk))

	// an item that is already in a full list is not an error.
	require.NoError(t, push(s, k, testItem(0), limit))
	require.Equal(t, before, listFields(t, r, rk))

	// a remove makes room for one more item.
	require.NoError(t, s.Remove(ctx, k, []string{testItem(0).ID}))
	require.NoError(t, push(s, k, testItem(100), limit))
	require.ErrorIs(t, push(s, k, testItem(101), limit), kv.ErrFull)

	// a push with no limit adds to a list that is past the limit of an
	// earlier push.
	require.NoError(t, push(s, k, testItem(102)))
	require.Len(t, listIDs(t, s, k), 4)
}

func TestListMaxBytes(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, listPrefix)
	k := testKey()
	rk := s.listKey(k)

	// count values of this size are exactly the limit.
	const count = 4
	const size = 1024 * 1024
	limit := kv.WithMaxBytes(count * size)
	n := 0
	big := func() kv.Item {
		n++
		return kv.Item{
			ID:    fmt.Sprintf("big-%d", n),
			Value: json.RawMessage(strings.Repeat("a", size)),
		}
	}
	small := kv.Item{
		ID:    "small",
		Value: json.RawMessage(`1`),
	}
	first := big()
	require.NoError(t, push(s, k, first, limit))
	for range count - 1 {
		require.NoError(t, push(s, k, big(), limit))
	}
	require.Equal(t, strconv.Itoa(count*size), r.HGet(rk, fieldSize))
	before := listFields(t, r, rk)

	require.ErrorIs(t, push(s, k, big(), limit), kv.ErrFull)
	require.ErrorIs(t, push(s, k, small, limit), kv.ErrFull)
	require.Equal(t, before, listFields(t, r, rk))

	// a remove gives the bytes of the item back.
	require.NoError(t, s.Remove(ctx, k, []string{first.ID}))
	require.Equal(t, strconv.Itoa((count-1)*size), r.HGet(rk, fieldSize))
	require.NoError(t, push(s, k, big(), limit))
	require.ErrorIs(t, push(s, k, small, limit), kv.ErrFull)
}

func TestListRemove(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, listPrefix)
	k := testKey()
	rk := s.listKey(k)

	var pushed []string
	for i := range 5 {
		it := testItem(i)
		require.NoError(t, push(s, k, it))
		pushed = append(pushed, it.ID)
	}

	// removing some ids keeps the rest in order.  an id that is given twice
	// is removed once.
	require.NoError(t, s.Remove(ctx, k, []string{pushed[3], pushed[0], pushed[3]}))
	require.Equal(t, []string{pushed[1], pushed[2], pushed[4]}, listIDs(t, s, k))

	// removing ids that are gone writes nothing.
	before := listFields(t, r, rk)
	require.NoError(t, s.Remove(ctx, k, []string{pushed[3], pushed[0], "other"}))
	require.NoError(t, s.Remove(ctx, k, nil))
	require.Equal(t, before, listFields(t, r, rk))

	// removing the rest leaves the counters, with no value bytes.
	require.NoError(t, s.Remove(ctx, k, []string{pushed[1], pushed[2], pushed[4]}))
	require.Empty(t, listIDs(t, s, k))
	require.Equal(t, map[string]string{
		fieldSeq:  "5",
		fieldSize: "0",
	}, listFields(t, r, rk))

	// a remove that is done again on an empty list writes nothing.
	require.NoError(t, s.Remove(ctx, k, pushed))
	require.Equal(t, "5", r.HGet(rk, fieldSeq))
}

// TestListOrderAfterRemove checks that an item that is pushed after a remove
// goes to the end, and that an item that is pushed again after it was
// removed goes to the end too.
func TestListOrderAfterRemove(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t, listPrefix)
	k := testKey()
	i1, i2, i3 := testItem(1), testItem(2), testItem(3)

	require.NoError(t, push(s, k, i1))
	require.NoError(t, push(s, k, i2))
	require.NoError(t, s.Remove(ctx, k, []string{i1.ID}))
	require.NoError(t, push(s, k, i3))
	require.Equal(t, []string{i2.ID, i3.ID}, listIDs(t, s, k))

	require.NoError(t, push(s, k, i1))
	require.Equal(t, []string{i2.ID, i3.ID, i1.ID}, listIDs(t, s, k))

	// a list that was emptied takes new items.
	require.NoError(t, s.Remove(ctx, k, []string{i1.ID, i2.ID, i3.ID}))
	require.NoError(t, push(s, k, i2))
	require.Equal(t, []string{i2.ID}, listIDs(t, s, k))
}

func TestListMissing(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, listPrefix)
	k := testKey()

	got, err := s.Peek(ctx, k, 0)
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, s.Remove(ctx, k, []string{"a"}))
	require.NoError(t, s.Clear(ctx, k))
	require.False(t, r.Exists(s.listKey(k)))
}

func TestListClear(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, listPrefix)
	k := testKey()
	rk := s.listKey(k)

	require.NoError(t, push(s, k, testItem(1)))
	require.True(t, r.Exists(rk))

	require.NoError(t, s.Clear(ctx, k))
	require.False(t, r.Exists(rk))
	require.Empty(t, listIDs(t, s, k))

	// a push after a clear starts a new list.
	require.NoError(t, push(s, k, testItem(2)))
	require.Equal(t, []string{testItem(2).ID}, listIDs(t, s, k))
}

func TestListTTL(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, listPrefix)
	k := testKey()
	rk := s.listKey(k)
	ttl := kv.WithTTL(24 * time.Hour)

	require.NoError(t, push(s, k, testItem(1), ttl))
	require.Equal(t, 24*time.Hour, r.TTL(rk))

	r.FastForward(time.Hour)
	require.NoError(t, push(s, k, testItem(2), ttl))
	require.Equal(t, 24*time.Hour, r.TTL(rk))

	r.FastForward(time.Hour)
	require.NoError(t, s.Remove(ctx, k, []string{testItem(1).ID}, ttl))
	require.Equal(t, 24*time.Hour, r.TTL(rk))

	// a push and a remove that write nothing leave the ttl as it is.
	r.FastForward(time.Hour)
	require.NoError(t, push(s, k, testItem(2), ttl))
	require.NoError(t, s.Remove(ctx, k, []string{"other"}, ttl))
	require.Equal(t, 23*time.Hour, r.TTL(rk))

	// a write without a ttl removes the expiry.
	require.NoError(t, push(s, k, testItem(3)))
	require.Equal(t, time.Duration(0), r.TTL(rk))
	require.NoError(t, push(s, k, testItem(4), ttl))
	require.NoError(t, s.Remove(ctx, k, []string{testItem(4).ID}))
	require.Equal(t, time.Duration(0), r.TTL(rk))
}

// TestListAndValue checks that a list and a value under the same key do not
// change each other.
func TestListAndValue(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t, listPrefix)
	k := testKey()
	value := json.RawMessage(`{"a":1}`)

	_, err := s.Set(ctx, kv.Entry{
		Key:   k,
		Value: value,
	})
	require.NoError(t, err)
	require.Empty(t, listIDs(t, s, k))

	require.NoError(t, push(s, k, testItem(1)))
	got, err := s.Get(ctx, k)
	require.NoError(t, err)
	require.Equal(t, value, got.Value)
	require.Equal(t, int64(1), got.Version)

	require.NoError(t, s.Delete(ctx, k))
	require.Equal(t, []string{testItem(1).ID}, listIDs(t, s, k))

	_, err = s.Set(ctx, kv.Entry{
		Key:   k,
		Value: value,
	})
	require.NoError(t, err)
	require.NoError(t, s.Clear(ctx, k))
	_, err = s.Get(ctx, k)
	require.NoError(t, err)

	// another key has another list.
	other := k
	other.Key = "step-2"
	require.NoError(t, push(s, k, testItem(1)))
	require.Empty(t, listIDs(t, s, other))
}

func TestListValidatesBeforeRedis(t *testing.T) {
	ctx := context.Background()
	s, r := newStore(t, listPrefix)
	k := testKey()

	bad := k
	bad.Namespace = "has/slash"
	require.ErrorIs(t, push(s, bad, testItem(1)), kv.ErrInvalidNamespace)
	_, err := s.Peek(ctx, bad, 0)
	require.ErrorIs(t, err, kv.ErrInvalidNamespace)
	require.ErrorIs(t, s.Remove(ctx, bad, []string{"a"}), kv.ErrInvalidNamespace)
	require.ErrorIs(t, s.Clear(ctx, bad), kv.ErrInvalidNamespace)

	// a counter field name is not a valid item id, so no call can read or
	// write a counter as an item.
	for _, id := range []string{"", fieldSeq, fieldSize, "a:b", strings.Repeat("x", kv.MaxPartLen+1)} {
		it := kv.Item{
			ID:    id,
			Value: json.RawMessage(`1`),
		}
		require.ErrorIs(t, push(s, k, it), kv.ErrInvalidItemID)
		require.ErrorIs(t, s.Remove(ctx, k, []string{"a", id}), kv.ErrInvalidItemID)
	}
	require.Empty(t, r.Keys())
}

func TestListInstrumented(t *testing.T) {
	ctx := context.Background()
	inner, _ := newStore(t, listPrefix)
	var ops []kv.Operation
	s := kv.InstrumentList(inner, kv.HookFunc(func(_ context.Context, op kv.Operation) {
		ops = append(ops, op)
	}))
	k := testKey()
	it := testItem(1)

	require.NoError(t, push(s, k, it, kv.WithMaxItems(1)))
	require.NoError(t, push(s, k, it, kv.WithMaxItems(1)))
	require.ErrorIs(t, push(s, k, testItem(2), kv.WithMaxItems(1)), kv.ErrFull)
	_, err := s.Peek(ctx, k, 0)
	require.NoError(t, err)
	require.NoError(t, s.Remove(ctx, k, []string{it.ID}))
	require.NoError(t, s.Clear(ctx, k))

	n := int64(len(it.Value))
	want := []kv.Operation{
		{
			Op:     kv.OpPush,
			Result: kv.ResultOK,
			Bytes:  n,
		},
		{
			Op:     kv.OpPush,
			Result: kv.ResultDuplicate,
		},
		{
			Op:     kv.OpPush,
			Result: kv.ResultFull,
		},
		{
			Op:     kv.OpPeek,
			Result: kv.ResultOK,
			Bytes:  n,
		},
		{
			Op:     kv.OpRemove,
			Result: kv.ResultOK,
		},
		{
			Op:     kv.OpClear,
			Result: kv.ResultOK,
		},
	}
	for i := range want {
		want[i].Tenant = k.Tenant
		want[i].Namespace = k.Namespace
	}
	require.Equal(t, want, ops)
}
