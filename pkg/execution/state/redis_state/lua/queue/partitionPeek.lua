--[[

Peek returns partition items from the queue in order via their index.

]]

local partitionIndex = KEYS[1]
local partitionKey   = KEYS[2]

local peekUntilMS  = tonumber(ARGV[1])
local limit        = tonumber(ARGV[2])
local sequential   = tonumber(ARGV[3])

local peekUntil    = math.ceil(peekUntilMS / 1000)

local count = redis.call("ZCOUNT", partitionIndex, "-inf", peekUntil)
local offset = 0

if count > limit and sequential == 0 then
	math.randomseed(peekUntilMS);
	-- We have to +1 then -1 to ensure that we have 0 as a valid random offset.
	offset = math.random((count-limit)+1) - 1
end

local partitionIds = redis.call("ZRANGE", partitionIndex, "-inf", peekUntil, "BYSCORE", "LIMIT", offset, limit)
if #partitionIds == 0 then
	return {}
end

local potentiallyMissingPartitions = redis.call("HMGET", partitionKey, unpack(partitionIds))


return {count, potentiallyMissingPartitions, partitionIds}
