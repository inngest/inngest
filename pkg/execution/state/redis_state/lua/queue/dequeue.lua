--[[

Output:
  0: Successfully dequeued item
  1: Queue item not found

]]

local keyQueueMap    = KEYS[1]
-- remove items from all outsanding queues it may be in
local keyPartitionA  = KEYS[2]  -- queue:sorted:$workflowID - zset
local keyPartitionB  = KEYS[3]  -- e.g. sorted:c|t:$workflowID - zset
local keyPartitionC  = KEYS[4]  -- e.g. sorted:c|t:$workflowID - zset

local keyConcurrencyA    = KEYS[5] -- Account concurrency level
local keyConcurrencyB    = KEYS[6] -- When leasing an item we need to place the lease into this key.
local keyConcurrencyC    = KEYS[7] -- Optional for eg. for concurrency amongst steps
local keyAcctConcurrency = KEYS[8]       
local keyIdempotency     = KEYS[9]
local concurrencyPointer = KEYS[10]
local keyItemIndexA      = KEYS[11]   -- custom item index 1
local keyItemIndexB      = KEYS[12]  -- custom item index 2

local queueID        = ARGV[1]
local idempotencyTTL = tonumber(ARGV[2])
local partitionName  = ARGV[3]

-- $include(get_queue_item.lua)
-- Fetch this item to see if it was in progress prior to deleting.
local item = get_queue_item(keyQueueMap, queueID)
if item == nil then
	return 1
end

redis.call("HDEL", keyQueueMap, queueID)
redis.call("ZREM", keyPartitionA, queueID)
redis.call("ZREM", keyPartitionB, queueID)
redis.call("ZREM", keyPartitionC, queueID)

if idempotencyTTL > 0 then
	redis.call("SETEX", keyIdempotency, idempotencyTTL, "")
end

-- This extends the item in the zset and also ensures that scavenger queues are
-- updated.
local function handleDequeue(keyConcurrency)
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
		else
			-- Ensure that we update the score with the earliest lease
			redis.call("ZADD", concurrencyPointer, earliestLease, keyConcurrency)
		end
	end
end

handleDequeue(keyConcurrencyA)
handleDequeue(keyConcurrencyB)
handleDequeue(keyConcurrencyC)
-- This does not have a scavenger queue, as it's purely an entitlement limitation. See extendLease
-- and Lease for respective ZADD calls.
redis.call("ZREM", keyAcctConcurrency, item.id)


-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
	redis.call("ZREM", keyItemIndexA, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
	redis.call("ZREM", keyItemIndexB, queueID)
end

return 0
