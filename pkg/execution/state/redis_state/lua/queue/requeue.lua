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
local keyPartitionA           = KEYS[6] -- queue:sorted:$workflowID - zset
local keyPartitionB           = KEYS[7] -- e.g. sorted:c|t:$workflowID - zset
local keyPartitionC           = KEYS[8] -- e.g. sorted:c|t:$workflowID - zset
-- We remove our queue item ID from each concurrency queue
local keyConcurrencyA    = KEYS[9] -- Account concurrency level
local keyConcurrencyB    = KEYS[10] -- When leasing an item we need to place the lease into this key
local keyConcurrencyC    = KEYS[11] -- Optional for eg. for concurrency amongst steps
local keyAcctConcurrency = KEYS[12]
-- We push pointers to partition concurrency items to the partition concurrency item
local concurrencyPointer      = KEYS[13]

-- Key queues v2
local keyBacklogSetA              = KEYS[14]          -- backlog:sorted:<backlogID> - zset
local keyBacklogSetB              = KEYS[15]          -- backlog:sorted:<backlogID> - zset
local keyBacklogSetC              = KEYS[16]          -- backlog:sorted:<backlogID> - zset
local keyBacklogMeta              = KEYS[17]          -- backlogs - hash
local keyGlobalShadowPartitionSet = KEYS[18]          -- shadow:sorted
local keyShadowPartitionSet       = KEYS[19]          -- shadow:sorted:<fnID|queueName> - zset
local keyShadowPartitionMeta      = KEYS[20]          -- shadows
local keyInProgress               = KEYS[21]
local keyAccountInProgress        = KEYS[22]
local keyActiveJobsKey1           = KEYS[23]
local keyActiveJobsKey2           = KEYS[24]

local keyItemIndexA           = KEYS[25]          -- custom item index 1
local keyItemIndexB           = KEYS[26]          -- custom item index 2


local queueItem           = ARGV[1]
local queueID             = ARGV[2]           -- id
local queueScore          = tonumber(ARGV[3]) -- vesting time, in ms
local nowMS               = tonumber(ARGV[4]) -- now in ms
local partitionItemA      = ARGV[5]
local partitionItemB      = ARGV[6]
local partitionItemC      = ARGV[7]
local partitionIdA        = ARGV[8]
local partitionIdB        = ARGV[9]
local partitionIdC        = ARGV[10]
local accountId           = ARGV[11]
local legacyPartitionName = ARGV[12]
local partitionTypeA = tonumber(ARGV[13])
local partitionTypeB = tonumber(ARGV[14])
local partitionTypeC = tonumber(ARGV[15])

-- Key queues v2
local requeueToBacklog				= tonumber(ARGV[16])
local partitionID             = ARGV[17]
local shadowPartitionItem     = ARGV[18]
local backlogItemA            = ARGV[19]
local backlogItemB            = ARGV[20]
local backlogItemC            = ARGV[21]
local backlogIdA              = ARGV[22]
local backlogIdB              = ARGV[23]
local backlogIdC              = ARGV[24]

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
local function handleRequeueConcurrency(keyConcurrency, partitionID, partitionType)
	redis.call("ZREM", keyConcurrency, item.id) -- Remove from in-progress queue

	if partitionType ~= 0 then
		-- If this is not a default partition, we don't need to update the concurrency pointer (used by scavenger)
		return
	end

	-- Backwards compatibility: For default partitions, use the partition ID (function ID) as the pointer
	local pointerMember = keyConcurrency
	if partitionType == 0 then
		pointerMember = partitionID
	end

	-- Get the earliest item in the partition concurrency set.  We may be dequeueing
	-- the only in-progress job and should remove this from the partition concurrency
	-- pointers, if this exists.
	--
	-- This ensures that scavengeres have updated pointer queues without the currently
	-- leased job, if exists.
	local concurrencyScores = redis.call("ZRANGE", keyConcurrency, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
	if concurrencyScores == false then
		redis.call("ZREM", concurrencyPointer, pointerMember)
	else
		local earliestLease = tonumber(concurrencyScores[2])
		if earliestLease == nil then
			redis.call("ZREM", concurrencyPointer, pointerMember)
		else
			-- Ensure that we update the score with the earliest lease
			redis.call("ZADD", concurrencyPointer, earliestLease, pointerMember)
		end
	end
end

--
-- Concurrency
--

handleRequeueConcurrency(keyConcurrencyA, partitionIdA, partitionTypeA)
handleRequeueConcurrency(keyConcurrencyB, partitionIdB, partitionTypeB)
handleRequeueConcurrency(keyConcurrencyC, partitionIdC, partitionTypeC)

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

	-- Get the earliest item in the partition concurrency set.  We may be dequeueing
	-- the only in-progress job and should remove this from the partition concurrency
	-- pointers, if this exists.
	--
	-- This ensures that scavengeres have updated pointer queues without the currently
	-- leased job, if exists.
	local concurrencyScores = redis.call("ZRANGE", keyInProgress, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
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
  requeue_to_backlog(keyBacklogSetA, backlogIdA, backlogItemA, partitionID, shadowPartitionItem, partitionIdA, partitionItemA, partitionTypeA, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)
  requeue_to_backlog(keyBacklogSetB, backlogIdB, backlogItemB, partitionID, shadowPartitionItem, partitionIdA, partitionItemA, partitionTypeA, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)
  requeue_to_backlog(keyBacklogSetC, backlogIdC, backlogItemC, partitionID, shadowPartitionItem, partitionIdA, partitionItemA, partitionTypeA, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)

	requeue_to_backlog(keyBacklogSetA, backlogIdA, backlogItemA, partitionID, shadowPartitionItem, partitionIdB, partitionItemB, partitionTypeB, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)
  requeue_to_backlog(keyBacklogSetB, backlogIdB, backlogItemB, partitionID, shadowPartitionItem, partitionIdB, partitionItemB, partitionTypeB, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)
  requeue_to_backlog(keyBacklogSetC, backlogIdC, backlogItemC, partitionID, shadowPartitionItem, partitionIdB, partitionItemB, partitionTypeB, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)

	requeue_to_backlog(keyBacklogSetA, backlogIdA, backlogItemA, partitionID, shadowPartitionItem, partitionIdC, partitionItemC, partitionTypeC, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)
  requeue_to_backlog(keyBacklogSetB, backlogIdB, backlogItemB, partitionID, shadowPartitionItem, partitionIdC, partitionItemC, partitionTypeC, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)
  requeue_to_backlog(keyBacklogSetC, backlogIdC, backlogItemC, partitionID, shadowPartitionItem, partitionIdC, partitionItemC, partitionTypeC, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, queueScore, queueID, nowMS)

else
  --
  -- Enqueue item to partition queues again
  --
  requeue_to_partition(keyPartitionA, partitionIdA, partitionItemA, partitionTypeA, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, nowMS, accountId)
  requeue_to_partition(keyPartitionB, partitionIdB, partitionItemB, partitionTypeB, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, nowMS, accountId)
  requeue_to_partition(keyPartitionC, partitionIdC, partitionItemC, partitionTypeC, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, nowMS, accountId)
end

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
    redis.call("ZADD", keyItemIndexA, queueScore, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
    redis.call("ZADD", keyItemIndexB, queueScore, queueID)
end

return 0
