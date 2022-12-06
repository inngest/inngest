--[[

Output:
  0: Successfully leased item
  1: Queue item not found
  2: Queue item has no lease
  3: Lease ID doesn't match (indicating someone else took the lease)

]]

local queueKey      = KEYS[1]
local queueIndexKey = KEYS[2]

local queueID         = ARGV[1]
local currentLeaseKey = ARGV[2]
local newLeaseKey     = ARGV[3]

-- $include(decode_ulid_time.lua)
-- $include(get_queue_item.lua)

-- Grab the current time from the new lease key.
local nextTime = decode_ulid_time(newLeaseKey)

-- Look up the current queue item.  We need to see if the queue item is already leased.
local item = get_queue_item(queueKey, queueID)
if item == nil then
	return 1
end
if item.leaseID == nil then
	return 2
end
if item.leaseID ~= currentLeaseKey then
	return 3
end

item.leaseID = newLeaseKey
-- Update the item's lease key.
redis.call("HSET", queueKey, queueID, cjson.encode(item))
-- Update the item's score in our sorted index.
redis.call("ZADD", queueIndexKey, math.floor(nextTime / 1000), item.id)

return 0
