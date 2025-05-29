--[[

peekOrderedPointerSet returns items from an ordered pointer ZSET.

If sequential is 1, items are returned in order of their index.
If sequential is 0, items are returned randomly if more items are available than the limit.

]]

local keyMetadataHash        = KEYS[1]
local keyOrderedPointerSet   = KEYS[2]

local peekUntil    = tonumber(ARGV[1])
local peekUntilMS  = tonumber(ARGV[2])
local limit        = tonumber(ARGV[3])
local sequential   = tonumber(ARGV[4])


local count = redis.call("ZCOUNT", keyOrderedPointerSet, "-inf", peekUntil)
local offset = 0

if count > limit and sequential == 0 then
	math.randomseed(peekUntilMS);
	-- We have to +1 then -1 to ensure that we have 0 as a valid random offset.
	offset = math.random((count-limit)+1) - 1
end

local pointerIDs = redis.call("ZRANGE", keyOrderedPointerSet, "-inf", peekUntil, "BYSCORE", "LIMIT", offset, limit)
if #pointerIDs == 0 then
	return {}
end

local potentiallyMissingItems = redis.call("HMGET", keyMetadataHash, unpack(pointerIDs))

return {count, potentiallyMissingItems, pointerIDs}
