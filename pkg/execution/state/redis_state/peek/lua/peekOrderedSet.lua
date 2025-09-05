--[[

peekPointerSet returns items from a pointer ZSET.

]]

local keyMetadataHash = KEYS[1]
local keyPointerSet = KEYS[2]

local limit = tonumber(ARGV[1])

local count = redis.call("ZCARD", keyPointerSet)

local pointers = redis.call("ZRANGE", keyPointerSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, limit)
if #pointers == 0 then
	return {}
end

local potentiallyMissingItems = redis.call("HMGET", keyMetadataHash, unpack(pointers))

local lastItemID = pointers[#pointers]
local cursor = tonumber(redis.call("ZSCORE", keyPointerSet, lastItemID))

return { count, potentiallyMissingItems, pointers, cursor }
