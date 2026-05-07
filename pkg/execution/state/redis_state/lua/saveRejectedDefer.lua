--[[
Idempotently write a Rejected meta sentinel.

No-op if any entry already exists for the hashedID (so a previously-accepted
AfterRun defer is never silently downgraded by a stale rejection signal).

KEYS[1] - defers meta hash key
ARGV[1] - hashedID
ARGV[2] - meta JSON ({FnSlug, HashedID, ScheduleStatus = Rejected})
ARGV[3] - integer max defers per run

Output:
   1: sentinel written
   0: no-op (entry already exists)
  -1: no-op (per-run count cap exceeded)
]]

local metaKey     = KEYS[1]
local hashedID    = ARGV[1]
local metaPayload = ARGV[2]
local maxDefers   = tonumber(ARGV[3])

if redis.call("HEXISTS", metaKey, hashedID) == 1 then
    return 0
end

local total = redis.call("HLEN", metaKey)
if total >= maxDefers then
    return -1
end

redis.call("HSET", metaKey, hashedID, metaPayload)
return 1
