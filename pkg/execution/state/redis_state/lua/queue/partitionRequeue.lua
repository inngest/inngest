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
local keyAccountPartitionPtr  = KEYS[3]
local keyGlobalAccountsPtr    = KEYS[4]
local keyShardPartitionPtr    = KEYS[5]
local partitionMeta           = KEYS[6]
local queueIndex              = KEYS[7]
local queueKey                = KEYS[8]
local partitionConcurrencyKey = KEYS[9] -- We can only GC a partition if no running jobs occur.

local workflowID              = ARGV[1]
local atMS                    = tonumber(ARGV[2]) -- time in milliseconds
local forceAt                 = tonumber(ARGV[3])

local atS = math.floor(atMS / 1000) -- in seconds;  partitions are currently second granularity, but this should change.

-- $include(get_partition_item.lua)
-- $include(has_shard_key.lua)
--
local existing                = get_partition_item(partitionKey, workflowID)
if existing == nil then
    return 1
end

-- If there are no items in the workflow queue, we can safely remove the
-- partition.
if tonumber(redis.call("ZCARD", queueIndex)) == 0 and tonumber(redis.call("ZCARD", partitionConcurrencyKey)) == 0 then
    redis.call("HDEL", partitionKey, workflowID)             -- Remove the item
    redis.call("DEL", partitionMeta)                         -- Remove the meta
    redis.call("ZREM", keyGlobalPartitionPtr, workflowID)    -- Remove the global index
    redis.call("ZREM", keyAccountPartitionPtr, workflowID)    -- Remove the account-level index

    -- Remove account from global accounts if there are no partitions to work on
    local account_items = tonumber(redis.call("ZCARD", keyAccountPartitionPtr))
    if (account_items == 0) then
      redis.call("ZREM", keyGlobalAccountsPtr, accountID)
    end

    if has_shard_key(keyShardPartitionPtr) then
        redis.call("ZREM", keyShardPartitionPtr, workflowID) -- Remove the shard index
    end
    return 2
end

-- Peek up the next available item from the queue
local items = redis.call("ZRANGE", queueIndex, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1)

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
redis.call("HSET", partitionKey, workflowID, cjson.encode(existing))
redis.call("ZADD", keyGlobalPartitionPtr, atS, workflowID)
redis.call("ZADD", keyAccountPartitionPtr, atS, workflowID)
-- TODO Do we need to update the global account ZSET?
if has_shard_key(keyShardPartitionPtr) then
    redis.call("ZADD", keyShardPartitionPtr, atS, workflowID) -- Update any index
end

return 0
