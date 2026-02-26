--[[

Output:
  0: Successfully leased item
  1: Queue item not found
  2: Queue item has no lease
  3: Lease ID doesn't match (indicating someone else took the lease)

]]

local keyQueueMap       = KEYS[1] -- queue:item - hash: { $itemID: item }

local keyConcurrencyPointer       = KEYS[2]
local keyPartitionScavengerIndex  = KEYS[3]

local queueID         = ARGV[1]
local currentLeaseKey = ARGV[2]
local newLeaseKey     = ARGV[3]

local partitionID 		      = ARGV[4]

-- $include(decode_ulid_time.lua)
-- $include(get_queue_item.lua)
-- $include(ends_with.lua)

-- Grab the current time from the new lease key.
local nextTime = decode_ulid_time(newLeaseKey)

-- Look up the current queue item.  We need to see if the queue item is already leased.
local item = get_queue_item(keyQueueMap, queueID)
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
redis.call("HSET", keyQueueMap, queueID, cjson.encode(item))
-- Update the item's score in our sorted index.

-- Update scavenger index
redis.call("ZADD", keyPartitionScavengerIndex, nextTime, item.id)

-- For every queue that we lease from, ensure that it exists in the scavenger pointer queue
-- so that expired leases can be re-processed.  We want to take the earliest time from the
-- scavenger index such that we get a previously lost job if possible.
local scavengerIndexScores =
	redis.call("ZRANGE", keyPartitionScavengerIndex, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if scavengerIndexScores ~= false and scavengerIndexScores ~= nil then
	local earliestLease = tonumber(scavengerIndexScores[2])

	-- Ensure that we update the score with the earliest lease
	redis.call("ZADD", keyConcurrencyPointer, earliestLease, partitionID)
end

return 0
