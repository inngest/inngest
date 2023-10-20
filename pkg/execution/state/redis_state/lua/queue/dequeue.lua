--[[

Output:
  0: Successfully dequeued item
  1: Queue item not found

]]

local queueKey       = KEYS[1]
local queueIndexKey  = KEYS[2]
local partitionKey   = KEYS[3]
local idempotencyKey = KEYS[4]
-- We must dequeue our queue item ID from each concurrency queue
local accountConcurrencyKey   = KEYS[5] -- Account concurrency level
local partitionConcurrencyKey = KEYS[6] -- Partition (function) concurrency level
local customConcurrencyKeyA   = KEYS[7] -- Optional for eg. for concurrency amongst steps 
local customConcurrencyKeyB   = KEYS[8] -- Optional for eg. for concurrency amongst steps 
-- We push pointers to partition concurrency items to the partition concurrency item
local concurrencyPointer      = KEYS[9]

local queueID        = ARGV[1]
local idempotencyTTL = tonumber(ARGV[2])
local partitionName  = ARGV[3]

-- $include(get_queue_item.lua)
-- Fetch this item to see if it was in progress prior to deleting.
local item = get_queue_item(queueKey, queueID)
if item == nil then
	return 1
end

redis.call("HDEL", queueKey, queueID)
redis.call("ZREM", queueIndexKey, queueID)
redis.call("HINCRBY", partitionKey, "len", -1) -- len of enqueued items decreases

if idempotencyTTL > 0 then
	redis.call("SETEX", idempotencyKey, idempotencyTTL, "")
end

if item.leaseID ~= nil and item.leaseID ~= cjson.null then
	-- Remove total number in progress, if there's a lease.
	redis.call("HINCRBY", partitionKey, "n", -1)
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

return 0
