--[[

Peek returns items from the queue that are unleased and the vesting time <= peekUntil

]]

local queueIndex = KEYS[1]
local queueKey   = KEYS[2]

local peekFrom     = ARGV[1]
local peekUntil    = ARGV[2]
local limit        = tonumber(ARGV[3])
local offset       = tonumber(ARGV[4])
local randomOffset = ARGV[5]

if randomOffset == "1" then
	local count = redis.call("ZCOUNT", queueIndex, peekFrom, peekUntil)
	if count > limit then
		math.randomseed(tonumber(peekUntil));
		-- We have to +1 then -1 to ensure that we have 0 as a valid random offset.
		offset = math.random((count-limit)+1) - 1
	end
end


local itemIds = redis.call("ZRANGE", queueIndex, peekFrom, peekUntil, "BYSCORE", "LIMIT", offset, limit)
if #itemIds == 0 then
	return {}
end

local potentiallyMissingQueueItems = redis.call("HMGET", queueKey, unpack(itemIds))

return {potentiallyMissingQueueItems, itemIds}
