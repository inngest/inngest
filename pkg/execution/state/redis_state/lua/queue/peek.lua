--[[

Peek returns items from the queue that are unleased and the vesting time <= peekUntil

]]

local queueIndex = KEYS[1]
local queueKey   = KEYS[2]

local peekUntil  = ARGV[1]
local offset     = tonumber(ARGV[2])
local limit      = tonumber(ARGV[3])

local items = redis.call("ZRANGE", queueIndex, "-inf", peekUntil, "BYSCORE", "LIMIT", offset, limit)
if #items == 0 then
	return {}
end

return redis.call("HMGET", queueKey, unpack(items))
