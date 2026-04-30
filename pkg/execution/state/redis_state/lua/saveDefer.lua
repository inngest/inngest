--[[
Atomically save a Defer record, refusing to resurrect a cancelled one.

Stores meta and Input as two separate hash fields so SetDeferStatus never
round-trips Input through cjson (which corrupts nested empty objects and
loses precision on integers above 2^53).

Without atomicity a retried DeferAdd can undo an interleaved DeferCancel:
T1 SaveDefer → T2 SetDeferStatus(Cancelled) → T3 retry of T1 would silently
overwrite. Reading and writing inside one Lua invocation closes the race.

KEYS[1] - defers hash key
ARGV[1] - hashedID
ARGV[2] - meta JSON ({FnSlug, HashedID, ScheduleStatus} only)
ARGV[3] - raw Input bytes (HSET verbatim — never decoded by Lua)
ARGV[4] - integer ScheduleStatusCancelled

Output:
  1: written
  0: no-op (existing record is already cancelled)
]]

local defersKey      = KEYS[1]
local hashedID       = ARGV[1]
local metaPayload    = ARGV[2]
local inputPayload   = ARGV[3]
local cancelledValue = tonumber(ARGV[4])

local metaField  = "meta:" .. hashedID
local inputField = "input:" .. hashedID

local existing = redis.call("HGET", defersKey, metaField)
if existing then
    local prev = cjson.decode(existing)
    if prev.ScheduleStatus == cancelledValue then
        return 0
    end
end

redis.call("HSET", defersKey, metaField, metaPayload, inputField, inputPayload)
return 1
