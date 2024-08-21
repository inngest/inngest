--[[

Re-enqueus a queue item within its queue, removing any lease.

Output:
  0: Successfully re-enqueued item
  1: Queue item not found

]]

local queueKey                = KEYS[1] -- queue:item - hash: { $itemID: $item }
local keyPartitionMap         = KEYS[2] -- partition:item - hash: { $workflowID: $partition }
local keyGlobalPointer        = KEYS[3] -- partition:sorted - zset
local keyPartitionA           = KEYS[4] -- queue:sorted:$workflowID - zset
local keyPartitionB           = KEYS[5] -- e.g. sorted:c|t:$workflowID - zset
local keyPartitionC           = KEYS[6] -- e.g. sorted:c|t:$workflowID - zset
-- We remove our queue item ID from each concurrency queue
local keyConcurrencyA    = KEYS[7] -- Account concurrency level
local keyConcurrencyB    = KEYS[8] -- When leasing an item we need to place the lease into this key
local keyConcurrencyC    = KEYS[9] -- Optional for eg. for concurrency amongst steps
local keyAcctConcurrency = KEYS[10]       
-- We push pointers to partition concurrency items to the partition concurrency item
local concurrencyPointer      = KEYS[11]
local keyItemIndexA           = KEYS[12]          -- custom item index 1
local keyItemIndexB           = KEYS[13]          -- custom item index 2

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
local legacyPartitionName = ARGV[11]

-- $include(get_queue_item.lua)
-- $include(get_partition_item.lua)
-- $include(update_pointer_score.lua)
-- $include(has_shard_key.lua)
-- $include(ends_with.lua)
-- $include(enqueue_to_partition.lua)

local item = get_queue_item(queueKey, queueID)
if item == nil then
    return 1
end

-- Update the queue item with a nil lease, at, atMS, etc.
redis.call("HSET", queueKey, queueID, queueItem)


-- This extends the item in the zset and also ensures that scavenger queues are
-- updated.
local function handleRequeue(keyConcurrency)
	redis.call("ZREM", keyConcurrency, item.id)

	-- Get the earliest item in the partition concurrency set.  We may be dequeueing
	-- the only in-progress job and should remove this from the partition concurrency
	-- pointers, if this exists.
	--
	-- This ensures that scavengeres have updated pointer queues without the currently
	-- leased job, if exists.
	local concurrencyScores = redis.call("ZRANGE", keyConcurrency, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
	if concurrencyScores == false then
		redis.call("ZREM", concurrencyPointer, keyConcurrency)
	else
		local earliestLease = tonumber(concurrencyScores[2])
		if earliestLease == nil then
			redis.call("ZREM", concurrencyPointer, keyConcurrency)
			redis.call("ZREM", concurrencyPointer, legacyPartitionName) -- remove previous item
		else
			-- Ensure that we update the score with the earliest lease
			redis.call("ZADD", concurrencyPointer, earliestLease, keyConcurrency)
			redis.call("ZREM", concurrencyPointer, legacyPartitionName) -- clean up previous item
		end
	end
end

--
-- Concurrency
--

handleRequeue(keyConcurrencyA)
handleRequeue(keyConcurrencyB)
handleRequeue(keyConcurrencyC)

-- Remove item from the account concurrency queue
-- This does not have a scavenger queue, as it's purely an entitlement limitation. See extendLease
-- and Lease for respective ZADD calls.
redis.call("ZREM", keyAcctConcurrency, item.id)

--
-- Partition manipulation
-- 
requeue_to_partition(keyPartitionA, partitionIdA, partitionItemA, keyPartitionMap, keyGlobalPointer, queueScore, queueID, nowMS)
requeue_to_partition(keyPartitionB, partitionIdB, partitionItemB, keyPartitionMap, keyGlobalPointer, queueScore, queueID, nowMS)
requeue_to_partition(keyPartitionC, partitionIdC, partitionItemC, keyPartitionMap, keyGlobalPointer, queueScore, queueID, nowMS)

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
    redis.call("ZADD", keyItemIndexA, queueScore, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
    redis.call("ZADD", keyItemIndexB, queueScore, queueID)
end

return 0
