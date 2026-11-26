package loops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state/kv/kvredis"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func newInboxStore(t testing.TB) (InboxStore, *miniredis.Miniredis) {
	t.Helper()
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	return NewInboxStore(kvredis.New(rc, "")), r
}

func testInbox() Inbox {
	return Inbox{
		Loop:     testID(),
		OpenedBy: ulid.Make(),
		Name:     "inbox-1",
	}
}

func testEvent(n int) InboxEvent {
	return InboxEvent{
		ID:    ulid.Make(),
		Event: json.RawMessage(fmt.Sprintf(`{"name":"app/test","data":{"n":%d}}`, n)),
	}
}

// onlyKey returns the one redis key in r.  each test writes one inbox.
func onlyKey(t *testing.T, r *miniredis.Miniredis) string {
	t.Helper()
	keys := r.Keys()
	require.Len(t, keys, 1)
	return keys[0]
}

func inboxIDs(t *testing.T, s InboxStore, in Inbox) []ulid.ULID {
	t.Helper()
	got, err := s.Peek(context.Background(), in, 0)
	require.NoError(t, err)
	ids := []ulid.ULID{}
	for _, e := range got {
		ids = append(ids, e.ID)
	}
	return ids
}

func TestInboxPushPeekRemove(t *testing.T) {
	ctx := context.Background()
	s, _ := newInboxStore(t)
	in := testInbox()

	// a missing inbox has no events.
	got, err := s.Peek(ctx, in, 0)
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, s.Remove(ctx, in, []ulid.ULID{ulid.Make()}))

	var pushed []InboxEvent
	for i := range 5 {
		ev := testEvent(i)
		require.NoError(t, s.Push(ctx, in, ev))
		pushed = append(pushed, ev)
	}

	all, err := s.Peek(ctx, in, 0)
	require.NoError(t, err)
	require.Equal(t, pushed, all)

	first, err := s.Peek(ctx, in, 2)
	require.NoError(t, err)
	require.Equal(t, pushed[:2], first)

	// an event that is pushed again before it is read is in the inbox once.
	require.NoError(t, s.Push(ctx, in, pushed[1]))
	require.Len(t, inboxIDs(t, s, in), 5)

	// removing some ids keeps the rest in order.  ids that are gone are
	// ignored.
	require.NoError(t, s.Remove(ctx, in, []ulid.ULID{pushed[3].ID, pushed[0].ID}))
	require.NoError(t, s.Remove(ctx, in, []ulid.ULID{pushed[3].ID, ulid.Make()}))
	require.NoError(t, s.Remove(ctx, in, nil))
	require.Equal(t, []ulid.ULID{pushed[1].ID, pushed[2].ID, pushed[4].ID}, inboxIDs(t, s, in))

	// an event that is pushed again after it was removed goes to the end.
	require.NoError(t, s.Push(ctx, in, pushed[0]))
	require.Equal(t, []ulid.ULID{pushed[1].ID, pushed[2].ID, pushed[4].ID, pushed[0].ID}, inboxIDs(t, s, in))
}

func TestInboxEventBytes(t *testing.T) {
	ctx := context.Background()
	s, _ := newInboxStore(t)
	in := testInbox()

	// the stored bytes are the pushed bytes, with white space and HTML
	// characters as they were.
	ev := InboxEvent{
		ID:    ulid.Make(),
		Event: json.RawMessage("{ \"data\" : {\"html\": \"<a href=\\\"x\\\">&</a>\"},\n \"n\": 1 }"),
	}
	require.NoError(t, s.Push(ctx, in, ev))
	got, err := s.Peek(ctx, in, 0)
	require.NoError(t, err)
	require.Equal(t, []InboxEvent{ev}, got)
}

func TestInboxCountCap(t *testing.T) {
	ctx := context.Background()
	s, _ := newInboxStore(t)
	in := testInbox()

	var first InboxEvent
	for i := range consts.MaxLoopInboxEvents {
		ev := testEvent(i)
		if i == 0 {
			first = ev
		}
		require.NoError(t, s.Push(ctx, in, ev))
	}
	require.ErrorIs(t, s.Push(ctx, in, testEvent(99)), ErrInboxFull)

	// an event that is already in a full inbox is not an error.
	require.NoError(t, s.Push(ctx, in, first))
	require.Len(t, inboxIDs(t, s, in), consts.MaxLoopInboxEvents)

	// a remove makes room for one more event.
	require.NoError(t, s.Remove(ctx, in, []ulid.ULID{first.ID}))
	require.NoError(t, s.Push(ctx, in, testEvent(100)))
	require.ErrorIs(t, s.Push(ctx, in, testEvent(101)), ErrInboxFull)
}

func TestInboxByteCap(t *testing.T) {
	ctx := context.Background()
	s, _ := newInboxStore(t)
	in := testInbox()

	// count events of this size are exactly MaxLoopInboxSize.
	const count = 4
	const size = consts.MaxLoopInboxSize / count
	big := func() InboxEvent {
		return InboxEvent{
			ID:    ulid.Make(),
			Event: json.RawMessage(`"` + strings.Repeat("a", size-2) + `"`),
		}
	}
	small := func() InboxEvent {
		return InboxEvent{
			ID:    ulid.Make(),
			Event: json.RawMessage(`1`),
		}
	}
	first := big()
	require.NoError(t, s.Push(ctx, in, first))
	for range count - 1 {
		require.NoError(t, s.Push(ctx, in, big()))
	}
	require.ErrorIs(t, s.Push(ctx, in, big()), ErrInboxFull)
	require.ErrorIs(t, s.Push(ctx, in, small()), ErrInboxFull)
	require.Len(t, inboxIDs(t, s, in), count)

	// a remove gives the bytes of the event back.
	require.NoError(t, s.Remove(ctx, in, []ulid.ULID{first.ID}))
	require.NoError(t, s.Push(ctx, in, big()))
	require.ErrorIs(t, s.Push(ctx, in, small()), ErrInboxFull)
}

func TestInboxDelete(t *testing.T) {
	ctx := context.Background()
	s, r := newInboxStore(t)
	in := testInbox()

	// a missing inbox is not an error.
	require.NoError(t, s.Delete(ctx, in))

	require.NoError(t, s.Push(ctx, in, testEvent(1)))
	require.Len(t, r.Keys(), 1)
	require.NoError(t, s.Delete(ctx, in))
	require.Empty(t, r.Keys())
	require.Empty(t, inboxIDs(t, s, in))

	// a push after a delete starts a new inbox.
	ev := testEvent(2)
	require.NoError(t, s.Push(ctx, in, ev))
	require.Equal(t, []ulid.ULID{ev.ID}, inboxIDs(t, s, in))
}

func TestInboxTTL(t *testing.T) {
	ctx := context.Background()
	s, r := newInboxStore(t)
	in := testInbox()

	ev := testEvent(1)
	require.NoError(t, s.Push(ctx, in, ev))
	key := onlyKey(t, r)
	require.Equal(t, InboxTTL, r.TTL(key))

	r.FastForward(time.Hour)
	kept := testEvent(2)
	require.NoError(t, s.Push(ctx, in, kept))
	require.Equal(t, InboxTTL, r.TTL(key))

	r.FastForward(time.Hour)
	require.NoError(t, s.Remove(ctx, in, []ulid.ULID{ev.ID}))
	require.Equal(t, InboxTTL, r.TTL(key))

	// a push and a remove that write nothing leave the ttl as it is.
	r.FastForward(time.Hour)
	require.NoError(t, s.Push(ctx, in, kept))
	require.NoError(t, s.Remove(ctx, in, []ulid.ULID{ulid.Make()}))
	require.Equal(t, InboxTTL-time.Hour, r.TTL(key))
}

func TestInboxKey(t *testing.T) {
	ctx := context.Background()
	s, _ := newInboxStore(t)
	in := testInbox()

	k, err := in.kvKey()
	require.NoError(t, err)
	require.NoError(t, k.Validate())
	require.Equal(t, InboxNamespace, k.Namespace)
	require.Equal(t, in.Loop.Tenant, k.Tenant)
	require.Len(t, k.Key, 32)

	// a name that is not a valid kv key part still gives a valid key.
	odd := in
	odd.Name = "inbox / with spaces and ünïcode " + strings.Repeat("x", 100)
	oddKey, err := odd.kvKey()
	require.NoError(t, err)
	require.NoError(t, oddKey.Validate())

	opener := in
	opener.OpenedBy = ulid.Make()
	name := in
	name.Name = "inbox-2"
	loopKey := in
	loopKey.Loop.Key = "app-2"
	fn := in
	fn.Loop.FunctionID = uuid.New()

	require.NoError(t, s.Push(ctx, in, testEvent(1)))
	for _, other := range []Inbox{opener, name, loopKey, fn, odd} {
		otherKey, err := other.kvKey()
		require.NoError(t, err)
		require.NotEqual(t, k, otherKey)
		require.Empty(t, inboxIDs(t, s, other))
	}

	// an inbox does not share a key with the instance record.
	require.NotEqual(t, in.Loop.kvKey(), k)
}

func TestInboxInvalid(t *testing.T) {
	ctx := context.Background()
	s, r := newInboxStore(t)
	in := testInbox()

	noName := in
	noName.Name = ""
	noOpener := in
	noOpener.OpenedBy = ulid.ULID{}
	for _, tc := range []struct {
		in   Inbox
		want error
	}{
		{
			in:   noName,
			want: ErrEmptyInboxName,
		},
		{
			in:   noOpener,
			want: ErrNoInboxOpener,
		},
	} {
		require.ErrorIs(t, s.Push(ctx, tc.in, testEvent(1)), tc.want)
		_, err := s.Peek(ctx, tc.in, 0)
		require.ErrorIs(t, err, tc.want)
		require.ErrorIs(t, s.Remove(ctx, tc.in, []ulid.ULID{ulid.Make()}), tc.want)
		require.ErrorIs(t, s.Delete(ctx, tc.in), tc.want)
	}

	noID := testEvent(1)
	noID.ID = ulid.ULID{}
	require.ErrorIs(t, s.Push(ctx, in, noID), ErrInvalidInboxEvent)
	for _, raw := range []json.RawMessage{nil, {}, json.RawMessage(`{bad`)} {
		ev := InboxEvent{
			ID:    ulid.Make(),
			Event: raw,
		}
		require.ErrorIs(t, s.Push(ctx, in, ev), ErrInvalidInboxEvent)
	}
	require.Empty(t, r.Keys())
}

// TestInboxConcurrentPush checks that a burst of pushes to one inbox fills
// it to consts.MaxLoopInboxEvents and that every other push gets
// ErrInboxFull.
func TestInboxConcurrentPush(t *testing.T) {
	ctx := context.Background()
	s, _ := newInboxStore(t)
	in := testInbox()

	const pushers = 2 * consts.MaxLoopInboxEvents
	events := make([]InboxEvent, pushers)
	errs := make([]error, pushers)
	for i := range events {
		events[i] = testEvent(i)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range pushers {
		wg.Go(func() {
			<-start
			errs[i] = s.Push(ctx, in, events[i])
		})
	}
	close(start)
	wg.Wait()

	added := map[ulid.ULID]bool{}
	full := 0
	for i, err := range errs {
		switch {
		case err == nil:
			added[events[i].ID] = true
		case errors.Is(err, ErrInboxFull):
			full++
		default:
			t.Fatalf("push %d: %v", i, err)
		}
	}
	require.Len(t, added, consts.MaxLoopInboxEvents)
	require.Equal(t, pushers-consts.MaxLoopInboxEvents, full)

	got := inboxIDs(t, s, in)
	require.Len(t, got, consts.MaxLoopInboxEvents)
	for _, id := range got {
		require.True(t, added[id], "event %s is in the inbox but its push failed", id)
	}
}
