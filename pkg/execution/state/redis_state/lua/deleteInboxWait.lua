--[[

Removes the wait of an inbox once its wake is queued. Only the wait with the
given id is removed: the read after it may have been recorded already, and
that wait is not this call's to delete.

]]

local keyWaits = KEYS[1]

local inbox = ARGV[1]
local waitID = ARGV[2]

local record = redis.call("HGET", keyWaits, inbox)
if record and cjson.decode(record).id == waitID then
	redis.call("HDEL", keyWaits, inbox)
end

return 1
