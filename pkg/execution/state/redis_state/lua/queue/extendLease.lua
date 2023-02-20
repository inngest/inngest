--[[

Output:
  0: Successfully leased item
  1: Queue item not found
  2: Queue item has no lease
  3: Lease ID doesn't match (indicating someone else took the lease)

]]

local queueKey      = KEYS[1]
-- We update the lease time in each concurrency queue, also
local accountConcurrencyKey   = KEYS[2] -- Account concurrency level
local partitionConcurrencyKey = KEYS[3] -- Partition/function level concurrency
local customConcurrencyKey    = KEYS[4] -- Optional for eg. for concurrency amongst steps 

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
if item.leaseID == nil or item.leaseID == cjson.null then
	return 2
end
if item.leaseID ~= currentLeaseKey then
	return 3
end

item.leaseID = newLeaseKey
-- Update the item's lease key.
redis.call("HSET", queueKey, queueID, cjson.encode(item))
-- Update the item's score in our sorted index.

-- Add the item to all keys
redis.call("ZADD", partitionConcurrencyKey, nextTime, item.id)
if accountConcurrencyKey ~= nil and accountConcurrencyKey ~= "" then
	redis.call("ZADD", accountConcurrencyKey, nextTime, item.id)
end
if customConcurrencyKey ~= nil and customConcurrencyKey ~= "" then
	redis.call("ZADD", customConcurrencyKey, nextTime, item.id)
end

return 0
