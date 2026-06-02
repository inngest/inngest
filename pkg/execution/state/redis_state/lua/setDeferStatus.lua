--[[
Atomically update a Defer's ScheduleStatus.

The Aborted transition also deletes the Input and decrements the aggregate
counter, ensuring we don't use the size budget for useless data. The meta entry
stays so saveDefer.lua's terminal-sticky check still dedupes retransmits.

KEYS[1] - defers meta hash key
KEYS[2] - defers input hash key
KEYS[3] - run metadata hash key
ARGV[1] - hashedID
ARGV[2] - new status (integer, as a string)
ARGV[3] - integer ScheduleStatusAborted

Output:
  1: status updated
  0: defer not found
]]

local metaKey      = KEYS[1]
local inputKey     = KEYS[2]
local metadataKey  = KEYS[3]
local hashedID     = ARGV[1]
local newStatus    = tonumber(ARGV[2])
local abortedValue = tonumber(ARGV[3])

local raw = redis.call("HGET", metaKey, hashedID)
if not raw then
    return 0
end

local meta = cjson.decode(raw)
meta.ScheduleStatus = newStatus
redis.call("HSET", metaKey, hashedID, cjson.encode(meta))

if newStatus == abortedValue then
    -- HSTRLEN returns 0 for missing fields (e.g. a Rejected sentinel).
    local oldLen = redis.call("HSTRLEN", inputKey, hashedID)
    if oldLen > 0 then
        redis.call("HDEL", inputKey, hashedID)
        redis.call("HINCRBY", metadataKey, "defer_input_size", -oldLen)
    end
end

return 1
