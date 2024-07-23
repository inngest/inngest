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

local idempotencyKey = KEYS[5]
-- We must dequeue our queue item ID from each concurrency queue
local accountConcurrencyKey   = KEYS[6] -- Account concurrency level
local partitionConcurrencyKey = KEYS[7] -- Partition (function) concurrency level
local customConcurrencyKeyA   = KEYS[8] -- Optional for eg. for concurrency amongst steps 
local customConcurrencyKeyB   = KEYS[9] -- Optional for eg. for concurrency amongst steps 
-- We push pointers to partition concurrency items to the partition concurrency item
local concurrencyPointer      = KEYS[10]
local keyItemIndexA           = KEYS[11]  -- custom item index 1
local keyItemIndexB           = KEYS[12]  -- custom item index 2

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
	redis.call("SETEX", idempotencyKey, idempotencyTTL, "")
end

redis.call("ZREM", partitionConcurrencyKey, item.id)
if accountConcurrencyKey ~= nil and accountConcurrencyKey ~= "" then
	redis.call("ZREM", accountConcurrencyKey, item.id)
end
if customConcurrencyKeyA ~= nil and customConcurrencyKeyA ~= "" then
	redis.call("ZREM", customConcurrencyKeyA, item.id)
end
if customConcurrencyKeyB ~= nil and customConcurrencyKeyB ~= "" then
	redis.call("ZREM", customConcurrencyKeyB, item.id)
end

-- Get the earliest item in the partition concurrency set.  We may be requeueing
-- the only in-progress job and should remove this from the partition concurrency
-- pointers, if this exists.
local concurrencyScores = redis.call("ZRANGE", partitionConcurrencyKey, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if concurrencyScores == false then
	redis.call("ZREM", concurrencyPointer, partitionName)
else
	local earliestLease = tonumber(concurrencyScores[2])
	if earliestLease == nil then
		redis.call("ZREM", concurrencyPointer, partitionName)
	else
		-- Ensure that we update the score with the earliest lease
		redis.call("ZADD", concurrencyPointer, earliestLease, partitionName)
	end
end

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
	redis.call("ZREM", keyItemIndexA, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
	redis.call("ZREM", keyItemIndexB, queueID)
end

return 0
