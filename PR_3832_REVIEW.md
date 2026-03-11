# PR #3832 Review: Centralized Redis Caching for Constraint Acquisition

## Summary

This PR adds a Redis-backed cache layer that short-circuits constraint evaluation when a constraint is already known to be exhausted. On cache hit, the Lua script returns immediately without evaluating any constraints, reducing latency and Redis command volume. The feature is opt-in via `EnableAcquireCacheFn` with per-constraint granularity and configurable TTL bounds.

---

## CRITICAL Issues

### 1. No cache invalidation on Release — stale denials

**File:** `pkg/constraintapi/lua/acquire.lua` (cache write at lines ~252-258, ~362-368)
**File:** Release path (no changes)

When a lease is released (or expires/scavenged), the cache entry remains until its TTL expires. During this window, **every acquire request is incorrectly denied** even though capacity is available. For a system requiring "strong consistency and full correctness in constraint enforcement," this is a correctness violation.

**Impact:** At hundreds of millions of requests/hour, even a 1-second stale window means millions of requests are incorrectly rejected after capacity becomes available.

**Fix:** Either:
- (a) Delete the cache key in the Release Lua script when releasing a lease frees capacity, or
- (b) Reduce the cache to an advisory hint — on cache hit, still check actual capacity for a percentage of requests (probabilistic validation), or
- (c) Document clearly that this cache intentionally trades correctness for throughput and must only be used for constraints where brief over-rejection is acceptable.

### 2. Last-writer-wins TTL bug — per-constraint TTL config is silently discarded

**File:** `pkg/constraintapi/acquire.go`, lines ~235-249

```go
for i, ci := range sortedConstraints {
    enabled, minTTL, maxTTL := r.enableAcquireCache(ctx, req.AccountID, req.EnvID, req.FunctionID, ci)
    if !enabled {
        continue
    }
    // ...
    cacheMinTTL = int(max(minTTL.Seconds(), 1))  // OVERWRITES previous value
    cacheMaxTTL = int(max(maxTTL.Seconds(), 1))  // OVERWRITES previous value
}
```

Only a **single** `cacheMinTTL`/`cacheMaxTTL` pair is sent to Lua (ARGV[12..13]) and applied to **all** constraints. If the `EnableAcquireCacheFn` returns different TTL bounds per constraint (e.g., 5s min for concurrency, 30s min for throttle), only the last enabled constraint's values survive. Earlier constraints get the wrong TTL.

**Fix:** Either:
- (a) Send per-constraint TTL values as additional ARGV entries (e.g., ARGV[14+2i], ARGV[14+2i+1] for min/max), or
- (b) Use `max()` aggregation across all constraints: `cacheMinTTL = max(cacheMinTTL, newMin)`, or
- (c) Document the single-TTL-pair design decision and enforce it in the `EnableAcquireCacheFn` contract (assert all enabled constraints return the same TTL).

### 3. Cache stampede / thundering herd on TTL expiry

**File:** `pkg/constraintapi/lua/acquire.lua`

When a cache entry expires, **all instances simultaneously** fall through to full constraint evaluation. For a high-QPS constraint (e.g., account-level concurrency), this creates a sudden load spike on the constraint's sorted set.

**Fix:** Add TTL jitter. In the Lua cache-write path:
```lua
local jitter = math.random(0, math.floor(cacheTTLSec * 0.1))  -- 10% jitter
cacheTTLSec = cacheTTLSec + jitter
```
Or implement probabilistic early expiration (XFetch algorithm) on the read path.

---

## HIGH Severity Issues

### 4. Cache keys accessed via ARGV violate Redis scripting contract

**File:** `pkg/constraintapi/lua/acquire.lua`, line ~173

```lua
local cacheValues = call("MGET", unpack(mgetKeys))
```

Cache keys are passed as ARGV and used to issue `MGET` and `SET` commands. Per Redis documentation, all keys a script accesses **must** be declared in the KEYS array. While this works because all keys use the `{cs}` hash tag (same slot), it violates the scripting contract and:
- May break on future Redis/Valkey/Garnet versions that enforce KEYS-only access
- Breaks `redis-cli --eval` key auditing
- The existing codebase already has this pattern (e.g., `scopedKeyPrefix` in ARGV), so this is pre-existing tech debt, but this PR deepens it

**Recommendation:** Track as tech debt; consider moving dynamic keys to KEYS in a follow-up.

### 5. Missing cache eviction path means cache is write-only

The cache is written on exhaustion but **never deleted** except by TTL expiry. There is no:
- Invalidation on Release
- Invalidation on Extend (which refreshes a lease)
- Invalidation when configuration changes (e.g., concurrency limit increased)

This makes the cache a one-way valve: once a constraint is cached as exhausted, it stays that way for `TTL` seconds regardless of actual state changes.

### 6. Negative or zero `constraintRetryAt` produces nonsensical cache entries

**File:** `pkg/constraintapi/lua/acquire.lua`, lines ~252-258

```lua
local cacheTTLSec = math.max(
    math.min(math.ceil((constraintRetryAt - nowMS) / 1000), cacheMaxTTL),
    cacheMinTTL
)
```

If `constraintRetryAt` is 0 (e.g., concurrency exhaustion with no known retry time), then `(0 - nowMS) / 1000` is a huge negative number. `math.min` with `cacheMaxTTL` keeps it negative, then `math.max` with `cacheMinTTL` clamps to `cacheMinTTL`. The cache entry stores `"0"` as the retryAt value.

When read back, `cachedRetryAt = tonumber("0") = 0`, and the condition `cachedRetryAt > retryAt` (where retryAt starts at 0) is **false**, so the cached retryAt is silently ignored. The caller gets `retryAt = 0` (epoch), which is before `now`, so `retryAfter` becomes `time.Time{}` (zero value). This is technically "correct" but confusing — we're caching a constraint as exhausted with no meaningful retry-after hint.

**Fix:** Skip cache write when `constraintRetryAt <= nowMS`:
```lua
if ck ~= nil and ck ~= "" and constraintRetryAt > nowMS then
```

---

## MEDIUM Severity Issues

### 7. `unpack()` stack limit with many constraints

**File:** `pkg/constraintapi/lua/acquire.lua`, line ~173

Lua 5.1's `unpack()` is limited by `LUAI_MAXCSTACK` (default ~8000). While constraint counts are typically small, if the system ever supports user-defined custom constraints at scale, `unpack(mgetKeys)` could hit this limit.

**Recommendation:** Low risk for now, but consider batching MGET calls for robustness.

### 8. Cache hit short-circuits idempotency key side effects

**File:** `pkg/constraintapi/lua/acquire.lua`, lines ~201-210

On cache hit, the script returns immediately **without**:
- Setting the operation idempotency key
- Setting the constraint check idempotency key
- Updating the scavenger shard

This means retrying the same request after a cache hit won't return an idempotent response — it will hit the cache again (which is fine functionally, but inconsistent with the non-cached path). If the cache expires between retries, the request could succeed on retry when it should have been idempotent. This is a subtle behavioral difference.

### 9. Duplicated cache-write logic (DRY violation)

**File:** `pkg/constraintapi/lua/acquire.lua`

The cache TTL calculation and SET logic is duplicated in three places:
- Pre-grant exhaustion check (~lines 252-258)
- Post-grant exhaustion check (~lines 362-368)

Extract to a local function:
```lua
local function writeCacheEntry(ck, constraintRetryAt)
    if ck == nil or ck == "" then return end
    local cacheTTLSec = math.max(
        math.min(math.ceil((constraintRetryAt - nowMS) / 1000), cacheMaxTTL),
        cacheMinTTL
    )
    if cacheTTLSec > 0 then
        call("SET", ck, tostring(constraintRetryAt), "EX", tostring(cacheTTLSec))
    end
end
```

---

## LOW Severity / Nits

### 10. Test helper `r.Set()` calls don't check errors

**File:** `pkg/constraintapi/acquire_test.go`, lines ~106, 205, 270, 469, 508

The miniredis `Set()` method doesn't return an error in v2, so these are actually fine. However, `SetTTL()` also doesn't error. If linting tools flag these, wrap in a helper or add a comment explaining miniredis's API.

### 11. Missing test coverage

- No test for constraints with `KeyExpressionHash`/`EvaluatedKeyHash` (custom key expressions)
- No test for concurrent goroutine access to the same cache key
- No test for the `constraintRetryAt == 0` edge case described in issue #6
- No test that Release clears cache (because it doesn't — see issue #1)
- No negative test: what happens when Redis is down? Does MGET failure bubble up as script error?

### 12. Snapshot file missing trailing newline

**File:** `pkg/constraintapi/testdata/snapshots/acquire.lua` — no newline at EOF.

---

## Architecture Questions

1. **Consistency contract:** Is it acceptable for this cache to reject valid requests for up to `MaxCacheTTL` (60 seconds) after capacity becomes available? If so, this should be documented at the `EnableAcquireCacheFn` level.

2. **Interaction with in-memory cache:** The PR description mentions "the same CacheKey logic as the in-memory cache." How do the two caches interact? Is there a risk of double-caching (in-memory + Redis) with compounding staleness?

3. **Rollout strategy:** The `EnableAcquireCacheFn` callback allows gradual rollout per account/env/function. Is there a plan for monitoring the false-rejection rate during rollout?

---

## What's Good

- Clean opt-in design via `EnableAcquireCacheFn` with per-constraint granularity
- TTL clamping prevents both overly aggressive and overly stale cache entries
- Metrics (`constraintapi_acquire_cache_total`) with hit/miss + shard tags
- Comprehensive test suite covering core scenarios, isolation, feature flags
- Lua compatibility test against real Redis-compatible backends (Valkey, Garnet)
- Cache keys properly scoped with account/env/function isolation via `{cs}` hash tag
- Short-circuit returns correct response structure (status=2, exhausted/limiting constraints)
