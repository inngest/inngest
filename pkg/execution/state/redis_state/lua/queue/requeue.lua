--[[

Re-enqueus a queue item within its queue, removing any lease.

Output:
  0: Successfully re-enqueued item
  1: Queue item not found

]]

local queueKey                = KEYS[1] -- queue:item - hash: { $itemID: $item }
local keyPartitionMap         = KEYS[2] -- partition:item - hash: { $workflowID: $partition }
local keyGlobalPointer        = KEYS[3] -- partition:sorted - zset
local keyGlobalAccountPointer = KEYS[4] -- accounts:sorted - zset
local keyAccountPartitions    = KEYS[5] -- accounts:$accountID:partition:sorted
local keyPartitionFn          = KEYS[6] -- queue:sorted:$workflowID - zset
-- We remove our queue item ID from each concurrency queue
local keyConcurrencyFn            = KEYS[7] -- Account concurrency level
local keyCustomConcurrencyKey1    = KEYS[8] -- When leasing an item we need to place the lease into this key
local keyCustomConcurrencyKey2    = KEYS[9] -- Optional for eg. for concurrency amongst steps
local keyAcctConcurrency          = KEYS[10]
-- We push pointers to partition concurrency items to the partition concurrency item
local concurrencyPointer      = KEYS[11]

-- Key queues v2
local keyBacklogSet                      = KEYS[12]          -- backlog:sorted:<backlogID> - zset
local keyBacklogMeta                     = KEYS[13]          -- backlogs - hash
local keyGlobalShadowPartitionSet        = KEYS[14]          -- shadow:sorted
local keyShadowPartitionSet              = KEYS[15]          -- shadow:sorted:<fnID|queueName> - zset
local keyShadowPartitionMeta             = KEYS[16]          -- shadows
local keyGlobalAccountShadowPartitionSet = KEYS[17]
local keyAccountShadowPartitionSet       = KEYS[18]

local keyItemIndexA           = KEYS[19]          -- custom item index 1
local keyItemIndexB           = KEYS[20]          -- custom item index 2

local queueItem           = ARGV[1]
local queueID             = ARGV[2]           -- id
local queueScore          = tonumber(ARGV[3]) -- vesting time, in ms
local nowMS               = tonumber(ARGV[4]) -- now in ms
local partitionItem       = ARGV[5]
local partitionID         = ARGV[6]
local accountID           = ARGV[7]

-- Key queues v2
local requeueToBacklog				= tonumber(ARGV[8])
local shadowPartitionItem     = ARGV[9]
local backlogItem             = ARGV[10]
local backlogID               = ARGV[11]

-- $include(get_queue_item.lua)
-- $include(get_partition_item.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(enqueue_to_partition.lua)

local item = get_queue_item(queueKey, queueID)
if item == nil then
    return 1
end

-- Update the queue item with a nil lease, at, atMS, etc.
redis.call("HSET", queueKey, queueID, queueItem)


-- This removes the queue item from the concurrency/in-progress queue and ensures that the concurrency
-- index/scavenger queue is updated to the next earliest item.
-- This is the first half of requeueing: Removing the in-progress item, which must be followed up
-- by enqueueing back to the partition queues
local function handleRequeueConcurrency(keyConcurrency)
	redis.call("ZREM", keyConcurrency, item.id) -- Remove from in-progress queue
end

--
-- Concurrency
--

handleRequeueConcurrency(keyConcurrencyFn)

-- Get the earliest item in the partition concurrency set.  We may be dequeueing
-- the only in-progress job and should remove this from the partition concurrency
-- pointers, if this exists.
--
-- This ensures that scavengeres have updated pointer queues without the currently
-- leased job, if exists.
local concurrencyScores = redis.call("ZRANGE", keyConcurrencyFn, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if concurrencyScores == false then
  redis.call("ZREM", concurrencyPointer, partitionID)
else
  local earliestLease = tonumber(concurrencyScores[2])
  if earliestLease == nil then
    redis.call("ZREM", concurrencyPointer, partitionID)
  else
    -- Ensure that we update the score with the earliest lease
    redis.call("ZADD", concurrencyPointer, earliestLease, partitionID)
  end
end

handleRequeueConcurrency(keyCustomConcurrencyKey1)
handleRequeueConcurrency(keyCustomConcurrencyKey2)

-- Remove item from the account concurrency queue
-- This does not have a scavenger queue, as it's purely an entitlement limitation. See extendLease
-- and Lease for respective ZADD calls.
redis.call("ZREM", keyAcctConcurrency, item.id)

if requeueToBacklog == 1 then
	--
	-- Requeue item to backlog queues again
	--
  requeue_to_backlog(keyBacklogSet, backlogID, backlogItem, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, queueScore, queueID, accountID)
else
  --
  -- Enqueue item to partition queues again
  --
  requeue_to_partition(keyPartitionFn, partitionID, partitionItem, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, nowMS, accountID)
end

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
    redis.call("ZADD", keyItemIndexA, queueScore, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
    redis.call("ZADD", keyItemIndexB, queueScore, queueID)
end

return 0
