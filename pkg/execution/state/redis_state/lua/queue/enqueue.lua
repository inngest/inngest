--[[

Enqueus an item within the queue.


--]]

local queueKey                	= KEYS[1]           -- queue:item - hash: { $itemID: $item }
local keyPartitionMap         	= KEYS[2]           -- partition:item - hash: { $workflowID: $partition }
local keyGlobalPointer        	= KEYS[3]           -- partition:sorted - zset
local keyGlobalAccountPointer 	= KEYS[4]           -- accounts:sorted - zset
local keyAccountPartitions    	= KEYS[5]           -- accounts:$accountID:partition:sorted - zset
local idempotencyKey          	= KEYS[6]           -- seen:$key
local keyFnMetadata           	= KEYS[7]           -- fnMeta:$id - hash
local keyPartition           	  = KEYS[8]           -- queue:sorted:$workflowID - zset

-- Key queues v2
local keyBacklogSetA                     = KEYS[9]          -- backlog:sorted:<backlogID> - zset
local keyBacklogSetB                     = KEYS[10]          -- backlog:sorted:<backlogID> - zset
local keyBacklogSetC                     = KEYS[11]          -- backlog:sorted:<backlogID> - zset
local keyBacklogMeta                     = KEYS[12]          -- backlogs - hash
local keyGlobalShadowPartitionSet        = KEYS[13]          -- shadow:sorted
local keyShadowPartitionSet              = KEYS[14]          -- shadow:sorted:<fnID|queueName> - zset
local keyShadowPartitionMeta             = KEYS[15]          -- shadows
local keyGlobalAccountShadowPartitionSet = KEYS[16]
local keyAccountShadowPartitionSet       = KEYS[17]

local keyItemIndexA           	= KEYS[18]          -- custom item index 1
local keyItemIndexB           	= KEYS[19]          -- custom item index 2


local queueItem           		= ARGV[1]           -- {id, lease id, attempt, max attempt, data, etc...}
local queueID             		= ARGV[2]           -- id
local queueScore          		= tonumber(ARGV[3]) -- vesting time, in milliseconds
local partitionTime       		= tonumber(ARGV[4]) -- score for partition, lower bounded to now in seconds
local nowMS               		= tonumber(ARGV[5]) -- now in ms
local fnMetadata          		= ARGV[6]          -- function meta: {paused}
local partitionItem      		  = ARGV[7]
local partitionID        		  = ARGV[8]
local accountID           		= ARGV[9]

-- Key queues v2
local enqueueToBacklog				= tonumber(ARGV[10])
local shadowPartitionItem     = ARGV[11]
local backlogItemA            = ARGV[12]
local backlogItemB            = ARGV[13]
local backlogItemC            = ARGV[14]
local backlogIdA              = ARGV[15]
local backlogIdB              = ARGV[16]
local backlogIdC              = ARGV[17]

-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(get_partition_item.lua)
-- $include(enqueue_to_partition.lua)
-- $include(ends_with.lua)

-- Check idempotency exists
if redis.call("EXISTS", idempotencyKey) ~= 0 then
    return 1
end

-- Make these a hash to save on memory usage
if redis.call("HSETNX", queueKey, queueID, queueItem) == 0 then
    -- This already exists;  return an error.
    return 1
end

if enqueueToBacklog == 1 then
	-- the default function queue could be any of the three, usually the first but possibly the middle or last if a custom concurrency key is used

	enqueue_to_backlog(keyBacklogSetA, backlogIdA, backlogItemA, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, queueScore, queueID, partitionTime, nowMS, accountID)
	enqueue_to_backlog(keyBacklogSetB, backlogIdB, backlogItemB, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, queueScore, queueID, partitionTime, nowMS, accountID)
	enqueue_to_backlog(keyBacklogSetC, backlogIdC, backlogItemC, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, queueScore, queueID, partitionTime, nowMS, accountID)
else
  enqueue_to_partition(keyPartition, partitionID, partitionItem, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, partitionTime, nowMS, accountID)
end

if exists_without_ending(keyFnMetadata, ":fnMeta:-") == true then
	-- note to future devs: if updating metadata, be sure you do not change the "off"
	-- (i.e. "paused") boolean in the function's metadata.
	redis.call("SET", keyFnMetadata, fnMetadata, "NX")
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
