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

local items = redis.call("ZRANGE", partitionIndex, "-inf", peekUntil, "BYSCORE", "LIMIT", offset, limit)
if #items == 0 then
	return {}
end

local partitions = redis.call("HMGET", partitionKey, unpack(items))

-- if there's a nil value in the partition at position i, we need to remove it and add it to a separate set of missing partitions
local missingPartitions = {}
for i, partition in ipairs(partitions) do
	if partition == false then
		table.insert(missingPartitions, items[i])
		table.remove(partitions, i)
	end
end

return {partitions, missingPartitions}
