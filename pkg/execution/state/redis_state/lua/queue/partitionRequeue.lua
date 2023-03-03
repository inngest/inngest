--[[

  Requeues a partition at a specific time.
  will take into account the new priority when weighted sampling items to
  work on.

  Return values:
  0 - Updated priority
  1 - Partition not found
  2 - Partition deleted

]]

local partitionKey   = KEYS[1]
local partitionIndex = KEYS[2]
local partitionMeta  = KEYS[3]
local queueIndex     = KEYS[4]
local queueKey       = KEYS[5]
local partitionConcurrencyKey = KEYS[6] -- We can only GC a partition if no running jobs occur.

local workflowID = ARGV[1]
local at         = tonumber(ARGV[2]) -- time in seconds
local forceAt    = tonumber(ARGV[3])

-- $include(get_partition_item.lua)
local existing = get_partition_item(partitionKey, workflowID)
if existing == nil then
	return 1
end

-- If there are no items in the workflow queue, we can safely remove the
-- partition.
if tonumber(redis.call("ZCARD", queueIndex)) == 0 and tonumber(redis.call("ZCARD", partitionConcurrencyKey)) == 0 then
	redis.call("HDEL", partitionKey, workflowID) -- Remove the item
	redis.call("DEL", partitionMeta) -- Remove the meta
	redis.call("ZREM", partitionIndex, workflowID) -- Remove the index
	return 2
end

-- Peek up the next available item from the queue
local items = redis.call("ZRANGE", queueIndex, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1)

if #items > 0 and forceAt ~= 1 then
	-- score = redis.call("ZSCORE", queueIndex, items[0])
	-- at = math.floor(Vcore / 1000)
	local encoded = redis.call("HMGET", queueKey, unpack(items))
	for k, v in pairs(encoded) do
		local item = cjson.decode(v)
		if (item.leaseID == nil or item.leaseID == cjson.null) and math.floor(item.at / 1000) < at then
			at = math.floor(item.at / 1000)
			break
		end
	end
end


existing.at = at
existing.leaseID = nil
redis.call("HSET", partitionKey, workflowID, cjson.encode(existing))
redis.call("ZADD", partitionIndex, at, workflowID)

return 0
