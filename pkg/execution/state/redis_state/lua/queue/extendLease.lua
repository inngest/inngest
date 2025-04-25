--[[

Output:
  0: Successfully leased item
  1: Queue item not found
  2: Queue item has no lease
  3: Lease ID doesn't match (indicating someone else took the lease)

]]

local keyQueueMap       = KEYS[1] -- queue:item - hash: { $itemID: item }

local keyConcurrencyFn            = KEYS[2]
local keyCustomConcurrencyKey1    = KEYS[3] -- When leasing an item we need to place the lease into this key
local keyCustomConcurrencyKey2    = KEYS[4] -- Optional for eg. for concurrency amongst steps
local keyAcctConcurrency          = KEYS[5] -- Account concurrency level

local keyConcurrencyPointer       = KEYS[6]

local queueID         = ARGV[1]
local currentLeaseKey = ARGV[2]
local newLeaseKey     = ARGV[3]

local partitionID 		= ARGV[4]

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


-- This extends the item in the zset and also ensures that scavenger queues are
-- updated.
local function handleExtendLease(keyConcurrency)
	redis.call("ZADD", keyConcurrency, nextTime, item.id)
end

-- For every queue that we lease from, ensure that it exists in the scavenger pointer queue
-- so that expired leases can be re-processed.  We want to take the earliest time from the
-- concurrency queue such that we get a previously lost job if possible.

local inProgressScores = redis.call("ZRANGE", keyConcurrencyFn, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if inProgressScores ~= false then
  local earliestLease = tonumber(inProgressScores[2])
  -- Add the earliest time to the pointer queue for in-progress, allowing us to scavenge
  -- lost jobs easily.
  redis.call("ZADD", keyConcurrencyPointer, earliestLease, partitionID)
end

-- Items always belong to an account
redis.call("ZADD", keyAcctConcurrency, nextTime, item.id)

handleExtendLease(keyConcurrencyFn)

if exists_without_ending(keyCustomConcurrencyKey1, ":-") == true then
	handleExtendLease(keyCustomConcurrencyKey1)
end

if exists_without_ending(keyCustomConcurrencyKey2, ":-") == true then
	handleExtendLease(keyCustomConcurrencyKey2)
end

return 0
