// Package kvtest contains tests that every kv backend must pass.  a backend
// runs them from its own tests with a constructor for a new, empty store.
package kvtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/domain"
	"github.com/inngest/inngest/pkg/execution/state/kv"
	"github.com/stretchr/testify/require"
)

// RunListStore runs the kv.ListStore contract against the store that
// newStore returns.  the backend tests expiry itself, because each backend
// moves time in another way.
func RunListStore(t *testing.T, newStore func(t *testing.T) kv.ListStore) {
	tests := map[string]func(*testing.T, kv.ListStore){
		"Order":                listOrder,
		"ValueBytes":           listValueBytes,
		"Duplicate":            listDuplicate,
		"MaxItems":             listMaxItems,
		"MaxBytes":             listMaxBytes,
		"Missing":              listMissing,
		"Clear":                listClear,
		"ConcurrentMaxItems":   listConcurrentMaxItems,
		"ConcurrentMaxBytes":   listConcurrentMaxBytes,
		"ConcurrentDuplicate":  listConcurrentDuplicate,
		"ConcurrentPushRemove": listConcurrentPushRemove,
	}
	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			fn(t, newStore(t))
		})
	}
}

func newKey() kv.Key {
	return kv.Key{
		Tenant:    domain.NewTenant(uuid.New(), uuid.New()),
		Namespace: "list",
		Key:       "list-1",
	}
}

func item(n int) kv.Item {
	return kv.Item{
		ID:    fmt.Sprintf("item-%d", n),
		Value: fmt.Appendf(nil, `{"n":%d}`, n),
	}
}

func bigItem(n, size int) kv.Item {
	return kv.Item{
		ID:    fmt.Sprintf("big-%d", n),
		Value: bytes.Repeat([]byte{'a'}, size),
	}
}

func push(t *testing.T, s kv.ListStore, k kv.Key, it kv.Item, opts ...kv.Option) bool {
	t.Helper()
	added, err := s.Push(context.Background(), k, it, opts...)
	require.NoError(t, err)
	return added
}

func ids(t *testing.T, s kv.ListStore, k kv.Key) []string {
	t.Helper()
	got, err := s.Peek(context.Background(), k, 0)
	require.NoError(t, err)
	out := []string{}
	for _, it := range got {
		out = append(out, it.ID)
	}
	return out
}

// requireItems compares ids and value bytes.  a nil value and an empty
// value are equal, so a backend can return either one.
func requireItems(t *testing.T, want, got []kv.Item) {
	t.Helper()
	require.Len(t, got, len(want))
	for i := range want {
		require.Equal(t, want[i].ID, got[i].ID)
		require.True(t, bytes.Equal(want[i].Value, got[i].Value), "item %s has other bytes", want[i].ID)
	}
}

func listOrder(t *testing.T, s kv.ListStore) {
	ctx := context.Background()
	k := newKey()

	// more than 9 items, so a backend that sorts arrival numbers as text
	// fails.
	var pushed []kv.Item
	for i := range 12 {
		it := item(i)
		require.True(t, push(t, s, k, it))
		pushed = append(pushed, it)
	}

	all, err := s.Peek(ctx, k, 0)
	require.NoError(t, err)
	requireItems(t, pushed, all)

	first, err := s.Peek(ctx, k, 2)
	require.NoError(t, err)
	requireItems(t, pushed[:2], first)

	neg, err := s.Peek(ctx, k, -1)
	require.NoError(t, err)
	requireItems(t, pushed, neg)

	more, err := s.Peek(ctx, k, 20)
	require.NoError(t, err)
	requireItems(t, pushed, more)
}

func listValueBytes(t *testing.T, s kv.ListStore) {
	ctx := context.Background()
	k := newKey()

	// a store returns the pushed bytes as they were.  this includes white
	// space and key order in JSON, HTML characters, bytes that are not
	// UTF-8, a NUL escape that postgres jsonb rejects, and an empty value.
	items := []kv.Item{
		{
			ID:    "json",
			Value: []byte("{ \"z\": 1,\n  \"a\" : \"<a href=\\\"x:y\\\">&</a>\" }"),
		},
		{
			ID:    "nul",
			Value: []byte(`{"s":"a\u0000b"}`),
		},
		{
			ID:    "binary",
			Value: []byte{0, 255, ':', 0},
		},
		{
			ID:    "empty",
			Value: []byte{},
		},
	}
	for _, it := range items {
		require.True(t, push(t, s, k, it))
	}
	got, err := s.Peek(ctx, k, 0)
	require.NoError(t, err)
	requireItems(t, items, got)
}

func listDuplicate(t *testing.T, s kv.ListStore) {
	ctx := context.Background()
	k := newKey()
	a, b := item(1), item(2)

	require.True(t, push(t, s, k, a))
	require.True(t, push(t, s, k, b))

	// the same ID with other bytes is a duplicate, and the first bytes stay.
	dup := a
	dup.Value = []byte(`{"other":true}`)
	require.False(t, push(t, s, k, a))
	require.False(t, push(t, s, k, dup))
	got, err := s.Peek(ctx, k, 0)
	require.NoError(t, err)
	requireItems(t, []kv.Item{a, b}, got)

	// after a remove, the same ID is added again at the end.
	require.NoError(t, s.Remove(ctx, k, []string{a.ID}))
	require.True(t, push(t, s, k, a))
	require.Equal(t, []string{b.ID, a.ID}, ids(t, s, k))
}

func listMaxItems(t *testing.T, s kv.ListStore) {
	ctx := context.Background()
	k := newKey()
	limit := kv.WithMaxItems(3)

	for i := range 3 {
		require.True(t, push(t, s, k, item(i), limit))
	}
	_, err := s.Push(ctx, k, item(99), limit)
	require.ErrorIs(t, err, kv.ErrFull)

	// an item that is already in a full list is a duplicate, not an error.
	require.False(t, push(t, s, k, item(0), limit))
	require.Equal(t, []string{item(0).ID, item(1).ID, item(2).ID}, ids(t, s, k))

	// a remove makes room for one more item.
	require.NoError(t, s.Remove(ctx, k, []string{item(0).ID}))
	require.True(t, push(t, s, k, item(100), limit))
	_, err = s.Push(ctx, k, item(101), limit)
	require.ErrorIs(t, err, kv.ErrFull)
}

func listMaxBytes(t *testing.T, s kv.ListStore) {
	ctx := context.Background()
	k := newKey()

	// count values of this size are exactly the limit.
	const count = 4
	const size = 64 * 1024
	limit := kv.WithMaxBytes(count * size)
	small := kv.Item{
		ID:    "small",
		Value: []byte(`1`),
	}

	for i := range count {
		require.True(t, push(t, s, k, bigItem(i, size), limit))
	}
	_, err := s.Push(ctx, k, bigItem(count, size), limit)
	require.ErrorIs(t, err, kv.ErrFull)
	_, err = s.Push(ctx, k, small, limit)
	require.ErrorIs(t, err, kv.ErrFull)
	require.Len(t, ids(t, s, k), count)

	// a remove gives the bytes of the item back.
	require.NoError(t, s.Remove(ctx, k, []string{bigItem(0, size).ID}))
	require.True(t, push(t, s, k, bigItem(count+1, size), limit))
	_, err = s.Push(ctx, k, small, limit)
	require.ErrorIs(t, err, kv.ErrFull)
}

func listMissing(t *testing.T, s kv.ListStore) {
	ctx := context.Background()
	k := newKey()

	got, err := s.Peek(ctx, k, 0)
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, s.Remove(ctx, k, []string{"a"}))
	require.NoError(t, s.Clear(ctx, k))
	require.Empty(t, ids(t, s, k))
}

func listClear(t *testing.T, s kv.ListStore) {
	ctx := context.Background()
	k := newKey()

	require.True(t, push(t, s, k, item(1)))
	require.NoError(t, s.Clear(ctx, k))
	require.Empty(t, ids(t, s, k))

	// a push after a clear starts a new list.  the item that was cleared is
	// not a duplicate.
	require.True(t, push(t, s, k, item(1)))
	require.Equal(t, []string{item(1).ID}, ids(t, s, k))
}

// pushAll starts one push per item at the same time and returns each
// result.
func pushAll(s kv.ListStore, k kv.Key, items []kv.Item, opts ...kv.Option) ([]bool, []error) {
	added := make([]bool, len(items))
	errs := make([]error, len(items))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range items {
		wg.Go(func() {
			<-start
			added[i], errs[i] = s.Push(context.Background(), k, items[i], opts...)
		})
	}
	close(start)
	wg.Wait()
	return added, errs
}

// requireFull checks that exactly want items were added, that the rest
// returned ErrFull, and that the list contains each added item once.
func requireFull(t *testing.T, s kv.ListStore, k kv.Key, items []kv.Item, added []bool, errs []error, want int) {
	t.Helper()
	ok := map[string]bool{}
	full := 0
	for i, err := range errs {
		switch {
		case err == nil:
			require.True(t, added[i], "push %d returned false for a new id", i)
			ok[items[i].ID] = true
		case errors.Is(err, kv.ErrFull):
			full++
		default:
			t.Fatalf("push %d: %v", i, err)
		}
	}
	require.Len(t, ok, want)
	require.Equal(t, len(items)-want, full)

	got := ids(t, s, k)
	require.Len(t, got, want)
	seen := map[string]bool{}
	for _, id := range got {
		require.True(t, ok[id], "item %s is in the list but its push failed", id)
		require.False(t, seen[id], "item %s is in the list twice", id)
		seen[id] = true
	}
}

func listConcurrentMaxItems(t *testing.T, s kv.ListStore) {
	k := newKey()
	const limit = 50
	items := make([]kv.Item, 2*limit)
	for i := range items {
		items[i] = item(i)
	}
	added, errs := pushAll(s, k, items, kv.WithMaxItems(limit))
	requireFull(t, s, k, items, added, errs, limit)
}

func listConcurrentMaxBytes(t *testing.T, s kv.ListStore) {
	k := newKey()
	const fit = 40
	const size = 16 * 1024
	items := make([]kv.Item, fit+20)
	for i := range items {
		items[i] = bigItem(i, size)
	}
	added, errs := pushAll(s, k, items, kv.WithMaxBytes(fit*size))
	requireFull(t, s, k, items, added, errs, fit)
}

func listConcurrentDuplicate(t *testing.T, s kv.ListStore) {
	k := newKey()
	items := make([]kv.Item, 20)
	for i := range items {
		items[i] = item(1)
	}
	added, errs := pushAll(s, k, items)
	n := 0
	for i := range items {
		require.NoError(t, errs[i])
		if added[i] {
			n++
		}
	}
	require.Equal(t, 1, n, "more than one push added the same id")
	require.Equal(t, []string{item(1).ID}, ids(t, s, k))
}

// listConcurrentPushRemove runs pushes into a bounded list while one reader
// peeks, removes, and removes again, like a loop run that reads an inbox and
// is then retried.  every pushed item must reach the reader once, and an
// item that was removed must not come back.
func listConcurrentPushRemove(t *testing.T, s kv.ListStore) {
	ctx := context.Background()
	k := newKey()
	const writers = 4
	const perWriter = 100
	limit := kv.WithMaxItems(20)

	var wg sync.WaitGroup
	for w := range writers {
		wg.Go(func() {
			for i := range perWriter {
				it := item(w*perWriter + i)
				for {
					added, err := s.Push(ctx, k, it, limit)
					if errors.Is(err, kv.ErrFull) {
						runtime.Gosched()
						continue
					}
					if err != nil {
						t.Errorf("push %s: %v", it.ID, err)
						return
					}
					if !added {
						t.Errorf("push %s was a duplicate", it.ID)
					}
					break
				}
			}
		})
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	read := map[string]bool{}
	for {
		finished := false
		select {
		case <-done:
			finished = true
		default:
		}
		got, err := s.Peek(ctx, k, 7)
		require.NoError(t, err)
		if len(got) == 0 && finished {
			break
		}
		batch := make([]string, 0, len(got))
		for _, it := range got {
			require.False(t, read[it.ID], "item %s came back after it was removed", it.ID)
			read[it.ID] = true
			batch = append(batch, it.ID)
		}
		require.NoError(t, s.Remove(ctx, k, batch))
		require.NoError(t, s.Remove(ctx, k, batch))
		if len(got) == 0 {
			runtime.Gosched()
		}
	}
	require.Len(t, read, writers*perWriter)
}
