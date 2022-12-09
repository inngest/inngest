--[[

Peek returns partition items from the queue in order via their index.

]]

local partitionIndex = KEYS[1]
local partitionKey   = KEYS[2]

local peekUntil  = tonumber(ARGV[1])
local limit      = tonumber(ARGV[2])

local items = redis.call("ZRANGE", partitionIndex, "-inf", peekUntil, "BYSCORE", "LIMIT", 0, limit)
if #items == 0 then
	return {}
end

return redis.call("HMGET", partitionKey, unpack(items))
