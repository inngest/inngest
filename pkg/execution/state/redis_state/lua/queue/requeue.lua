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
local keyAccountPartitions    = KEYS[5] -- accounts:$accountId:partition:sorted
local keyPartitionFn          = KEYS[6] -- queue:sorted:$workflowID - zset
-- We remove our queue item ID from each concurrency queue
local keyConcurrencyFn            = KEYS[7] -- Account concurrency level
local keyCustomConcurrencyKey1    = KEYS[8] -- When leasing an item we need to place the lease into this key
local keyCustomConcurrencyKey2    = KEYS[9] -- Optional for eg. for concurrency amongst steps
local keyAcctConcurrency          = KEYS[10]
-- We push pointers to partition concurrency items to the partition concurrency item
local concurrencyPointer      = KEYS[11]

-- Key queues v2
local keyBacklogSetA              = KEYS[12]          -- backlog:sorted:<backlogID> - zset
local keyBacklogSetB              = KEYS[13]          -- backlog:sorted:<backlogID> - zset
local keyBacklogSetC              = KEYS[14]          -- backlog:sorted:<backlogID> - zset
local keyBacklogMeta              = KEYS[15]          -- backlogs - hash
local keyGlobalShadowPartitionSet = KEYS[16]          -- shadow:sorted
local keyShadowPartitionSet       = KEYS[17]          -- shadow:sorted:<fnID|queueName> - zset
local keyShadowPartitionMeta      = KEYS[18]          -- shadows
local keyInProgress               = KEYS[19]
local keyAccountInProgress        = KEYS[20]
local keyActiveJobsKey1           = KEYS[21]
local keyActiveJobsKey2           = KEYS[22]

local keyItemIndexA           = KEYS[23]          -- custom item index 1
local keyItemIndexB           = KEYS[24]          -- custom item index 2

local queueItem           = ARGV[1]
local queueID             = ARGV[2]           -- id
local queueScore          = tonumber(ARGV[3]) -- vesting time, in ms
local nowMS               = tonumber(ARGV[4]) -- now in ms
local partitionItem     = ARGV[5]
local partitionID         = ARGV[6]
local accountId           = ARGV[7]

-- Key queues v2
local requeueToBacklog				= tonumber(ARGV[8])
local shadowPartitionItem     = ARGV[9]
local backlogItemA            = ARGV[10]
local backlogItemB            = ARGV[11]
local backlogItemC            = ARGV[12]
local backlogIdA              = ARGV[13]
local backlogIdB              = ARGV[14]
local backlogIdC              = ARGV[15]

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

-- Accounting for key queues v2

-- We need to update new key-specific concurrency indexes, as well as account + function level concurrency
-- as accounting is completely separate to allow for a gradual migration. Once key queues v2 are fully rolled out,
-- we can remove the old accounting logic above.

-- account-level concurrency (ignored for system queues)
if exists_without_ending(keyAccountInProgress, ":-") == true then
	redis.call("ZREM", keyAccountInProgress, item.id)
end

-- function-level concurrency
if exists_without_ending(keyInProgress, ":-") == true then
	redis.call("ZREM", keyInProgress, item.id)
end

-- backlog 1 (concurrency key 1)
if exists_without_ending(keyActiveJobsKey1, ":-") == true then
	redis.call("ZREM", keyActiveJobsKey1, item.id)
end

-- backlog 2 (concurrency key 2)
if exists_without_ending(keyActiveJobsKey2, ":-") == true then
	redis.call("ZREM", keyActiveJobsKey2, item.id)
end

-- Remove item from the account concurrency queue
-- This does not have a scavenger queue, as it's purely an entitlement limitation. See extendLease
-- and Lease for respective ZADD calls.
redis.call("ZREM", keyAcctConcurrency, item.id)

if requeueToBacklog == 1 then
	--
	-- Requeue item to backlog queues again
	--
  requeue_to_backlog(keyBacklogSetA, backlogIdA, backlogItemA, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)
  requeue_to_backlog(keyBacklogSetB, backlogIdB, backlogItemB, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)
  requeue_to_backlog(keyBacklogSetC, backlogIdC, backlogItemC, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)
else
  --
  -- Enqueue item to partition queues again
  --
  requeue_to_partition(keyPartitionFn, partitionID, partitionItem, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, nowMS, accountId)
end

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
    redis.call("ZADD", keyItemIndexA, queueScore, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
    redis.call("ZADD", keyItemIndexB, queueScore, queueID)
end

return 0
