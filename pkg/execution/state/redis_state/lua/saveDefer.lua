--[[
Atomically insert a Defer record

Insert-only: any existing entry for the hashedID (AfterRun, Aborted, Rejected)
is a no-op, so SDK retransmits are idempotent regardless of payload.

New defers past the per-run count cap are rejected. Writes that would exceed the
aggregate-input cap are converted into a Rejected sentinel (status=Rejected, no
input, no aggregate increment).

KEYS[1] - defers meta hash key
KEYS[2] - defers input hash key
KEYS[3] - run metadata hash key
ARGV[1] - hashedID
ARGV[2] - meta JSON ({FnSlug, HashedID, UserlandID, ScheduleStatus} only)
ARGV[3] - raw Input bytes (HSET verbatim, never decoded by Lua)
ARGV[4] - integer max defers per run
ARGV[5] - integer max defer input aggregate size (bytes)
ARGV[6] - integer ScheduleStatusRejected

Output:
   1: written
   0: no-op (entry already exists)
  -1: no-op (per-run count cap exceeded)
  -2: rejected sentinel written (aggregate cap exceeded)
]]

local metaKey       = KEYS[1]
local inputKey      = KEYS[2]
local metadataKey   = KEYS[3]
local hashedID      = ARGV[1]
local metaPayload   = ARGV[2]
local inputPayload  = ARGV[3]
local maxDefers     = tonumber(ARGV[4])
local maxAggInput   = tonumber(ARGV[5])
local rejectedValue = tonumber(ARGV[6])

if redis.call("HEXISTS", metaKey, hashedID) == 1 then
    return 0
end

local total = redis.call("HLEN", metaKey)
if total >= maxDefers then
    return -1
end

local newInputLen = #inputPayload
if newInputLen > 0 then
    local current = tonumber(redis.call("HGET", metadataKey, "defer_input_size")) or 0
    if current + newInputLen > maxAggInput then
        -- Write a Rejected sentinel (no input, no aggregate increment).
        local meta = cjson.decode(metaPayload)
        meta.ScheduleStatus = rejectedValue
        redis.call("HSET", metaKey, hashedID, cjson.encode(meta))
        return -2
    end
end

redis.call("HSET", metaKey, hashedID, metaPayload)
redis.call("HSET", inputKey, hashedID, inputPayload)
if newInputLen > 0 then
    redis.call("HINCRBY", metadataKey, "defer_input_size", newInputLen)
end
return 1
