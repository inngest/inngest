--[[
Atomically update only the ScheduleStatus field of a Defer record.

Only the meta hash field is read/written here — Input lives in a separate
field (input:<hashedID>) and is never round-tripped through cjson by this
script. The meta payload contains only strings and a small integer status,
so the cjson decode/encode round-trip is safe by construction.

Avoids a read-modify-write race against a concurrent SaveDefer: the whole
op happens inside a single Redis Lua invocation, which Redis runs atomically.

KEYS[1] - defers hash key
ARGV[1] - hashedID
ARGV[2] - new status (integer, as a string)

Output:
  1: status updated
  0: defer not found
]]

local defersKey = KEYS[1]
local hashedID  = ARGV[1]
local newStatus = tonumber(ARGV[2])

local metaField = "meta:" .. hashedID

local raw = redis.call("HGET", defersKey, metaField)
if not raw then
    return 0
end

local meta = cjson.decode(raw)
meta.ScheduleStatus = newStatus
redis.call("HSET", defersKey, metaField, cjson.encode(meta))
return 1
