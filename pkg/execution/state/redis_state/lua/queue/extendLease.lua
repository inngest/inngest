--[[

Output:
  0: Successfully leased item
  1: Queue item not found
  2: Queue item has no lease
  3: Lease ID doesn't match (indicating someone else took the lease)

]]

local itemHashKey       = KEYS[1] -- queue:item - hash: { $itemID: item }
local itemQueueKey      = KEYS[2] -- queue:sorted:$workflowID - zset of queue items
local partitionIndexKey = KEYS[3] -- partition:sorted - zset of queues by earliest item
-- We update the lease time in each concurrency queue, also
local accountConcurrencyKey   = KEYS[4] -- Account concurrency level
local partitionConcurrencyKey = KEYS[5] -- Partition/function level concurrency
local customConcurrencyKeyA   = KEYS[6] -- Optional for eg. for concurrency amongst steps 
local customConcurrencyKeyB   = KEYS[7] -- Optional for eg. for concurrency amongst steps 

local queueID         = ARGV[1]
local currentLeaseKey = ARGV[2]
local newLeaseKey     = ARGV[3]
local partitionID     = ARGV[4]

-- $include(decode_ulid_time.lua)
-- $include(get_queue_item.lua)

-- Grab the current time from the new lease key.
local nextTime = decode_ulid_time(newLeaseKey)

-- Look up the current queue item.  We need to see if the queue item is already leased.
local item = get_queue_item(itemHashKey, queueID)
if item == nil then
	return 1
end
if item.leaseID == nil or item.leaseID == cjson.null then
	return 2
end
if item.leaseID ~= currentLeaseKey then
	return 3
end

item.leaseID = newLeaseKey
-- Update the item's lease key.
redis.call("HSET", itemHashKey, queueID, cjson.encode(item))
-- Update the item's score in our sorted index.

-- Add the item to all keys
redis.call("ZADD", partitionConcurrencyKey, nextTime, item.id)
if accountConcurrencyKey ~= nil and accountConcurrencyKey ~= "" then
	redis.call("ZADD", accountConcurrencyKey, nextTime, item.id)
end
if customConcurrencyKeyA ~= nil and customConcurrencyKeyA ~= "" then
	redis.call("ZADD", customConcurrencyKeyA, nextTime, item.id)
end
if customConcurrencyKeyB ~= nil and customConcurrencyKeyA ~= "" then
	redis.call("ZADD", customConcurrencyKeyB, nextTime, item.id)
end

-- If there's nothing in the queue of queues, extend the queue expiry time by the
-- lease time.  This lets us ensure that a partition exists and will NOT be garbage
-- collected while an item is worked on, which is necessary if the job fails.
-- 
-- Partitions are garbage collected during the peeking & leasing process, so
-- if the current partition is empty we want to prevent that.
if tonumber(redis.call("ZCARD", itemQueueKey)) == 0 then
	-- NOTE: this is math.ceil to minimize race conditions;  partitions are stored
	-- using seconds and we should round up to process this after the 
	--
	-- We also add 2 seconds because we peek queues in advance.
	redis.call("ZADD", partitionIndexKey, (math.ceil(nextTime) / 1000) + 2, partitionID)
end

return 0
