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
local concurrencyPointer      = KEYS[2]

local keyReadyQueue           = KEYS[3]           -- queue:sorted:$workflowID - zset

-- In progress ZSETs for concurrency accounting
local keyInProgressAccount               = KEYS[4]
local keyInProgressPartition  				   = KEYS[5]
local keyInProgressCustomConcurrencyKey1 = KEYS[6]
local keyInProgressCustomConcurrencyKey2 = KEYS[7]

-- Active counters for constraint capacity accounting
local keyActiveAccount             = KEYS[8]
local keyActivePartition           = KEYS[9]
local keyActiveConcurrencyKey1     = KEYS[10]
local keyActiveConcurrencyKey2     = KEYS[11]
local keyActiveCompound            = KEYS[12]
local keyActiveRun                 = KEYS[13]
local keyIndexActivePartitionRuns  = KEYS[14]

local throttleKey             = KEYS[15]

local queueID      						= ARGV[1]
local partitionID 					  = ARGV[2]
local accountId       				= ARGV[3]
local runID                   = ARGV[4]
local newLeaseID  						= ARGV[5]

local currentTime  						= tonumber(ARGV[6]) -- in ms

-- We check concurrency limits when leasing queue items.
local concurrencyAcct 				= tonumber(ARGV[7])
local concurrencyPartition    = tonumber(ARGV[8])
local customConcurrencyKey1   = tonumber(ARGV[9])
local customConcurrencyKey2   = tonumber(ARGV[10])

-- key queues v2
local checkConstraints    = tonumber(ARGV[11])
local refilledFromBacklog = tonumber(ARGV[12])

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
local nextTime = decode_ulid_time(newLeaseID)
-- check if the item is leased.
if item.leaseID ~= nil and item.leaseID ~= cjson.null and decode_ulid_time(item.leaseID) > currentTime then
    -- This is already leased;  don't let this requester lease the item.
    return -2
end

-- Track the earliest time this job was attempted in the queue.
item = set_item_peek_time(keyQueueMap, queueID, item, currentTime)

if checkConstraints then
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
      if check_concurrency(currentTime, keyInProgressCustomConcurrencyKey1, customConcurrencyKey1) <= 0 then
          return -4
      end
  end
  if customConcurrencyKey2 > 0 then
      if check_concurrency(currentTime, keyInProgressCustomConcurrencyKey2, customConcurrencyKey2) <= 0 then
          return -5
      end
  end
  if concurrencyPartition > 0 then
      if check_concurrency(currentTime, keyInProgressPartition, concurrencyPartition) <= 0 then
          return -3
      end
  end
  if concurrencyAcct > 0 then
      if check_concurrency(currentTime, keyInProgressAccount, concurrencyAcct) <= 0 then
          return -6
      end
  end
end

-- Update the item's lease key.
item.leaseID = newLeaseID
redis.call("HSET", keyQueueMap, queueID, cjson.encode(item))

local function handleLease(keyConcurrency, concurrencyLimit)
	if concurrencyLimit > 0 then
		-- Add item to in-progress/concurrency queue and set score to lease expiry time to be picked up by scavenger
		redis.call("ZADD", keyConcurrency, nextTime, item.id)
	end
end

-- Remove the item from our sorted index, as this is no longer on the queue; it's in-progress
-- and stored in functionConcurrencyKey.
redis.call("ZREM", keyReadyQueue, item.id)

if exists_without_ending(keyInProgressAccount, ":-") then
  -- Always add this to acct level concurrency queues
  redis.call("ZADD", keyInProgressAccount, nextTime, item.id)
end

-- Always add this to fn level concurrency queues for scavenging
redis.call("ZADD", keyInProgressPartition, nextTime, item.id)

-- For every queue that we lease from, ensure that it exists in the scavenger pointer queue
-- so that expired leases can be re-processed.  We want to take the earliest time from the
-- concurrency queue such that we get a previously lost job if possible.
local inProgressScores = redis.call("ZRANGE", keyInProgressPartition, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if inProgressScores ~= false then
  local earliestLease = tonumber(inProgressScores[2])
  -- Add the earliest time to the pointer queue for in-progress, allowing us to scavenge
  -- lost jobs easily.
  -- Note: Previously, we stored the queue name in the zset, so we have to add an extra
  -- check to the scavenger logic to handle partition uuids for old queue items

  redis.call("ZADD", concurrencyPointer, earliestLease, partitionID)
end

if exists_without_ending(keyInProgressCustomConcurrencyKey1, ":-") == true then
  handleLease(keyInProgressCustomConcurrencyKey1, customConcurrencyKey1)
end

if exists_without_ending(keyInProgressCustomConcurrencyKey2, ":-") == true then
  handleLease(keyInProgressCustomConcurrencyKey2, customConcurrencyKey2)
end

-- If item was not refilled from backlog, we must increase active counters during lease
-- to account used capacity for future backlog refills
if refilledFromBacklog ~= 1 then
  -- Increase active counters by 1
  redis.call("INCR", keyActivePartition)

  if exists_without_ending(keyActiveAccount, ":-") then
    redis.call("INCR", keyActiveAccount)
  end

  if exists_without_ending(keyActiveCompound, ":-") then
    redis.call("INCR", keyActiveCompound)
  end

  if exists_without_ending(keyActiveConcurrencyKey1, ":-") then
    redis.call("INCR", keyActiveConcurrencyKey1)
  end

  if exists_without_ending(keyActiveConcurrencyKey2, ":-") then
    redis.call("INCR", keyActiveConcurrencyKey2)
  end

  if exists_without_ending(keyActiveRun, ":-") then
    -- increase number of active items in the run
    redis.call("INCR", keyActiveRun)

    -- update set of active function runs
    if exists_without_ending(keyIndexActivePartitionRuns, ":-") then
      redis.call("SADD", keyIndexActivePartitionRuns, runID)
    end
  end
end

return 0
