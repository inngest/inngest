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
local keyItemIndexA           = KEYS[14]          -- custom item index 1
local keyItemIndexB           = KEYS[15]          -- custom item index 2

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

-- Remove item from the account concurrency queue
-- This does not have a scavenger queue, as it's purely an entitlement limitation. See extendLease
-- and Lease for respective ZADD calls.
redis.call("ZREM", keyAcctConcurrency, item.id)

--
-- Enqueue item to partition queues again
-- 
requeue_to_partition(keyPartitionA, partitionIdA, partitionItemA, partitionTypeA, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, nowMS, accountId)
requeue_to_partition(keyPartitionB, partitionIdB, partitionItemB, partitionTypeB, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, nowMS, accountId)
requeue_to_partition(keyPartitionC, partitionIdC, partitionItemC, partitionTypeC, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, nowMS, accountId)

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
    redis.call("ZADD", keyItemIndexA, queueScore, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
    redis.call("ZADD", keyItemIndexB, queueScore, queueID)
end

return 0
