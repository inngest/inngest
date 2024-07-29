--[[

  Requeues a partition at a specific time.
  will take into account the new priority when weighted sampling items to
  work on.

  Return values:
  0 - Updated priority
  1 - Partition not found
  2 - Partition deleted

]]

local partitionKey            = KEYS[1]
local keyGlobalPartitionPtr   = KEYS[2]
local keyGlobalAccountPointer = KEYS[4] -- accounts:sorted - zset
local keyAccountPartitions    = KEYS[5] -- accounts:$accountId:partition:sorted - zset
local keyShardPartitionPtr    = KEYS[3]
local partitionMeta           = KEYS[4]
local keyPartitionZset        = KEYS[5]
local partitionConcurrencyKey = KEYS[6] -- We can only GC a partition if no running jobs occur.
local queueKey                = KEYS[7]

local partitionID             = ARGV[1]
local atMS                    = tonumber(ARGV[2]) -- time in milliseconds
local forceAt                 = tonumber(ARGV[3])
local accountId               = ARGV[4]

local atS = math.floor(atMS / 1000) -- in seconds;  partitions are currently second granularity, but this should change.

-- $include(get_partition_item.lua)
-- $include(has_shard_key.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)

local existing = get_partition_item(partitionKey, partitionID)
if existing == nil then
    return 1
end

-- If there are no items in the workflow queue, we can safely remove the
-- partition.
if tonumber(redis.call("ZCARD", keyPartitionZset)) == 0 and tonumber(redis.call("ZCARD", partitionConcurrencyKey)) == 0 then
    redis.call("HDEL", partitionKey, partitionID)             -- Remove the item
    redis.call("DEL", partitionMeta)                         -- Remove the meta

    redis.call("ZREM", keyGlobalPartitionPtr, partitionID)    -- Remove the partition from global index

    if account_is_set(keyAccountPartitions) then
      redis.call("ZREM", keyAccountPartitions, partitionID)    -- Remove the partition from account index

      -- If this was the last account partition, remove account from global queue of accounts
      local numAccountPartitions = tonumber(redis.call("ZCARD", keyAccountPartitions))
      if numAccountPartitions == 0 then
        redis.call("ZREM", keyGlobalAccountPointer, accountId)
      end
    end

    if has_shard_key(keyShardPartitionPtr) then
        redis.call("ZREM", keyShardPartitionPtr, partitionID) -- Remove the shard index
    end
    return 2
end

-- Peek up the next available item from the queue
local items = redis.call("ZRANGE", keyPartitionZset, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1)

if #items > 0 and forceAt ~= 1 then
    local encoded = redis.call("HMGET", queueKey, unpack(items))
    for k, v in pairs(encoded) do
        local item = cjson.decode(v)
        if (item.leaseID == nil or item.leaseID == cjson.null) and math.floor(item.at / 1000) < atS then
            atS = math.floor(item.at / 1000)
            break
        end
    end
end

if forceAt == 1 then
	existing.forceAtMS = atMS
else
	existing.forceAtMS = 0
end


existing.at = atS
existing.leaseID = nil
redis.call("HSET", partitionKey, partitionID, cjson.encode(existing))
update_pointer_score_to(partitionID, keyGlobalPartitionPtr, atS)
update_account_queues(keyGlobalAccountPointer, keyAccountPartitions, partitionID, accountId, atS)

if has_shard_key(keyShardPartitionPtr) then
    redis.call("ZADD", keyShardPartitionPtr, atS, partitionID) -- Update any index
end

return 0
