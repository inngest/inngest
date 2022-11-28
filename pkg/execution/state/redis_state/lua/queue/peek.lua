--[[

Peek returns items from the queue that are unleased.

]]

local queueIndex = KEYS[1]
local queueKey   = KEYS[2]

local peekUntil  = ARGV[1]
local limit      = tonumber(ARGV[2])

local items = redis.call("ZRANGE", queueIndex, "-inf", peekUntil, "BYSCORE", "LIMIT", 0, limit)
if #items == 0 then
	return {}
end

return redis.call("HMGET", queueKey, unpack(items))
