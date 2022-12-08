--[[

Output:
  0: Successfully leased item
  1: Queue item not found
  2: Queue item already leased

]]

local queueKey      = KEYS[1]
local queueIndexKey = KEYS[2]
local partitionKey  = KEYS[3]

local queueID       = ARGV[1]
local newLeaseKey   = ARGV[2]
local currentTime   = tonumber(ARGV[3]) -- in ms

-- Use our custom Go preprocessor to inject the file from ./includes/
-- $include(decode_ulid_time.lua)

-- Grab the current time from the new lease key.
local nextTime = decode_ulid_time(newLeaseKey)

-- Look up the current queue item.  We need to see if the queue item is already leased.
local encoded = redis.call("HGET", queueKey, queueID)
if encoded == false then
	return 1
end

local item = cjson.decode(encoded)
if item == nil then
	return 1
end

if item.leaseID ~= nil and item.leaseID ~= cjson.null and decode_ulid_time(item.leaseID) > currentTime then
	-- This is already leased;  don't let this requester lease the item.
	return 2
end

if item.leaseID == nil or item.leaseID == cjson.null then
	-- Increase the in-progress count by 1 as we've just leased an item.
	-- This lets us calculate the number of concurrent items when multiple shared-nothing
	-- workers are working on the same queue.
	-- 
	-- If the lease ID is not nil then we assume this has already been increaased and is
	-- a dead worker.
	redis.call("HINCRBY", partitionKey, "n", 1)
end


item.leaseID = newLeaseKey
-- Update the item's lease key.
redis.call("HSET", queueKey, queueID, cjson.encode(item))
-- Update the item's score in our sorted index.
redis.call("ZADD", queueIndexKey, math.floor(nextTime / 1000), item.id)

return 0
