--[[

Peek returns items from the queue that are unleased and the vesting time <= peekUntil

]]

local queueIndex = KEYS[1]
local queueKey   = KEYS[2]

local peekUntil  = ARGV[1]
local limit      = tonumber(ARGV[2])

local itemIds = redis.call("ZRANGE", queueIndex, "-inf", peekUntil, "BYSCORE", "LIMIT", 0, limit)
if #itemIds == 0 then
	return {}
end

local potentiallyMissingQueueItems = redis.call("HMGET", queueKey, unpack(itemIds))

return {potentiallyMissingQueueItems, itemIds}
