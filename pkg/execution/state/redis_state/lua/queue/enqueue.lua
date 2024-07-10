--[[

Enqueus an item within the queue.


--]]

local queueKey                = KEYS[1]           -- queue:item - hash: { $itemID: $item }
local keyPartitionMap         = KEYS[2]           -- partition:item - hash: { $workflowID: $partition }
local keyGlobalPointer        = KEYS[3]           -- partition:sorted - zset
local keyGlobalAccountPointer = KEYS[4]           -- accounts:sorted - zset
local keyAccountPointer       = KEYS[5]           -- accounts:$accountId:partition:sorted - zset
local shardIndexKey           = KEYS[6]           -- shard:$name:sorted - zset
local shardMapKey             = KEYS[7]           -- shards - hmap of shards
local idempotencyKey          = KEYS[8]           -- seen:$key
local keyFnMetadata           = KEYS[9]           -- fnMeta:$id - hash
local keyPartitionA           = KEYS[10]           -- queue:sorted:$workflowID - zset
local keyPartitionB           = KEYS[11]           -- e.g. sorted:c|t:$workflowID - zset
local keyPartitionC           = KEYS[12]          -- e.g. sorted:c|t:$workflowID - zset
local keyItemIndexA           = KEYS[13]          -- custom item index 1
local keyItemIndexB           = KEYS[14]          -- custom item index 2

local queueItem           = ARGV[1]           -- {id, lease id, attempt, max attempt, data, etc...}
local queueID             = ARGV[2]           -- id
local queueScore          = tonumber(ARGV[3]) -- vesting time, in milliseconds
local partitionTime       = tonumber(ARGV[4]) -- score for partition, lower bounded to now in seconds
local shard               = ARGV[5]
local shardName           = ARGV[6]
local nowMS               = tonumber(ARGV[7]) -- now in ms
local fnMetadata          = ARGV[8]          -- function meta: {paused}
local partitionItemA      = ARGV[9]
local partitionItemB      = ARGV[10]
local partitionItemC      = ARGV[11]
local partitionIdA        = ARGV[12]
local partitionIdB        = ARGV[13]
local partitionIdC        = ARGV[14]
local accountId           = ARGV[15]

-- $include(get_partition_item.lua)
-- $include(enqueue_to_partition.lua)

-- Check idempotency exists
if redis.call("EXISTS", idempotencyKey) ~= 0 then
    return 1
end

-- Make these a hash to save on memory usage
if redis.call("HSETNX", queueKey, queueID, queueItem) == 0 then
    -- This already exists;  return an error.
    return 1
end

-- Enqueue to all partitions.
enqueue_to_partition(keyPartitionA, partitionIdA, partitionItemA, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPointer,  queueScore, queueID, partitionTime, nowMS)
enqueue_to_partition(keyPartitionB, partitionIdB, partitionItemB, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPointer,  queueScore, queueID, partitionTime, nowMS)
enqueue_to_partition(keyPartitionC, partitionIdC, partitionItemC, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPointer,  queueScore, queueID, partitionTime, nowMS)

-- Potentially update the queue of queues (global accounts).
local currentScore = redis.call("ZSCORE", keyGlobalAccountPointer, accountId)
if currentScore == false or tonumber(currentScore) > partitionTime then
  -- local existing = get_partition_item(keyPartitionMap, accountId)
  -- if nowMS > existing.forceAtMS then
  redis.call("ZADD", keyGlobalAccountPointer, partitionTime, accountId)
  -- end
end

-- note to future devs: if updating metadata, be sure you do not change the "off"
-- (i.e. "paused") boolean in the function's metadata.
redis.call("SET", keyFnMetadata, fnMetadata, "NX")

-- If this is a sharded item, upsert the shard.
if shard ~= "" and shard ~= "null" then
    -- NOTE: We do not want to overwrite the shard leases, so here
    -- we fetch the shard item, set the lease values in the passed in shard
    -- item, then write the updated value.
    local existingShard = redis.call("HGET", shardMapKey, shardName)
    if existingShard ~= nil and existingShard ~= false then
        local updatedShard = cjson.decode(shard)
        existingShard = cjson.decode(existingShard)
        updatedShard.leases = existingShard.leases
        shard = cjson.encode(updatedShard)
    end
    redis.call("HSET", shardMapKey, shardName, shard)
end

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
    redis.call("ZADD", keyItemIndexA, queueScore, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
    redis.call("ZADD", keyItemIndexB, queueScore, queueID)
end

-- TODO: For the given workflow ID increase scheduled count, store a history item,
-- etc:  this can be atomic in the redis queue as it combines state + queue.

return 0
