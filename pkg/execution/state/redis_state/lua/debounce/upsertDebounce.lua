--[[

Atomically creates a new debounce OR updates an existing one.

Prior to this script, the Go layer performed two separate Lua calls
(newDebounce → updateDebounce) with a Go-level dispatch in between.
The gap between those calls created a race window: a concurrent worker
could read a "new" pointer, then call updateDebounce before the
originating worker had enqueued the timeout job, returning
DebounceUpdateNotFound and triggering needless retries.

This script collapses the state decision into a single atomic execution.
Queue-item scheduling (Enqueue / RequeueByJobID) is intentionally left
to the Go caller: the enqueue pipeline is too complex to inline here, and
the atomicity of the state layer alone eliminates the observed race.

KEYS
  [1] keyPtr  -- string: fn-scoped debounce pointer  (GET / SETEX)
  [2] keyDbc  -- hash:   debounce item store          (HGET / HSET)

ARGV
  [1] debounceID    -- proposed new debounce ULID (used only when creating)
  [2] debounce      -- JSON-encoded DebounceItem
  [3] ttl           -- debounce period in seconds
  [4] currentTimeMS -- now in milliseconds (for timeout cap arithmetic)
  [5] eventTimeMS   -- event.ts in milliseconds (for out-of-order guard)

Return values (always a table so callers can inspect status + payload)
  { 1, debounceID }             CREATED       new debounce; caller Enqueue
  { 2, effectiveTTL, debounceID } UPDATED     refreshed; caller RequeueByJobID
  { 3 }                         OUT_OF_ORDER  newer event present; drop
  { 4, debounceID }             ORPHANED      pointer existed but hash entry
                                              was gone (post-execution slot);
                                              re-created as CREATED

]]--

local keyPtr      = KEYS[1]
local keyDbc      = KEYS[2]

local debounceID   = ARGV[1]
local debounce     = ARGV[2]
local ttl          = tonumber(ARGV[3])
local currentTime  = tonumber(ARGV[4])
local eventTime    = tonumber(ARGV[5])

-- ── CREATE path ──────────────────────────────────────────────────────────────
local existing = redis.call("GET", keyPtr)
if existing == nil or existing == false then
    redis.call("SETEX", keyPtr, ttl, debounceID)
    redis.call("HSET",  keyDbc, debounceID, debounce)
    return { 1, debounceID }
end

-- ── Orphan check ─────────────────────────────────────────────────────────────
-- The pointer exists but the hash entry is gone.  This happens after execution:
-- StartExecution rotates the pointer to a fresh ID, then DeleteDebounceItem
-- removes the hash entry.  Treat this as a fresh create so the next event
-- starts a new debounce cycle without waiting.
local existingStr = redis.call("HGET", keyDbc, existing)
if existingStr == false then
    redis.call("SETEX", keyPtr, ttl, debounceID)
    redis.call("HSET",  keyDbc, debounceID, debounce)
    return { 4, debounceID }
end

-- ── UPDATE path ───────────────────────────────────────────────────────────────
local existingItem = cjson.decode(existingStr)

-- Out-of-order guard: drop events whose timestamp is older than the stored one.
if existingItem ~= nil and existingItem.e ~= nil and existingItem.e.ts > eventTime then
    return { 3 }
end

-- Max-timeout cap: if the debounce was created with an absolute timeout (t),
-- ensure the new TTL does not push the execution window past that cap.
-- This preserves the invariant from updateDebounce.lua unchanged.
if existingItem ~= nil and existingItem.t ~= nil and existingItem.t > 0 then
    local nextTTL = currentTime + (ttl * 1000)
    if nextTTL > existingItem.t then
        ttl = math.floor((existingItem.t - currentTime) / 1000)
        if ttl <= 0 then
            ttl = 1
        end
        ttl = tonumber(ttl)
    end
    -- Propagate the original timeout into the updated item so any subsequent
    -- upsert sees the correct cap.
    local next = cjson.decode(debounce)
    next.t = existingItem.t
    debounce = cjson.encode(next)
end

redis.call("SETEX", keyPtr, ttl, existing)
redis.call("HSET",  keyDbc, existing, debounce)
return { 2, ttl, existing }
