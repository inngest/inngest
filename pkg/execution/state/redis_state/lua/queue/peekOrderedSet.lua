--[[

peekPointerSet returns items from a pointer ZSET.

]]

local keyMetadataHash        = KEYS[1]
local keyPointerSet          = KEYS[2]

local limit        = tonumber(ARGV[1])

local count = redis.call("ZCARD", keyPointerSet)

local pointerIDs = redis.call("ZRANGE", keyPointerSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, limit)
if #pointerIDs == 0 then
	return {}
end

local potentiallyMissingItems = redis.call("HMGET", keyMetadataHash, unpack(pointerIDs))

local lastItemID = pointerIDs[#pointerIDs]
local cursor = tonumber(redis.call("ZSCORE", keyPointerSet, lastItemID))

return { count, potentiallyMissingItems, pointerIDs, cursor }
