--[[

peekOrderedPointerSet returns items from an ordered pointer ZSET.

If sequential is 1, items are returned in order of their index.
If sequential is 0, items are returned randomly if more items are available than the limit.

]]

local keyMetadataHash        = KEYS[1]
local keyOrderedPointerSet   = KEYS[2]

local from       = ARGV[1]
local to         = ARGV[2]

local limit      = tonumber(ARGV[3])
local offset     = tonumber(ARGV[4])

local untilMS    = tonumber(ARGV[5])
local sequential = tonumber(ARGV[6])

-- NOTE: ZCOUNT becomes unreliable when combined with the `offset` option
local count = redis.call("ZCOUNT", keyOrderedPointerSet, from, to)

if count > limit and sequential == 0 then
	math.randomseed(untilMS);
	-- We have to +1 then -1 to ensure that we have 0 as a valid random offset.
	offset = math.random((count-limit)+1) - 1
end

local pointers = redis.call("ZRANGE", keyOrderedPointerSet, from, to, "BYSCORE", "LIMIT", offset, limit)
if #pointers == 0 then
	return {}
end

local potentiallyMissingItems = redis.call("HMGET", keyMetadataHash, unpack(pointers))

return { count, potentiallyMissingItems, pointers }
