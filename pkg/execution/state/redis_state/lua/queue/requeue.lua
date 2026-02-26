--[[

Re-enqueus a queue item within its queue, removing any lease.

Output:
  0: Successfully re-enqueued item
  1: Queue item not found
  2: Successfully re-queued to backlog -- TODO: this should be a temporary status
]]

local queueKey                = KEYS[1] -- queue:item - hash: { $itemID: $item }
local keyPartitionMap         = KEYS[2] -- partition:item - hash: { $workflowID: $partition }
local concurrencyPointer      = KEYS[3]

local keyGlobalPointer        = KEYS[4] -- partition:sorted - zset
local keyGlobalAccountPointer = KEYS[5] -- accounts:sorted - zset
local keyAccountPartitions    = KEYS[6] -- accounts:$accountID:partition:sorted

local keyReadyQueue           = KEYS[7] -- queue:sorted:$workflowID - zset

local keyBacklogSet                      = KEYS[8]          -- backlog:sorted:<backlogID> - zset
local keyBacklogMeta                     = KEYS[9]          -- backlogs - hash
local keyGlobalShadowPartitionSet        = KEYS[10]          -- shadow:sorted
local keyShadowPartitionSet              = KEYS[11]          -- shadow:sorted:<fnID|queueName> - zset
local keyShadowPartitionMeta             = KEYS[12]          -- shadows
local keyGlobalAccountShadowPartitionSet = KEYS[13]
local keyAccountShadowPartitionSet       = KEYS[14]

local keyPartitionScavengerIndex  = KEYS[15]

local keyItemIndexA           = KEYS[16]          -- custom item index 1
local keyItemIndexB           = KEYS[17]          -- custom item index 2

local queueID             = ARGV[1]           -- id
local queueItem           = ARGV[2]
local queueScore          = tonumber(ARGV[3]) -- vesting time, in ms
local accountID           = ARGV[4]
local partitionID         = ARGV[5]
local partitionItem       = ARGV[6]

local nowMS               = tonumber(ARGV[7]) -- now in ms

-- Key queues v2
local requeueToBacklog				= tonumber(ARGV[8])
local shadowPartitionItem     = ARGV[9]
local backlogID               = ARGV[10]
local backlogItem             = ARGV[11]

-- $include(get_queue_item.lua)
-- $include(get_partition_item.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(enqueue_to_partition.lua)
-- $include(update_active_sets.lua)

local item = get_queue_item(queueKey, queueID)
if item == nil then
    return 1
end

-- Update the queue item with a nil lease, at, atMS, etc.
redis.call("HSET", queueKey, queueID, queueItem)

-- Remove item from ready queue
redis.call("ZREM", keyReadyQueue, item.id)

-- Remove item from scavenger index
redis.call("ZREM", keyPartitionScavengerIndex, item.id)

-- Get the earliest item in the new scavenger index.  We may be dequeueing
-- the only in-progress job and should remove this from the partition concurrency
-- pointers, if this exists.
--
-- This ensures that scavengeres have updated pointer queues without the currently
-- leased job, if exists.
local scavengerIndexScores = redis.call("ZRANGE", keyPartitionScavengerIndex, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if scavengerIndexScores == false or scavengerIndexScores == nil or #scavengerIndexScores == 0 then
  redis.call("ZREM", concurrencyPointer, partitionID)
else
  local earliestLease = tonumber(scavengerIndexScores[2])

  -- Ensure that we update the score with the earliest lease
  redis.call("ZADD", concurrencyPointer, earliestLease, partitionID)
end

if requeueToBacklog == 1 then
	--
	-- Requeue item to backlog queues again
	--
  requeue_to_backlog(keyBacklogSet, backlogID, backlogItem, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, queueScore, queueID, accountID)
else
  --
  -- Enqueue item to partition queues again
  --
  requeue_to_partition(keyReadyQueue, partitionID, partitionItem, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, nowMS, accountID)
end

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
    redis.call("ZADD", keyItemIndexA, queueScore, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
    redis.call("ZADD", keyItemIndexB, queueScore, queueID)
end

return 0
