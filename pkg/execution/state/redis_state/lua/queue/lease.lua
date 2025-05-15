--[[

Output:
  positive number: discrepancy in active counter

  0: Successfully leased item
  -1: Queue item not found
  -2: Queue item already leased

  -3: First partition concurrency limit hit
  -4: Second partition concurrency limit hit
  -5: Third partition concurrency limit hit

  -6: Account concurrency limit hit

  -7: Rate limited via throttling;  no capacity.
]]

local keyQueueMap            	= KEYS[1]
local keyPartitionFn          = KEYS[2]           -- queue:sorted:$workflowID - zset

-- We push our queue item ID into each concurrency queue
local keyConcurrencyFn  				= KEYS[3] -- Account concurrency level
local keyCustomConcurrencyKey1  = KEYS[4] -- When leasing an item we need to place the lease into this key.
local keyCustomConcurrencyKey2  = KEYS[5] -- Optional for eg. for concurrency amongst steps
-- We push pointers to partition concurrency items to the partition concurrency item
local concurrencyPointer      = KEYS[6]
local keyGlobalPointer        = KEYS[7]
local keyGlobalAccountPointer = KEYS[8] -- accounts:sorted - zset
local keyAccountPartitions    = KEYS[9] -- accounts:$accountId:partition:sorted - zset
local throttleKey             = KEYS[10] -- key used for throttling function run starts.
local keyAcctConcurrency      = KEYS[11]
local keyActiveCounter        = KEYS[12]

local queueID      						= ARGV[1]
local newLeaseKey  						= ARGV[2]
local currentTime  						= tonumber(ARGV[3]) -- in ms
local partitionID 					  = ARGV[4]
-- We check concurrency limits when leasing queue items.
local concurrencyFn    				= tonumber(ARGV[5])
local customConcurrencyKey1   = tonumber(ARGV[6])
local customConcurrencyKey2   = tonumber(ARGV[7])
-- And we always check against account concurrency limits
local concurrencyAcct 				= tonumber(ARGV[8])
local accountId       				= ARGV[9]

-- key queues v2
local disableLeaseChecks = tonumber(ARGV[10])

-- Use our custom Go preprocessor to inject the file from ./includes/
-- $include(decode_ulid_time.lua)
-- $include(check_concurrency.lua)
-- $include(get_queue_item.lua)
-- $include(set_item_peek_time.lua)
-- $include(update_pointer_score.lua)
-- $include(gcra.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)

-- first, get the queue item.  we must do this and bail early if the queue item
-- was not found.
local item = get_queue_item(keyQueueMap, queueID)
if item == nil then
    return -1
end

-- Grab the current time from the new lease key.
local nextTime = decode_ulid_time(newLeaseKey)
-- check if the item is leased.
if item.leaseID ~= nil and item.leaseID ~= cjson.null and decode_ulid_time(item.leaseID) > currentTime then
    -- This is already leased;  don't let this requester lease the item.
    return -2
end

-- Track the earliest time this job was attempted in the queue.
item = set_item_peek_time(keyQueueMap, queueID, item, currentTime)

if disableLeaseChecks ~= 1 then
	-- Track throttling/rate limiting IF the queue item has throttling info set.  This allows
	-- us to target specific queue items with rate limiting individually.
	--
	-- We handle this before concurrency as it's typically not used, and it's faster to handle than concurrency,
	-- with o(1) operations vs o(log(n)).
	if item.data ~= nil and item.data.throttle ~= nil then
		local throttleResult = gcra(throttleKey, currentTime, item.data.throttle.p * 1000, item.data.throttle.l, item.data.throttle.b)
		if throttleResult == false then
			return -7
		end
	end

  -- Check the concurrency limits for the account and custom key;  partition keys are checked when
  -- leasing the partition and do not need to be checked again (only one worker can run a partition at
  -- once, and the capacity is kept in memory after leasing a partition)
  if customConcurrencyKey1 > 0 then
      if check_concurrency(currentTime, keyCustomConcurrencyKey1, customConcurrencyKey1) <= 0 then
          return -4
      end
  end
  if customConcurrencyKey2 > 0 then
      if check_concurrency(currentTime, keyCustomConcurrencyKey2, customConcurrencyKey2) <= 0 then
          return -5
      end
  end
  if concurrencyFn > 0 then
      if check_concurrency(currentTime, keyConcurrencyFn, concurrencyFn) <= 0 then
          return -3
      end
  end
  if concurrencyAcct > 0 then
      if check_concurrency(currentTime, keyAcctConcurrency, concurrencyAcct) <= 0 then
          return -6
      end
  end
end

-- Update the item's lease key.
item.leaseID = newLeaseKey
redis.call("HSET", keyQueueMap, queueID, cjson.encode(item))

local function handleLease(keyConcurrency, concurrencyLimit)
	if concurrencyLimit > 0 then
		-- Add item to in-progress/concurrency queue and set score to lease expiry time to be picked up by scavenger
		redis.call("ZADD", keyConcurrency, nextTime, item.id)
	end
end

-- Remove the item from our sorted index, as this is no longer on the queue; it's in-progress
-- and stored in functionConcurrencyKey.
redis.call("ZREM", keyPartitionFn, item.id)

-- Always add this to acct level concurrency queues
redis.call("ZADD", keyAcctConcurrency, nextTime, item.id)

-- Always add this to fn level concurrency queues for scavenging
redis.call("ZADD", keyConcurrencyFn, nextTime, item.id)

-- For every queue that we lease from, ensure that it exists in the scavenger pointer queue
-- so that expired leases can be re-processed.  We want to take the earliest time from the
-- concurrency queue such that we get a previously lost job if possible.
local inProgressScores = redis.call("ZRANGE", keyConcurrencyFn, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if inProgressScores ~= false then
  local earliestLease = tonumber(inProgressScores[2])
  -- Add the earliest time to the pointer queue for in-progress, allowing us to scavenge
  -- lost jobs easily.
  -- Note: Previously, we stored the queue name in the zset, so we have to add an extra
  -- check to the scavenger logic to handle partition uuids for old queue items

  redis.call("ZADD", concurrencyPointer, earliestLease, partitionID)
end

if exists_without_ending(keyCustomConcurrencyKey1, ":-") == true then
  handleLease(keyCustomConcurrencyKey1, customConcurrencyKey1)
end

if exists_without_ending(keyCustomConcurrencyKey2, ":-") == true then
  handleLease(keyCustomConcurrencyKey2, customConcurrencyKey2)
end

-- Clean up active counter to reset it in a subsequent release
redis.call("DEL", keyActiveCounter)

return 0
