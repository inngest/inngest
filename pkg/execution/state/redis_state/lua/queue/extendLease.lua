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
local keyPartitionScavengerIndex  = KEYS[7]

local queueID         = ARGV[1]
local currentLeaseKey = ARGV[2]
local newLeaseKey     = ARGV[3]

local partitionID 		      = ARGV[4]
local updateConstraintState = tonumber(ARGV[5])

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
-- TODO: Remove check on keyInProgressPartition once all new executors have rolled out and no more old items are in progress
local concurrencyScores =
	redis.call("ZRANGE", keyConcurrencyFn, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
local scavengerIndexScores =
	redis.call("ZRANGE", keyPartitionScavengerIndex, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if scavengerIndexScores ~= false or concurrencyScores ~= false then
	-- Either scavenger index or partition in progress set includes more items

	local earliestLease = nil
	if scavengerIndexScores ~= false and scavengerIndexScores ~= nil then
		earliestLease = tonumber(scavengerIndexScores[2])
	end

	-- Fall back to in progress set
	-- TODO: Remove this check once all items are tracked in scavenger index
	if
		earliestLease == nil
		or (concurrencyScores ~= false and concurrencyScores ~= nil and tonumber(concurrencyScores[2]) < earliestLease)
	then
		earliestLease = tonumber(concurrencyScores[2])
	end

	if earliestLease ~= nil then
		-- Ensure that we update the score with the earliest lease
		redis.call("ZADD", keyConcurrencyPointer, earliestLease, partitionID)
	end
end

if updateConstraintState == 1 then
  -- This extends the item in the zset and also ensures that scavenger queues are
  -- updated.
  local function handleExtendLease(keyConcurrency)
    redis.call("ZADD", keyConcurrency, nextTime, item.id)
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
end

return 0
