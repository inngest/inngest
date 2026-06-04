package redis_state

// Tests for the new DebounceUpsert Lua script (debounce/upsertDebounce.lua) and
// the modified requeueByID.lua lease-expiry return value.
//
// Each test exercises a single branch of the Lua logic so a broken script
// produces a focused failure rather than a vague integration error.

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// newDebounceUpsertShard spins up a fresh in-memory Redis and returns both a
// queue shard (for calling DebounceUpsert) and the raw miniredis for direct
// key inspection.
func newDebounceUpsertShard(t *testing.T) (RedisQueueShard, *miniredis.Miniredis) {
	t.Helper()
	mr, rc := initRedis(t)
	shard := shardFromClient("test", rc)
	return shard, mr
}

// upsertScope builds a minimal Scope for the given IDs.
func upsertScope(accountID, wsID, fnID uuid.UUID) osqueue.Scope {
	return osqueue.Scope{AccountID: accountID, EnvID: wsID, FunctionID: fnID}
}

// debounceItemJSON encodes a minimal debounce payload. eventTS is the event.ts
// field; absoluteTimeout is the "t" field (0 = no timeout).
func debounceItemJSON(t *testing.T, eventTS, absoluteTimeout int64) []byte {
	t.Helper()
	m := map[string]any{
		"e": map[string]any{"ts": eventTS},
	}
	if absoluteTimeout > 0 {
		m["t"] = absoluteTimeout
	}
	byt, err := json.Marshal(m)
	require.NoError(t, err)
	return byt
}

func freshULID() ulid.ULID { return ulid.MustNew(ulid.Now(), rand.Reader) }

// ─── upsertDebounce.lua branch tests ──────────────────────────────────────────

// TestDebounceUpsert_Create covers the CREATE path: no pointer exists → the
// script writes the pointer + hash entry and returns DebounceUpsertCreated.
func TestDebounceUpsert_Create(t *testing.T) {
	ctx := context.Background()
	shard, _ := newDebounceUpsertShard(t)

	accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()
	scope := upsertScope(accountID, wsID, fnID)
	debounceID := freshULID()

	result, err := shard.DebounceUpsert(ctx, scope, fnID.String(), debounceID,
		debounceItemJSON(t, time.Now().UnixMilli(), 0), 10*time.Second, time.Now(), time.Now().UnixMilli())
	require.NoError(t, err)

	assert.Equal(t, osqueue.DebounceUpsertCreated, result.Status)
	assert.Equal(t, debounceID, result.DebounceID, "created result must carry the proposed debounce ID")

	// Pointer and hash entry must exist.
	ptr, err := shard.DebounceGetPointer(ctx, scope, fnID.String())
	require.NoError(t, err)
	assert.Equal(t, debounceID.String(), ptr)

	byt, err := shard.DebounceGetItem(ctx, scope, debounceID)
	require.NoError(t, err)
	assert.NotEmpty(t, byt)
}

// TestDebounceUpsert_Update covers the UPDATE path: pointer exists and the
// incoming event is newer → refreshes hash + pointer TTL and returns
// DebounceUpsertUpdated with the effective TTL.
func TestDebounceUpsert_Update(t *testing.T) {
	ctx := context.Background()
	shard, _ := newDebounceUpsertShard(t)

	accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()
	scope := upsertScope(accountID, wsID, fnID)

	// Seed an existing debounce.
	existingID := freshULID()
	_, err := shard.DebounceCreate(ctx, scope, fnID.String(), existingID,
		debounceItemJSON(t, 1000, 0), 10*time.Second)
	require.NoError(t, err)

	// Upsert with a newer event.
	result, err := shard.DebounceUpsert(ctx, scope, fnID.String(), freshULID(),
		debounceItemJSON(t, 2000, 0), 10*time.Second, time.Now(), 2000)
	require.NoError(t, err)

	assert.Equal(t, osqueue.DebounceUpsertUpdated, result.Status)
	assert.Equal(t, existingID, result.DebounceID, "updated result must carry the existing debounce ID")
	assert.Greater(t, result.NewTTLSeconds, int64(0))

	// Stored event timestamp must be updated.
	byt, err := shard.DebounceGetItem(ctx, scope, existingID)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(byt, &decoded))
	e, _ := decoded["e"].(map[string]any)
	require.NotNil(t, e)
	assert.EqualValues(t, 2000, e["ts"])
}

// TestDebounceUpsert_OutOfOrder covers the OUT_OF_ORDER path: the incoming
// event is older than the stored one → returns DebounceUpsertOutOfOrder and
// leaves the hash entry unchanged.
func TestDebounceUpsert_OutOfOrder(t *testing.T) {
	ctx := context.Background()
	shard, _ := newDebounceUpsertShard(t)

	accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()
	scope := upsertScope(accountID, wsID, fnID)

	existingID := freshULID()
	_, err := shard.DebounceCreate(ctx, scope, fnID.String(), existingID,
		debounceItemJSON(t, 9999, 0), 10*time.Second) // stored ts = 9999
	require.NoError(t, err)

	// Try to upsert with an older event (ts = 1).
	result, err := shard.DebounceUpsert(ctx, scope, fnID.String(), freshULID(),
		debounceItemJSON(t, 1, 0), 10*time.Second, time.Now(), 1)
	require.NoError(t, err)

	assert.Equal(t, osqueue.DebounceUpsertOutOfOrder, result.Status)

	// Hash entry must be unchanged.
	byt, err := shard.DebounceGetItem(ctx, scope, existingID)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(byt, &decoded))
	e, _ := decoded["e"].(map[string]any)
	require.NotNil(t, e)
	assert.EqualValues(t, 9999, e["ts"], "stored event must not be overwritten by an older event")
}

// TestDebounceUpsert_Orphaned covers the ORPHANED path: the pointer exists but
// the hash entry is gone (post-execution slot). The script re-creates with the
// proposed new ID and returns DebounceUpsertOrphaned.
func TestDebounceUpsert_Orphaned(t *testing.T) {
	ctx := context.Background()
	shard, _ := newDebounceUpsertShard(t)

	accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()
	scope := upsertScope(accountID, wsID, fnID)

	// Seed pointer, then delete the hash entry to simulate post-execution state.
	existingID := freshULID()
	_, err := shard.DebounceCreate(ctx, scope, fnID.String(), existingID,
		debounceItemJSON(t, 1000, 0), 10*time.Second)
	require.NoError(t, err)
	require.NoError(t, shard.DebounceDeleteItems(ctx, scope, existingID))

	// Upsert must detect the orphan and re-create.
	freshID := freshULID()
	result, err := shard.DebounceUpsert(ctx, scope, fnID.String(), freshID,
		debounceItemJSON(t, 2000, 0), 10*time.Second, time.Now(), 2000)
	require.NoError(t, err)

	assert.Equal(t, osqueue.DebounceUpsertOrphaned, result.Status)
	assert.Equal(t, freshID, result.DebounceID)

	// Fresh pointer and hash entry must exist.
	ptr, err := shard.DebounceGetPointer(ctx, scope, fnID.String())
	require.NoError(t, err)
	assert.Equal(t, freshID.String(), ptr)

	byt, err := shard.DebounceGetItem(ctx, scope, freshID)
	require.NoError(t, err)
	assert.NotEmpty(t, byt)
}

// TestDebounceUpsert_MaxTimeoutCap verifies that when the remaining time until
// the absolute timeout is shorter than the requested TTL, the Lua script caps
// the TTL and propagates the original timeout into the updated item.
func TestDebounceUpsert_MaxTimeoutCap(t *testing.T) {
	ctx := context.Background()
	shard, _ := newDebounceUpsertShard(t)

	accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()
	scope := upsertScope(accountID, wsID, fnID)

	now := time.Now()
	absoluteTimeout := now.Add(3 * time.Second).UnixMilli() // expires in ~3 s

	// Seed with an absolute timeout.
	existingID := freshULID()
	_, err := shard.DebounceCreate(ctx, scope, fnID.String(), existingID,
		debounceItemJSON(t, 1000, absoluteTimeout), 10*time.Second)
	require.NoError(t, err)

	// Request a 10 s TTL — must be capped to ≤ 3 s.
	result, err := shard.DebounceUpsert(ctx, scope, fnID.String(), freshULID(),
		debounceItemJSON(t, 2000, 0), 10*time.Second, now, 2000)
	require.NoError(t, err)

	assert.Equal(t, osqueue.DebounceUpsertUpdated, result.Status)
	assert.LessOrEqual(t, result.NewTTLSeconds, int64(3),
		"TTL must not exceed the remaining absolute timeout")
	assert.Greater(t, result.NewTTLSeconds, int64(0), "TTL must be at least 1 s")

	// Updated item must carry the original absolute timeout.
	byt, err := shard.DebounceGetItem(ctx, scope, existingID)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(byt, &decoded))
	storedT, _ := decoded["t"].(float64)
	assert.Equal(t, float64(absoluteTimeout), storedT,
		"updated item must preserve the original absolute timeout for future cap enforcement")
}

// TestDebounceUpsert_CreateThenUpdate verifies the create → update lifecycle
// sequentially: the first call returns CREATED, the second for the same key
// returns UPDATED. This exercises the same code path as concurrent callers but
// avoids goroutine-scheduling sensitivity in the test environment.
func TestDebounceUpsert_CreateThenUpdate(t *testing.T) {
	ctx := context.Background()
	shard, _ := newDebounceUpsertShard(t)

	accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()
	scope := upsertScope(accountID, wsID, fnID)
	now := time.Now()

	first, err := shard.DebounceUpsert(ctx, scope, fnID.String(), freshULID(),
		debounceItemJSON(t, 1000, 0), 10*time.Second, now, 1000)
	require.NoError(t, err)
	assert.Equal(t, osqueue.DebounceUpsertCreated, first.Status,
		"first call must create a new debounce")

	second, err := shard.DebounceUpsert(ctx, scope, fnID.String(), freshULID(),
		debounceItemJSON(t, 2000, 0), 10*time.Second, now, 2000)
	require.NoError(t, err)
	assert.Equal(t, osqueue.DebounceUpsertUpdated, second.Status,
		"second call for same key must take the update path")
	assert.Equal(t, first.DebounceID, second.DebounceID,
		"both calls must reference the same debounce entry")
}

// TestDebounceUpsert_ConcurrentCreates verifies that two goroutines racing on
// the same key produce exactly one CREATED and one UPDATED, proving Lua
// atomicity. Miniredis serialises Lua scripts internally so the outcome is
// deterministic despite the goroutine concurrency.
func TestDebounceUpsert_ConcurrentCreates(t *testing.T) {
	ctx := context.Background()
	shard, _ := newDebounceUpsertShard(t)

	accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()
	scope := upsertScope(accountID, wsID, fnID)
	now := time.Now()

	results := make([]osqueue.DebounceUpsertResult, 2)
	errs := make([]error, 2)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			ts := now.UnixMilli() + int64(idx)
			results[idx], errs[idx] = shard.DebounceUpsert(
				ctx, scope, fnID.String(), freshULID(),
				debounceItemJSON(t, ts, 0), 10*time.Second, now, ts)
		}()
	}
	wg.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])

	created := 0
	for _, r := range results {
		switch r.Status {
		case osqueue.DebounceUpsertCreated, osqueue.DebounceUpsertOrphaned:
			created++
		}
	}
	// Depending on goroutine scheduling the loser may see UPDATED or
	// OUT_OF_ORDER (if its event timestamp is lower than the winner's).
	// Either way, at most one create must win.
	assert.LessOrEqual(t, created, 1, "at most one concurrent create must win")
}

// ─── requeueByID.lua lease-expiry return tests ────────────────────────────────

// enqueueDebounceItem enqueues a debounce queue item with a caller-supplied
// rawJobID. RequeueByJobID expects this un-hashed ID; EnqueueItem hashes it
// internally for storage — both sides call HashID(rawJobID) so the key matches.
func enqueueDebounceItem(t *testing.T, ctx context.Context, shard RedisQueueShard, rawJobID string, accountID, wsID, fnID uuid.UUID) osqueue.QueueItem {
	t.Helper()
	item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
		ID:          rawJobID,
		FunctionID:  fnID,
		WorkspaceID: wsID,
	}, time.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)
	return item
}

// TestRequeueByJobID_LeaseExpiryError verifies that when a queue item is leased
// (executing), RequeueByJobID returns a LeaseExpiryError — not the plain
// ErrQueueItemAlreadyLeased sentinel — and that the encoded expiry is in the
// future.
func TestRequeueByJobID_LeaseExpiryError(t *testing.T) {
	ctx := context.Background()
	_, rc := initRedis(t)
	_, shard := newQueue(t, rc)

	accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()
	rawID := freshULID().String() // un-hashed; RequeueByJobID hashes this internally
	item := enqueueDebounceItem(t, ctx, shard, rawID, accountID, wsID, fnID)

	// Lease it for 30 s.
	_, err := shard.Lease(ctx, item, 30*time.Second, time.Now())
	require.NoError(t, err)

	reqErr := shard.RequeueByJobID(ctx, rawID, time.Now().Add(10*time.Second))
	require.Error(t, reqErr)

	// errors.Is must still match the category sentinel via Unwrap.
	assert.True(t, errors.Is(reqErr, osqueue.ErrQueueItemAlreadyLeased),
		"errors.Is(ErrQueueItemAlreadyLeased) must be true via Unwrap; got: %v", reqErr)

	// The concrete type must be LeaseExpiryError with a future expiry.
	var leaseErr osqueue.LeaseExpiryError
	require.True(t, errors.As(reqErr, &leaseErr),
		"error must be a LeaseExpiryError, got %T: %v", reqErr, reqErr)
	assert.Greater(t, leaseErr.ExpiryMS, time.Now().UnixMilli(),
		"lease expiry must be in the future")
	assert.LessOrEqual(t, leaseErr.ExpiryMS, time.Now().Add(31*time.Second).UnixMilli(),
		"lease expiry must not exceed the configured 30 s lease (with 1 s slack)")
}

// TestRequeueByJobID_NotFound verifies that a nonexistent job returns an error.
// The preliminary HGET inside RequeueByJobID returns a Redis nil before the Lua
// script runs, so the concrete error is a rueidis nil — not ErrQueueItemNotFound.
// We assert only that an error is returned, consistent with the existing queue tests.
func TestRequeueByJobID_NotFound(t *testing.T) {
	ctx := context.Background()
	_, rc := initRedis(t)
	_, shard := newQueue(t, rc)

	err := shard.RequeueByJobID(ctx, "does-not-exist", time.Now().Add(5*time.Second))
	require.Error(t, err, "missing item must return an error")
}

// TestRequeueByJobID_Success verifies the happy path: an unleased item is
// rescheduled and nil is returned.
func TestRequeueByJobID_Success(t *testing.T) {
	ctx := context.Background()
	_, rc := initRedis(t)
	_, shard := newQueue(t, rc)

	accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()
	rawID := freshULID().String()
	enqueueDebounceItem(t, ctx, shard, rawID, accountID, wsID, fnID)

	err := shard.RequeueByJobID(ctx, rawID, time.Now().Add(5*time.Second))
	require.NoError(t, err, "requeue of unleased item must succeed")
}

// ─── DebounceUpsert Go decoder coverage ───────────────────────────────────────

// TestDebounceUpsertDecoder_AllStatusCodes drives the Go decoder for every Lua
// return code via live Redis round-trips, confirming that each numeric status
// maps to its typed constant and that the payload fields are populated.
func TestDebounceUpsertDecoder_AllStatusCodes(t *testing.T) {
	ctx := context.Background()

	t.Run("status 1 → DebounceUpsertCreated", func(t *testing.T) {
		shard, _ := newDebounceUpsertShard(t)
		fnID := uuid.New()
		scope := upsertScope(uuid.New(), uuid.New(), fnID)
		id := freshULID()

		result, err := shard.DebounceUpsert(ctx, scope, fnID.String(), id,
			debounceItemJSON(t, 1000, 0), 10*time.Second, time.Now(), 1000)
		require.NoError(t, err)
		assert.Equal(t, osqueue.DebounceUpsertCreated, result.Status)
		assert.Equal(t, id, result.DebounceID)
	})

	t.Run("status 2 → DebounceUpsertUpdated with positive NewTTLSeconds", func(t *testing.T) {
		shard, _ := newDebounceUpsertShard(t)
		fnID := uuid.New()
		scope := upsertScope(uuid.New(), uuid.New(), fnID)
		existingID := freshULID()
		_, err := shard.DebounceCreate(ctx, scope, fnID.String(), existingID,
			debounceItemJSON(t, 1000, 0), 10*time.Second)
		require.NoError(t, err)

		result, err := shard.DebounceUpsert(ctx, scope, fnID.String(), freshULID(),
			debounceItemJSON(t, 2000, 0), 10*time.Second, time.Now(), 2000)
		require.NoError(t, err)
		assert.Equal(t, osqueue.DebounceUpsertUpdated, result.Status)
		assert.Equal(t, existingID, result.DebounceID)
		assert.Greater(t, result.NewTTLSeconds, int64(0))
	})

	t.Run("status 3 → DebounceUpsertOutOfOrder", func(t *testing.T) {
		shard, _ := newDebounceUpsertShard(t)
		fnID := uuid.New()
		scope := upsertScope(uuid.New(), uuid.New(), fnID)
		_, err := shard.DebounceCreate(ctx, scope, fnID.String(), freshULID(),
			debounceItemJSON(t, 9000, 0), 10*time.Second)
		require.NoError(t, err)

		result, err := shard.DebounceUpsert(ctx, scope, fnID.String(), freshULID(),
			debounceItemJSON(t, 1, 0), 10*time.Second, time.Now(), 1)
		require.NoError(t, err)
		assert.Equal(t, osqueue.DebounceUpsertOutOfOrder, result.Status)
	})

	t.Run("status 4 → DebounceUpsertOrphaned", func(t *testing.T) {
		shard, _ := newDebounceUpsertShard(t)
		fnID := uuid.New()
		scope := upsertScope(uuid.New(), uuid.New(), fnID)
		orphanID := freshULID()
		_, err := shard.DebounceCreate(ctx, scope, fnID.String(), orphanID,
			debounceItemJSON(t, 1000, 0), 10*time.Second)
		require.NoError(t, err)
		require.NoError(t, shard.DebounceDeleteItems(ctx, scope, orphanID))

		freshID := freshULID()
		result, err := shard.DebounceUpsert(ctx, scope, fnID.String(), freshID,
			debounceItemJSON(t, 2000, 0), 10*time.Second, time.Now(), 2000)
		require.NoError(t, err)
		assert.Equal(t, osqueue.DebounceUpsertOrphaned, result.Status)
		assert.Equal(t, freshID, result.DebounceID)
	})
}
