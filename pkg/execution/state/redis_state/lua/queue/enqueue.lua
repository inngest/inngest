--[[

Enqueus an item within the queue.


--]]

local queueKey                  = KEYS[1]           -- queue:item - hash: { $itemID: $item }
local keyPartitionMap           = KEYS[2]           -- partition:item - hash: { $workflowID: $partition }
local keyGlobalPointer          = KEYS[3]           -- partition:sorted - zset
local keyGlobalAccountPointer   = KEYS[4]           -- accounts:sorted - zset
local keyAccountPointer         = KEYS[5]           -- accounts:$accountId:partition:sorted - zset
local guaranteedCapacityMapKey  = KEYS[6]           -- shards - hmap of shards
local idempotencyKey            = KEYS[7]           -- seen:$key
local keyFnMetadata             = KEYS[8]           -- fnMeta:$id - hash
local keyPartitionA             = KEYS[9]           -- queue:sorted:$workflowID - zset
local keyPartitionB             = KEYS[10]           -- e.g. sorted:c|t:$workflowID - zset
local keyPartitionC             = KEYS[11]          -- e.g. sorted:c|t:$workflowID - zset
local keyItemIndexA             = KEYS[12]          -- custom item index 1
local keyItemIndexB             = KEYS[13]          -- custom item index 2

local queueItem               = ARGV[1]           -- {id, lease id, attempt, max attempt, data, etc...}
local queueID                 = ARGV[2]           -- id
local queueScore              = tonumber(ARGV[3]) -- vesting time, in milliseconds
local partitionTime           = tonumber(ARGV[4]) -- score for partition, lower bounded to now in seconds
local guaranteedCapacity      = ARGV[5]
local guaranteedCapacityName  = ARGV[6]
local nowMS                   = tonumber(ARGV[7]) -- now in ms
local fnMetadata              = ARGV[8]          -- function meta: {paused}
local partitionItemA          = ARGV[9]
local partitionItemB          = ARGV[10]
local partitionItemC          = ARGV[11]
local partitionIdA            = ARGV[12]
local partitionIdB            = ARGV[13]
local partitionIdC            = ARGV[14]
local accountId               = ARGV[15]

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
enqueue_to_partition(keyPartitionA, partitionIdA, partitionItemA, keyPartitionMap, keyGlobalPointer, keyAccountPointer,  queueScore, queueID, partitionTime, nowMS)
enqueue_to_partition(keyPartitionB, partitionIdB, partitionItemB, keyPartitionMap, keyGlobalPointer, keyAccountPointer,  queueScore, queueID, partitionTime, nowMS)
enqueue_to_partition(keyPartitionC, partitionIdC, partitionItemC, keyPartitionMap, keyGlobalPointer, keyAccountPointer,  queueScore, queueID, partitionTime, nowMS)

-- Potentially update the account index (global accounts pointers).
local currentScore = redis.call("ZSCORE", keyGlobalAccountPointer, accountId)
if currentScore == false or tonumber(currentScore) > partitionTime then
  redis.call("ZADD", keyGlobalAccountPointer, partitionTime, accountId)
end

-- note to future devs: if updating metadata, be sure you do not change the "off"
-- (i.e. "paused") boolean in the function's metadata.
redis.call("SET", keyFnMetadata, fnMetadata, "NX")

-- If the account has guaranteed capacity, upsert the guaranteed capacity map.
if guaranteedCapacity ~= "" and guaranteedCapacity ~= "null" then
    -- NOTE: We do not want to overwrite the account leases, so here
    -- we fetch the guaranteed capacity item, set the lease values in the passed in guaranteed capacity
    -- item, then write the updated value.
    local existingGuaranteedCapacity = redis.call("HGET", guaranteedCapacityMapKey, guaranteedCapacityName)
    if existingGuaranteedCapacity ~= nil and existingGuaranteedCapacity ~= false then
        local updatedGuaranteedCapacity = cjson.decode(guaranteedCapacity)
        existingGuaranteedCapacity = cjson.decode(existingGuaranteedCapacity)
        updatedGuaranteedCapacity.leases = existingGuaranteedCapacity.leases
        guaranteedCapacity = cjson.encode(updatedGuaranteedCapacity)
    end
    redis.call("HSET", guaranteedCapacityMapKey, guaranteedCapacityName, guaranteedCapacity)
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
