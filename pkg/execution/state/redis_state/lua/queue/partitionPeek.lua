--[[

Peek returns partition items from the queue in order via their index.

]]

local partitionIndex = KEYS[1]
local partitionKey   = KEYS[2]

local peekUntilMS  = tonumber(ARGV[1])
local limit        = tonumber(ARGV[2])
local sequential   = tonumber(ARGV[3])
local after        = ARGV[4]

local peekUntil    = math.ceil(peekUntilMS / 1000)

local count = redis.call("ZCOUNT", partitionIndex, "-inf", peekUntil)
local offset = 0

if count > limit and sequential == 0 then
	math.randomseed(peekUntilMS);
	-- We have to +1 then -1 to ensure that we have 0 as a valid random offset.
	offset = math.random((count-limit)+1) - 1
elseif sequential == 1 and after ~= nil and after ~= "" then
	-- Re-read the member's rank so removals before it don't skip entries.
	-- Wrap to the head if the member is gone or no due entries follow it.
	local rank = redis.call("ZRANK", partitionIndex, after)
	if rank ~= false and rank ~= nil and rank + 1 < count then
		offset = rank + 1
	end
end

if count == 0 then
	return {}
end
-- The first count members are due. Rank lookup avoids the linear traversal
-- that a large BYSCORE offset would require.
local partitionIds = redis.call("ZRANGE", partitionIndex, offset, math.min(offset + limit, count) - 1)
if #partitionIds == 0 then
	return {}
end

local potentiallyMissingPartitions = redis.call("HMGET", partitionKey, unpack(partitionIds))


return {count - offset - #partitionIds, potentiallyMissingPartitions, partitionIds}
