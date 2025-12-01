--[[

Output:
  0: Successfully leased item
  -1: Queue item not found
  -2: Queue item already leased

  -3: First partition concurrency limit hit
  -4: Second partition concurrency limit hit
  -5: Third partition concurrency limit hit

  -6: Account concurrency limit hit

  -7: Rate limited via throttling;  no capacity.
]]

local keyQueueMap = KEYS[1]
local concurrencyPointer = KEYS[2]

local keyReadyQueue = KEYS[3] -- queue:sorted:$workflowID - zset

-- In progress ZSETs for concurrency accounting
local keyInProgressAccount = KEYS[4]
local keyInProgressPartition = KEYS[5]
local keyInProgressCustomConcurrencyKey1 = KEYS[6]
local keyInProgressCustomConcurrencyKey2 = KEYS[7]

-- Active sets for constraint capacity accounting
local keyActiveAccount = KEYS[8]
local keyActivePartition = KEYS[9]
local keyActiveConcurrencyKey1 = KEYS[10]
local keyActiveConcurrencyKey2 = KEYS[11]
local keyActiveCompound = KEYS[12]

local keyActiveRun = KEYS[13]
local keyActiveRunsAccount = KEYS[14]
local keyActiveRunsPartition = KEYS[15]
local keyActiveRunsCustomConcurrencyKey1 = KEYS[16]
local keyActiveRunsCustomConcurrencyKey2 = KEYS[17]

local throttleKey = KEYS[18]

local keyInProgressLeasesAcct = KEYS[19]
local keyInProgressLeasesFn = KEYS[20]
local keyInProgressLeasesCustom1 = KEYS[21]
local keyInProgressLeasesCustom2 = KEYS[22]
local keyConstraintCheckIdempotency = KEYS[23]

local keyPartitionScavengerIndex = KEYS[24]

local queueID = ARGV[1]
local partitionID = ARGV[2]
local accountId = ARGV[3]
local runID = ARGV[4]
local newLeaseID = ARGV[5]

local currentTime = tonumber(ARGV[6]) -- in ms

-- We check concurrency limits when leasing queue items.
local concurrencyAcct = tonumber(ARGV[7])
local concurrencyPartition = tonumber(ARGV[8])
local customConcurrencyKey1 = tonumber(ARGV[9])
local customConcurrencyKey2 = tonumber(ARGV[10])
local marshaledConstraints = ARGV[11]

-- key queues v2
local refilledFromBacklog = tonumber(ARGV[12])

local checkConstraints = tonumber(ARGV[13])

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

-- NOTE: we can probably skip this entire section if item comes from backlog?
if checkConstraints == 1 then
	local constraints = cjson.decode(marshaledConstraints)

	-- Track throttling/rate limiting IF the queue item has throttling info set.  This allows
	-- us to target specific queue items with rate limiting individually.
	--
	-- We handle this before concurrency as it's typically not used, and it's faster to handle than concurrency,
	-- with o(1) operations vs o(log(n)).
	local itemHasThrottle = item.data ~= nil and item.data.throttle ~= nil
	local throttleConstraintExists = constraints.t ~= nil and constraints.t.p > 0
	local checkThrottle = itemHasThrottle and throttleConstraintExists
	-- Disable GCRA enforcement for refilled items as we already checked and updated throttle state during refill
	if refilledFromBacklog ~= 0 then
		checkThrottle = false
	-- Handle fallback for Constraint API (Acquire script succeeded, request failed transiently)
	elseif
		exists_without_ending(keyConstraintCheckIdempotency, ":-")
		and redis.call("EXISTS", keyConstraintCheckIdempotency) == 1
	then
		checkThrottle = false
	end

	if checkThrottle then
		local throttleResult = gcra(throttleKey, currentTime, constraints.t.p * 1000, constraints.t.l, constraints.t.b)
		if throttleResult == false then
			return -7
		end
	end

	-- Check the concurrency limits for the account and custom key;  partition keys are checked when
	-- leasing the partition and do not need to be checked again (only one worker can run a partition at
	-- once, and the capacity is kept in memory after leasing a partition)
	if customConcurrencyKey1 > 0 then
		local customCap = check_concurrency(currentTime, keyInProgressCustomConcurrencyKey1, customConcurrencyKey1)
		if exists_without_ending(keyInProgressLeasesCustom1, ":-") then
			customCap = customCap - count_concurrency(keyInProgressLeasesCustom1, currentTime)
		end
		if customCap <= 0 then
			return -4
		end
	end
	if customConcurrencyKey2 > 0 then
		local customCap = check_concurrency(currentTime, keyInProgressCustomConcurrencyKey2, customConcurrencyKey2)
		if exists_without_ending(keyInProgressLeasesCustom2, ":-") then
			customCap = customCap - count_concurrency(keyInProgressLeasesCustom2, currentTime)
		end
		if customCap <= 0 then
			return -5
		end
	end
	if concurrencyPartition > 0 then
		local partCap = check_concurrency(currentTime, keyInProgressPartition, concurrencyPartition)
		if exists_without_ending(keyInProgressLeasesFn, ":-") then
			partCap = partCap - count_concurrency(keyInProgressLeasesFn, currentTime)
		end
		if partCap <= 0 then
			return -3
		end
	end
	if concurrencyAcct > 0 then
		local accountCap = check_concurrency(currentTime, keyInProgressAccount, concurrencyAcct)
		if exists_without_ending(keyInProgressLeasesAcct, ":-") then
			accountCap = accountCap - count_concurrency(keyInProgressLeasesAcct, currentTime)
		end
		if accountCap <= 0 then
			return -6
		end
	end
end

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
-- TODO: Remove check on keyInProgressPartition once all new executors have rolled out and no more old items are in progress
local concurrencyScores = redis.call("ZRANGE", keyInProgressPartition, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
local scavengerIndexScores = redis.call("ZRANGE", keyPartitionScavengerIndex, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if scavengerIndexScores ~= false or concurrencyScores ~= false then
  -- Either scavenger index or partition in progress set includes more items

  local earliestLease = nil
  if scavengerIndexScores ~= false and scavengerIndexScores ~= nil then
    earliestLease = tonumber(scavengerIndexScores[2])
  end

  -- Fall back to in progress set
  -- TODO: Remove this check once all items are tracked in scavenger index
  if earliestLease == nil or (concurrencyScores ~= false and concurrencyScores ~= nil and tonumber(concurrencyScores[2]) < earliestLease) then
    earliestLease = tonumber(concurrencyScores[2])
  end

  if earliestLease ~= nil then
    -- Ensure that we update the score with the earliest lease
    redis.call("ZADD", concurrencyPointer, earliestLease, partitionID)
  end
end

-- Update in progress items sets / concurrency constraint state if we checked them
if checkConstraints == 1 then
	local function handleLease(keyConcurrency, concurrencyLimit)
		if concurrencyLimit > 0 then
			-- Add item to in-progress/concurrency queue and set score to lease expiry time to be picked up by scavenger
			redis.call("ZADD", keyConcurrency, nextTime, item.id)
		end
	end

	if exists_without_ending(keyInProgressAccount, ":-") then
		-- Always add this to acct level concurrency queues
		redis.call("ZADD", keyInProgressAccount, nextTime, item.id)
	end

	-- Always add this to fn level concurrency queues for scavenging
	redis.call("ZADD", keyInProgressPartition, nextTime, item.id)

	if exists_without_ending(keyInProgressCustomConcurrencyKey1, ":-") == true then
		handleLease(keyInProgressCustomConcurrencyKey1, customConcurrencyKey1)
	end

	if exists_without_ending(keyInProgressCustomConcurrencyKey2, ":-") == true then
		handleLease(keyInProgressCustomConcurrencyKey2, customConcurrencyKey2)
	end

	-- Update active sets for BacklogRefill
	addToActiveSets(
		keyActivePartition,
		keyActiveAccount,
		keyActiveCompound,
		keyActiveConcurrencyKey1,
		keyActiveConcurrencyKey2,
		{ item.id }
	)
	addToActiveRunSets(
		keyActiveRun,
		keyActiveRunsPartition,
		keyActiveRunsAccount,
		keyActiveRunsCustomConcurrencyKey1,
		keyActiveRunsCustomConcurrencyKey2,
		runID,
		item.id
	)
end

return 0
