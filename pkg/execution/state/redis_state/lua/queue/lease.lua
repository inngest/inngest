--[[

Output:
  0: Successfully leased item
  1: Queue item not found
  2: Queue item already leased
  3: No capacity

]]

local queueKey      = KEYS[1]
local queueIndexKey = KEYS[2]
local partitionKey  = KEYS[3]
-- We push our queue item ID into each concurrency queue
local accountConcurrencyKey   = KEYS[4] -- Account concurrency level
local partitionConcurrencyKey = KEYS[5] -- When leasing an item we need to place the lease into this key.
local customConcurrencyKey    = KEYS[6] -- Optional for eg. for concurrency amongst steps 
-- We push pointers to partition concurrency items to the partition concurrency item
local concurrencyPointer      = KEYS[7]

local queueID       = ARGV[1]
local newLeaseKey   = ARGV[2]
local currentTime   = tonumber(ARGV[3]) -- in ms
-- We check concurrency limits when leasing queue items: an account-level concurrency limit,
-- and a custom key.  The custom key is option.  It's used to add concurrency limits to individual
-- steps across differing functions
local accountConcurrency   = tonumber(ARGV[4])
local partitionConcurrency = tonumber(ARGV[5])
local customConcurrency    = tonumber(ARGV[6])
local partitionName        = ARGV[7]

-- Use our custom Go preprocessor to inject the file from ./includes/
-- $include(decode_ulid_time.lua)
-- $include(check_concurrency.lua)

-- Check the concurrency limits for the account and custom key;  partition keys are checked when
-- leasing the partition and do not need to be checked again (only one worker can run a partition at
-- once, and the capacity is kept in memory after leasing a partition)
if partitionConcurrency > 0 then
	if check_concurrency(currentTime, partitionConcurrencyKey, partitionConcurrency) <= 0 then
		return 3
	end
end
if accountConcurrency > 0 then
	if check_concurrency(currentTime, accountConcurrencyKey, accountConcurrency) <= 0 then
		return 3
	end
end
if customConcurrency > 0 then
	if check_concurrency(currentTime, customConcurrencyKey, customConcurrency) <= 0 then
		return 3
	end
end

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

-- Update the item's lease key.
item.leaseID = newLeaseKey
redis.call("HSET", queueKey, queueID, cjson.encode(item))

-- Add the item to all concurrency keys
redis.call("ZADD", partitionConcurrencyKey, nextTime, item.id)

-- NOTE: We check if concurrency > 0 here because this disables concurrency.  AccountID
-- and custom concurrency items may not be set, but the keys need to be set for clustered
-- mode.
if accountConcurrency > 0 then
	redis.call("ZADD", accountConcurrencyKey, nextTime, item.id)
end
if customConcurrency > 0 then
	redis.call("ZADD", customConcurrencyKey, nextTime, item.id)
end

-- Get the earliest item in the partition concurrency set.  If the current lease is
-- the only item in the set, we'll get the current lease.  Otherwise, we might get
-- a lost job or a previously lost job.
local concurrencyScores = redis.call("ZRANGE", partitionConcurrencyKey, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if concurrencyScores ~= false then
	local earliestLease = tonumber(concurrencyScores[2])
	-- Ensure that we update the score with the earliest lease, which may be a no-op.
	redis.call("ZADD", concurrencyPointer, earliestLease, partitionName)
end

-- Remove the item from our sorted index, as this is no longer on the queue.
redis.call("ZREM", queueIndexKey, item.id)

return 0
