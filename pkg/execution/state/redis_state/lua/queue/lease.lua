--[[

Output:
  0: Successfully leased item
  -1: Queue item not found
  -2: Queue item already leased
]]

local keyQueueMap = KEYS[1]
local concurrencyPointer = KEYS[2]

local keyReadyQueue = KEYS[3] -- queue:sorted:$workflowID - zset

local keyPartitionScavengerIndex = KEYS[4]

local queueID = ARGV[1]
local partitionID = ARGV[2]
local newLeaseID = ARGV[3]
local currentTime = tonumber(ARGV[4]) -- in ms

-- Use our custom Go preprocessor to inject the file from ./includes/
-- $include(decode_ulid_time.lua)
-- $include(check_concurrency.lua)
-- $include(get_queue_item.lua)
-- $include(set_item_peek_time.lua)
-- $include(update_pointer_score.lua)
-- $include(gcra.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(update_active_sets.lua)

-- first, get the queue item.  we must do this and bail early if the queue item
-- was not found.
local item = get_queue_item(keyQueueMap, queueID)
if item == nil then
	return -1
end

-- Grab the current time from the new lease key.
local nextTime = decode_ulid_time(newLeaseID)
-- check if the item is leased.
if item.leaseID ~= nil and item.leaseID ~= cjson.null and decode_ulid_time(item.leaseID) > currentTime then
	-- This is already leased;  don't let this requester lease the item.
	return -2
end

-- Track the earliest time this job was attempted in the queue.
item = set_item_peek_time(keyQueueMap, queueID, item, currentTime)

-- Update the item's lease key.
item.leaseID = newLeaseID
redis.call("HSET", keyQueueMap, queueID, cjson.encode(item))

-- Remove the item from our sorted index, as this is no longer on the queue; it's in-progress
-- and stored in functionConcurrencyKey.
redis.call("ZREM", keyReadyQueue, item.id)

-- Always add to partition scavenging index
redis.call("ZADD", keyPartitionScavengerIndex, nextTime, item.id)

-- For every queue that we lease from, ensure that it exists in the scavenger pointer queue
-- so that expired leases can be re-processed.  We want to take the earliest time from the
-- scavenger index such that we get a previously lost job if possible.
local scavengerIndexScores =
	redis.call("ZRANGE", keyPartitionScavengerIndex, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if scavengerIndexScores ~= false and scavengerIndexScores ~= nil then
	local earliestLease = tonumber(scavengerIndexScores[2])

	-- Ensure that we update the score with the earliest lease
	redis.call("ZADD", concurrencyPointer, earliestLease, partitionID)
end

return 0
