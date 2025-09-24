# Lua Number to String Conversion Issues in Redis Scripts

## Background

Lua 5.3 changed the behavior of automatic number-to-string conversions compared to Lua 5.2. When passing numeric values directly to Redis commands without explicit string formatting, this can cause compatibility issues. For example:
- In Lua 5.2: `tostring(3.0)` returns `"3"`
- In Lua 5.3: `tostring(3.0)` returns `"3.0"`

This can cause Redis commands to fail when they expect integer strings but receive float strings.

## Affected Files

### 1. connect/state/lua/

#### lease.lua
- **Line 30**: `redis.call("SET", keyRequestLease, cjson.encode(requestItem), "EX", expiry)`
- **Issue**: `expiry` is from `tonumber(ARGV[2])` and passed directly without string formatting

#### extend_lease.lua
- **Line 39**: `redis.call("SET", keyRequestLease, cjson.encode(requestItem), "EX", expiry)`
- **Issue**: `expiry` is from `tonumber(ARGV[3])` and passed directly without string formatting

### 2. execution/debounce/lua/

#### newDebounce.lua
- **Line 26**: `redis.call("SETEX", keyPtr, ttl, debounceID)`
- **Issue**: `ttl` is from `tonumber(ARGV[3])` and passed directly without string formatting

#### updateDebounce.lua
- **Line 60**: `redis.call("SETEX", keyPtr, ttl, debounceID)`
- **Line 108**: `redis.call("SETEX", keyPtr, ttl, debounceID)`
- **Issue**: `ttl` is from `tonumber(ARGV[3])` and passed directly without string formatting

### 3. execution/state/redis_state/lua/

#### includes/gcra.lua
- **Line 52**: `redis.call("SET", key, new_tat, "EX", expiry)`
- **Issue**: `expiry` is calculated as `(period_ms / 1000)` and passed directly without string formatting
- **Note**: This is inconsistent with line 30 in the same file which correctly uses `string.format("%d", period_ms / 1000)`

#### includes/check_concurrency.lua
- **Line 14**: `redis.call("ZADD", keyZset, score, partitionID)`
- **Issue**: `score` parameter is passed directly without string conversion

#### queue/peek.lua
- **Line 28**: `redis.call("ZRANGE", queueIndex, peekFrom, peekUntil, "BYSCORE", "LIMIT", offset, limit)`
- **Issue**: `offset` and `limit` are numbers passed directly without string formatting
- **Note**: `limit` is from `tonumber(ARGV[3])`, `offset` is calculated

#### queue/peekPointerUntil.lua
- **Line 21**: `redis.call("ZRANGE", keyOrderedPointerSet, "-inf", peekUntil, "BYSCORE", "LIMIT", offset, limit)`
- **Issue**: `offset` and `limit` are numbers passed directly without string formatting
- **Note**: `limit` is from `tonumber(ARGV[3])`, `offset` is calculated

#### queue/accountPeek.lua
- **Line 22**: `redis.call("ZRANGE", keyGlobalAccountPointer, "-inf", peekUntil, "BYSCORE", "LIMIT", offset, limit)`
- **Issue**: `offset` and `limit` are numbers passed directly without string formatting
- **Note**: `limit` is from `tonumber(ARGV[2])`, `offset` is calculated

#### queue/partitionPeek.lua
- **Line 25**: `redis.call("ZRANGE", partitionIndex, "-inf", peekUntil, "BYSCORE", "LIMIT", offset, limit)`
- **Issue**: `offset` and `limit` are numbers passed directly without string formatting

#### queue/peekOrderedSetUntil.lua
- **Line 28**: `redis.call("ZRANGE", keyOrderedPointerSet, peekFrom, peekUntil, "BYSCORE", "LIMIT", offset, limit)`
- **Issue**: `offset` and `limit` are numbers passed directly without string formatting

#### queue/peekOrderedSet.lua
- **Line 14**: `redis.call("ZRANGE", keyPointerSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, limit)`
- **Issue**: `limit` is a number passed directly without string formatting

## Recommended Fixes

### For SET/SETEX commands with TTL/expiry:
```lua
-- Instead of:
redis.call("SET", key, value, "EX", expiry)
-- Use:
redis.call("SET", key, value, "EX", string.format("%d", expiry))
```

### For ZRANGE commands with LIMIT:
```lua
-- Instead of:
redis.call("ZRANGE", key, min, max, "BYSCORE", "LIMIT", offset, limit)
-- Use:
redis.call("ZRANGE", key, min, max, "BYSCORE", "LIMIT", string.format("%d", offset), string.format("%d", limit))
```

### For ZADD commands with scores:
```lua
-- Instead of:
redis.call("ZADD", key, score, member)
-- Use:
redis.call("ZADD", key, tostring(score), member)
```

## Scripts Without Issues

The following directories were checked and found to have no implicit numeric conversions:
- `pkg/execution/batch/lua/` - All numeric values are properly handled

## Floating Point Operations in Lua Scripts

### Scripts Using Division (/) That Can Produce Floats

#### 1. execution/debounce/lua/updateDebounce.lua
- **Line 88**: `ttl = math.floor((item.t - currentTime) / 1000)`
- **Context**: Division by 1000 to convert milliseconds to seconds, wrapped in `math.floor` to ensure integer result

#### 2. execution/state/redis_state/lua/includes/gcra.lua
- **Line 8**: `local emission = period_ms / math.max(limit, 1)`
- **Line 29**: `local expiry = string.format("%d", period_ms / 1000)` (correctly formatted)
- **Line 37**: `local emission = period_ms / math.max(limit, 1)`
- **Line 51**: `local expiry = (period_ms / 1000)` (NOT formatted - potential issue!)
- **Line 58**: `local emission = period_ms / math.max(limit, 1)`
- **Line 75, 78**: `local capacity = math.floor(time_capacity_remain / emission)`
- **Context**: GCRA rate limiting calculations involve division to calculate emission intervals

#### 3. execution/state/redis_state/lua/includes/update_pointer_score.lua
- **Line 21**: `return math.floor(tonumber(earliestScore[2]) / 1000)`
- **Context**: Converting milliseconds to seconds, wrapped in `math.floor`

#### 4. execution/state/redis_state/lua/includes/enqueue_to_partition.lua
- **Line 167**: `local updateTo = earliestScore/1000`
- **Context**: Division by 1000 without `math.floor` - potential float!

#### 5. execution/state/redis_state/lua/queue/dequeue.lua
- **Line 120**: `local earliestScore = tonumber(minScores[2])/1000`
- **Context**: Division by 1000 without `math.floor` - potential float!

#### 6. execution/state/redis_state/lua/queue/partitionRequeue.lua
- **Line 31**: `local atS = math.floor(atMS / 1000)`
- **Line 91-92**: `math.floor(item.at / 1000)`
- **Context**: Converting milliseconds to seconds, properly wrapped in `math.floor`

#### 7. execution/state/redis_state/lua/queue/accountPeek.lua
- **Line 11**: `local peekUntil = math.ceil(peekUntilMS / 1000)`
- **Context**: Converting milliseconds to seconds, wrapped in `math.ceil`

#### 8. execution/state/redis_state/lua/queue/partitionPeek.lua
- **Line 14**: `local peekUntil = math.ceil(peekUntilMS / 1000)`
- **Context**: Converting milliseconds to seconds, wrapped in `math.ceil`

### Scripts Using math.pow (Can Produce Large Numbers)

#### 1. connect/state/lua/includes/decode_ulid_time.lua
- **Line 15**: `time = time + (ulidMap[string.sub(rev, i, i)] * math.pow(32, i-1))`
- **Context**: ULID decoding using powers of 32

#### 2. execution/debounce/lua/updateDebounce.lua
- **Line 40**: `time = time + (ulidMap[string.sub(rev, i, i)] * math.pow(32, i-1))`
- **Context**: ULID decoding using powers of 32

#### 3. execution/state/redis_state/lua/includes/decode_ulid_time.lua
- **Line 15**: `time = time + (ulidMap[string.sub(rev, i, i)] * math.pow(32, i-1))`
- **Context**: ULID decoding using powers of 32

### Scripts Using math.random (Returns Floats Between 0 and 1)

Multiple peek-related scripts use `math.random` to calculate offsets:
- `queue/peek.lua`
- `queue/peekPointerUntil.lua`
- `queue/accountPeek.lua`
- `queue/partitionPeek.lua`
- `queue/peekOrderedSetUntil.lua`

All properly handle the result by subtracting 1 to ensure integer offsets.

### Critical Findings

1. **Unguarded Division Operations**: Several scripts perform division by 1000 (converting ms to seconds) without using `math.floor`:
   - `includes/enqueue_to_partition.lua:167`
   - `queue/dequeue.lua:120`
   - These could produce float values that are then used in Redis operations

2. **Inconsistent Handling**: The same file (`gcra.lua`) handles division differently in different places:
   - Line 29: Correctly uses `string.format("%d", period_ms / 1000)`
   - Line 51: Just uses `(period_ms / 1000)` without formatting

3. **No Decimal Literals**: No scripts use explicit decimal literals (e.g., `0.5`, `3.14`)

4. **Proper Float Handling**: Most scripts that perform division operations properly use:
   - `math.floor()` to ensure integer results
   - `math.ceil()` for rounding up
   - `string.format("%d", ...)` for explicit integer formatting