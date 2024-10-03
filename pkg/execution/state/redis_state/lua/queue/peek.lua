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

local potentiallyMissingQueueItems = redis.call("HMGET", queueKey, unpack(items))
local missingQueueItems = {}
local validQueueItems = {}

for i, queueItem in ipairs(potentiallyMissingQueueItems) do
	if queueItem ~= false and queueItem ~= nil then
		table.insert(validQueueItems, queueItem)
	else
		table.insert(missingQueueItems, items[i])
	end
end

return {validQueueItems, missingQueueItems}
