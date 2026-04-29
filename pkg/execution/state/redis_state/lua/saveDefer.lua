--[[
Atomically save a Defer record, refusing to resurrect a cancelled one.

Without atomicity a retried DeferAdd can undo an interleaved DeferCancel:
T1 SaveDefer → T2 SetDeferStatus(Cancelled) → T3 retry of T1 would silently
overwrite. Reading and writing inside one Lua invocation closes the race.

KEYS[1] - defers hash key
ARGV[1] - hashedID (hash field)
ARGV[2] - new Defer JSON (HSET verbatim — never decoded/re-encoded)
ARGV[3] - integer ScheduleStatusCancelled

Output:
  1: written
  0: no-op (existing record is already cancelled)
]]

local defersKey      = KEYS[1]
local field          = ARGV[1]
local newPayload     = ARGV[2]
local cancelledValue = tonumber(ARGV[3])

local existing = redis.call("HGET", defersKey, field)
if existing then
    local prev = cjson.decode(existing)
    if prev.ScheduleStatus == cancelledValue then
        return 0
    end
end

redis.call("HSET", defersKey, field, newPayload)
return 1
