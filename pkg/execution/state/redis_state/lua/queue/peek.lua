--[[

Peek returns items from the queue that are unleased and the vesting time <= peekUntil

]]

local queueIndex = KEYS[1]
local queueKey   = KEYS[2]

local peekUntil  = ARGV[1]
local limit      = tonumber(ARGV[2])

local items = redis.call("ZRANGE", queueIndex, "-inf", peekUntil, "BYSCORE", "LIMIT", 0, limit)
if #items == 0 then
	return {}
end

local queueItems = redis.call("HMGET", queueKey, unpack(items))

-- if there's a nil value in the queue item at position i, we need to remove it and add it to a separate set of missing queue items
local missingQueueItems = {}
for i, qi in ipairs(queueItems) do
	if qi == false then
		table.insert(missingQueueItems, items[i])
		table.remove(queueItems, i)
	end
end

return {queueItems, missingQueueItems}
