--[[

Enqueus an item within the queue.


--]]

local queueKey                	= KEYS[1]           -- queue:item - hash: { $itemID: $item }
local keyPartitionMap         	= KEYS[2]           -- partition:item - hash: { $workflowID: $partition }
local keyGlobalPointer        	= KEYS[3]           -- partition:sorted - zset
local keyGlobalAccountPointer 	= KEYS[4]           -- accounts:sorted - zset
local keyAccountPartitions    	= KEYS[5]           -- accounts:$accountID:partition:sorted - zset
local idempotencyKey          	= KEYS[6]           -- seen:$key
local keyPartition           	  = KEYS[7]           -- queue:sorted:$workflowID - zset

-- Key queues v2
local keyBacklogSet                      = KEYS[8]          -- backlog:sorted:<backlogID> - zset
local keyBacklogMeta                     = KEYS[9]          -- backlogs - hash
local keyGlobalShadowPartitionSet        = KEYS[10]          -- shadow:sorted
local keyShadowPartitionSet              = KEYS[11]          -- shadow:sorted:<fnID|queueName> - zset
local keyShadowPartitionMeta             = KEYS[12]          -- shadows
local keyGlobalAccountShadowPartitionSet = KEYS[13]
local keyAccountShadowPartitionSet       = KEYS[14]

local keyNormalizeFromBacklogSet         = KEYS[15] -- signals if this is part of a normalization
local keyPartitionNormalizeSet           = KEYS[16]
local keyAccountNormalizeSet             = KEYS[17]
local keyGlobalNormalizeSet              = KEYS[18]

local singletonRunKey           	  = KEYS[19]
local singletonKey           	  = KEYS[20]

local keyItemIndexA           	= KEYS[21]          -- custom item index 1
local keyItemIndexB           	= KEYS[22]          -- custom item index 2

local queueItem           		= ARGV[1]           -- {id, lease id, attempt, max attempt, data, etc...}
local queueID             		= ARGV[2]           -- id
local queueScore          		= tonumber(ARGV[3]) -- vesting time, in milliseconds
local partitionTime       		= tonumber(ARGV[4]) -- score for partition, lower bounded to now in seconds
local nowMS               		= tonumber(ARGV[5]) -- now in ms
local partitionItem      		  = ARGV[6]
local partitionID        		  = ARGV[7]
local accountID           		= ARGV[8]
local runID                   = ARGV[9]

-- Key queues v2
local enqueueToBacklog				= tonumber(ARGV[10])
local shadowPartitionItem     = ARGV[11]
local backlogItem             = ARGV[12]
local backlogID               = ARGV[13]
local normalizeFromBacklogID  = ARGV[14]

-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(get_partition_item.lua)
-- $include(enqueue_to_partition.lua)
-- $include(ends_with.lua)
-- $include(update_backlog_pointer.lua)

-- Only skip idempotency checks if we're normalizing a backlog (we want to enqueue an existing item to a new backlog)
local is_normalize = exists_without_ending(keyNormalizeFromBacklogSet, ":-")

-- Check idempotency exists
if redis.call("EXISTS", idempotencyKey) ~= 0 and not is_normalize then
  return 1
end

-- Make these a hash to save on memory usage
if redis.call("HSETNX", queueKey, queueID, queueItem) == 0 then
  if is_normalize then
    redis.call("HSET", queueKey, queueID, queueItem)
  else
    -- This already exists;  return an error.
    return 1
  end
end

-- Check if the item is a singleton and if an existing item already exists
if exists_without_ending(singletonKey, ":singleton:-") then 
  if redis.call("EXISTS", singletonKey) ~= 0 then
    return 2
  end

  -- Set the singleton key to the item ID
  redis.call("SET", singletonRunKey, singletonKey)
  redis.call("SET", singletonKey, runID)
end

if enqueueToBacklog == 1 then
	enqueue_to_backlog(keyBacklogSet, backlogID, backlogItem, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, queueScore, queueID, partitionTime, nowMS, accountID)
else
  enqueue_to_partition(keyPartition, partitionID, partitionItem, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, partitionTime, nowMS, accountID)
end

-- Normalization only: Remove from old backlog after enqueueing to new backlog
if is_normalize then
  redis.call("ZREM", keyNormalizeFromBacklogSet, queueID)

  -- Clean up backlog pointers for old backlog
  updateBacklogPointer(keyShadowPartitionMeta, keyBacklogMeta, keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyNormalizeFromBacklogSet, keyPartitionNormalizeSet, accountID, partitionID, normalizeFromBacklogID)

  -- Clean up normalize pointers if backlog is empty
  if tonumber(redis.call("ZCARD", keyNormalizeFromBacklogSet)) == 0 then
    -- Clean up normalize pointer from partition -> normalizeFromBacklogID
    redis.call("ZREM", keyPartitionNormalizeSet, normalizeFromBacklogID)

    -- If no more backlogs to normalize in partition, clean up account -> partition pointer
    if tonumber(redis.call("ZCARD", keyPartitionNormalizeSet)) == 0 then
      redis.call("ZREM", keyAccountNormalizeSet, partitionID)

      -- If no more partitions to normalize in account, clean up global -> account pointer
      if tonumber(redis.call("ZCARD", keyAccountNormalizeSet)) == 0 then
        redis.call("ZREM", keyGlobalNormalizeSet, accountID)
      end
    end
  end
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
