--[[
Atomically update only the ScheduleStatus field of a Defer record.

Avoids a read-modify-write race against a concurrent SaveDefer: the whole
op happens inside a single Redis Lua invocation, which Redis runs atomically.

KEYS[1] - defers hash key
ARGV[1] - hashedID (hash field)
ARGV[2] - new status (integer, as a string)

Output:
  1: status updated
  0: defer not found

Note: cjson round-trips nested JSON objects correctly, but an *empty* JSON
object ({}) in a Defer's Input field will be re-encoded as [] because Lua
cannot distinguish empty objects from empty arrays. Callers should avoid
storing empty-object inputs (use null or omit the field instead).
]]

local defersKey = KEYS[1]
local field     = ARGV[1]
local newStatus = tonumber(ARGV[2])

local raw = redis.call("HGET", defersKey, field)
if not raw then
    return 0
end

local defer = cjson.decode(raw)
defer.ScheduleStatus = newStatus
redis.call("HSET", defersKey, field, cjson.encode(defer))
return 1
