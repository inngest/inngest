--[[

Enqueus an item within the queue.


--]]

local queueKey                = KEYS[1]           -- queue:item - hash: { $itemID: $item }
local keyPartitionMap         = KEYS[2]           -- partition:item - hash: { $workflowID: $partition }
local keyGlobalPointer        = KEYS[3]           -- partition:sorted - zset
local keyGlobalAccountPointer = KEYS[4]           -- accounts:sorted - zset
local keyAccountPartitions    = KEYS[5]           -- accounts:$accountId:partition:sorted - zset
local idempotencyKey          = KEYS[6]           -- seen:$key
local keyFnMetadata           = KEYS[7]           -- fnMeta:$id - hash
local keyPartitionA           = KEYS[8]           -- queue:sorted:$workflowID - zset
local keyPartitionB           = KEYS[9]           -- e.g. sorted:c|t:$workflowID - zset
local keyPartitionC           = KEYS[10]          -- e.g. sorted:c|t:$workflowID - zset
local keyItemIndexA           = KEYS[11]          -- custom item index 1
local keyItemIndexB           = KEYS[12]          -- custom item index 2

local queueItem           = ARGV[1]           -- {id, lease id, attempt, max attempt, data, etc...}
local queueID             = ARGV[2]           -- id
local queueScore          = tonumber(ARGV[3]) -- vesting time, in milliseconds
local partitionTime       = tonumber(ARGV[4]) -- score for partition, lower bounded to now in seconds
local nowMS               = tonumber(ARGV[5]) -- now in ms
local fnMetadata          = ARGV[6]          -- function meta: {paused}
local partitionItemA      = ARGV[7]
local partitionItemB      = ARGV[8]
local partitionItemC      = ARGV[9]
local partitionIdA        = ARGV[10]
local partitionIdB        = ARGV[11]
local partitionIdC        = ARGV[12]
local accountId           = ARGV[13]

-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
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
enqueue_to_partition(keyPartitionA, partitionIdA, partitionItemA, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, partitionTime, nowMS)
enqueue_to_partition(keyPartitionB, partitionIdB, partitionItemB, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, partitionTime, nowMS)
enqueue_to_partition(keyPartitionC, partitionIdC, partitionItemC, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, partitionTime, nowMS)

-- note to future devs: if updating metadata, be sure you do not change the "off"
-- (i.e. "paused") boolean in the function's metadata.
redis.call("SET", keyFnMetadata, fnMetadata, "NX")

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
