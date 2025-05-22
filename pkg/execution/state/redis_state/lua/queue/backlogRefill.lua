--[[

  Checks available capacity by validating concurrency and other queue capacity constraints.
  Moves as many queue items from backlog into ready queue as capacity.

  backlogRefill will always attempt to move queue items from backlogs into ready queues up to
  hitting concurrency.

  Returns a tuple of {
    status,               -- See status section below
    items_refilled,       -- Number of items refilled to ready queue
    items_until,          -- Number of items within provided time range in backlog before refilling
    items_total,          -- Total number of items in backlog before refilling
    constraintCapacity,   -- Most limiting constraint capacity
    refill                -- Number of items to refill (may include missing items)
  }

  Status values:

  0 - Did not hit constraint
  1 - Account concurrency limit reached
  2 - Function concurrency limit reached
  3 - Custom concurrency key 1 limit reached
  4 - Custom concurrency key 2 limit reached
  5 - Throttled
]]

local keyBacklogSet                      = KEYS[1]
local keyShadowPartitionSet              = KEYS[2]
local keyGlobalShadowPartitionSet        = KEYS[3]
local keyGlobalAccountShadowPartitionSet = KEYS[4]
local keyAccountShadowPartitionSet       = KEYS[5]

local keyReadySet                        = KEYS[6]
local keyGlobalPointer        	         = KEYS[7] -- partition:sorted - zset
local keyGlobalAccountPointer 	         = KEYS[8] -- accounts:sorted - zset
local keyAccountPartitions    	         = KEYS[9] -- accounts:$accountID:partition:sorted - zset

local keyQueueItemHash                   = KEYS[10]

-- Constraint-related accounting keys
local keyActiveAccount           = KEYS[11]
local keyActivePartition         = KEYS[12]
local keyActiveConcurrencyKey1   = KEYS[13]
local keyActiveConcurrencyKey2   = KEYS[14]
local keyActiveCompound          = KEYS[15]

local backlogID     = ARGV[1]
local partitionID   = ARGV[2]
local accountID     = ARGV[3]
local refillUntilMS = tonumber(ARGV[4])
local refillLimit   = tonumber(ARGV[5])
local nowMS         = tonumber(ARGV[6])

-- We check concurrency limits before refilling
local concurrencyAcct 				= tonumber(ARGV[7])
local concurrencyFn    				= tonumber(ARGV[8])
local customConcurrencyKey1   = tonumber(ARGV[9])
local customConcurrencyKey2   = tonumber(ARGV[10])

-- We check throttle before refilling
local throttleKey    = ARGV[11]
local throttleLimit  = tonumber(ARGV[12])
local throttleBurst  = tonumber(ARGV[13])
local throttlePeriod = tonumber(ARGV[14])

local keyPrefix = ARGV[15]

-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(gcra.lua)
-- $include(update_backlog_pointer.lua)

--
-- Retrieve current backlog size
--

local backlogCountTotal = redis.call("ZCARD", keyBacklogSet)
if backlogCountTotal == false or backlogCountTotal == nil then
  backlogCountTotal = 0
end

if backlogCountTotal == 0 then
  -- update backlog pointers
  updateBacklogPointer(keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, accountID, partitionID, backlogID)

  return { 0, 0, 0, backlogCountTotal, 0, 0 }
end

local backlogCountUntil = redis.call("ZCOUNT", keyBacklogSet, "-inf", refillUntilMS)
if backlogCountUntil == false or backlogCountUntil == nil then
  backlogCountUntil = 0
end

if backlogCountUntil == 0 then
  -- update backlog pointers
  updateBacklogPointer(keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, accountID, partitionID, backlogID)

  return { 0, 0, backlogCountUntil, backlogCountTotal, 0, 0 }
end

--
-- Calculate initial number of items to refill
--

-- Set items to refill to number of items in backlog
local refill = backlogCountUntil

-- Limit items to refill to max refill limit if more items are in backlog
if refill > refillLimit then
  refill = refillLimit
end

--
-- Check constraints and adjust capacity
--

-- Initialize capacity as nil, which represents no constraint limits
local constraintCapacity = nil

-- Set initial status to success, progressively add more specific capacity constraints
local status = 0

local function check_active_capacity(now_ms, keyActiveCounter, limit)
	local count = redis.call("GET", keyActiveCounter)
	if count ~= false and count ~= nil then
    return tonumber(limit) - tonumber(count)
  end

	return tonumber(limit)
end

-- Check throttle capacity
if (constraintCapacity == nil or constraintCapacity > 0) and throttleLimit > 0 then
  local remainingThrottleCapacity = gcraCapacity(throttleKey, nowMS, throttlePeriod * 1000, throttleLimit, throttleBurst)
  if constraintCapacity == nil or remainingThrottleCapacity < constraintCapacity then
    constraintCapacity = remainingThrottleCapacity
    status = 5
  end
end

-- Check custom concurrency key 2 capacity
if (constraintCapacity == nil or constraintCapacity > 0) and exists_without_ending(keyActiveConcurrencyKey2, ":-") == true and customConcurrencyKey2 > 0 then
  local remainingCustomConcurrencyCapacityKey2 = check_active_capacity(nowMS, keyActiveConcurrencyKey2, customConcurrencyKey2)
  if constraintCapacity == nil or remainingCustomConcurrencyCapacityKey2 < constraintCapacity then
    -- Custom concurrency key 2 imposes limits
    constraintCapacity = remainingCustomConcurrencyCapacityKey2
    status = 4
  end
end

-- Check custom concurrency key 1 capacity
if (constraintCapacity == nil or constraintCapacity > 0) and exists_without_ending(keyActiveConcurrencyKey1, ":-") == true and customConcurrencyKey1 > 0 then
  local remainingCustomConcurrencyCapacityKey1 = check_active_capacity(nowMS, keyActiveConcurrencyKey1, customConcurrencyKey1)
  if constraintCapacity == nil or remainingCustomConcurrencyCapacityKey1 < constraintCapacity then
    -- Custom concurrency key 1 imposes limits
    constraintCapacity = remainingCustomConcurrencyCapacityKey1
    status = 3
  end
end

-- Check function concurrency capacity
if (constraintCapacity == nil or constraintCapacity > 0) and exists_without_ending(keyActivePartition, ":-") == true and concurrencyFn > 0 then
  local remainingFunctionCapacity = check_active_capacity(nowMS, keyActivePartition, concurrencyFn)
  if constraintCapacity == nil or remainingFunctionCapacity < constraintCapacity then
    -- Function concurrency imposes limits
    constraintCapacity = remainingFunctionCapacity
    status = 2
  end
end

-- Check account concurrency capacity
if (constraintCapacity == nil or constraintCapacity > 0) and exists_without_ending(keyActiveAccount, ":-") == true and concurrencyAcct > 0 then
  local remainingAccountCapacity = check_active_capacity(nowMS, keyActiveAccount, concurrencyAcct)

  if constraintCapacity == nil or remainingAccountCapacity < constraintCapacity then
    -- Account concurrency imposes limits
    constraintCapacity = remainingAccountCapacity
    status = 1
  end
end

if constraintCapacity > 0 then
  -- Reset status as we're not limited
  status = 0
end

-- If we are constrained, reduce refill to max allowed capacity
if constraintCapacity < refill then
  -- Most limiting status will be kept
  refill = constraintCapacity
end

--
-- Refill to match capacity
--

local refilled = 0

-- Only attempt to refill if we have capacity
if refill > 0 then
  -- Move item(s) out of backlog and into partition

  -- Peek item IDs and scores
  local itemIDs = redis.call("ZRANGE", keyBacklogSet, "-inf", refillUntilMS, "BYSCORE", "LIMIT", 0, refill)
  local itemScores = redis.call("ZMSCORE", keyBacklogSet, unpack(itemIDs))

  -- Attempt to load item data
  local potentiallyMissingQueueItems = redis.call("HMGET", keyQueueItemHash, unpack(itemIDs))

  -- Reverse the items to be added to the ready set
  local readyArgs = {}

  local backlogRemArgs = {}
  local hasRemove = false

  local itemUpdateArgs = {}

  for i = 1, #itemIDs do
    local itemID = itemIDs[i]
    local itemScore = tonumber(itemScores[i])
    local itemData = potentiallyMissingQueueItems[i]

    -- If queue item does not exist in hash, delete from backlog
    if itemData == false or itemData == nil or itemData == "" then
      table.insert(backlogRemArgs, itemID)  -- remove from backlog
      hasRemove = true
    else
      -- Insert new members into ready set
      table.insert(readyArgs, itemScore)
      table.insert(readyArgs, itemID)

      -- Remove item from backlog
      table.insert(backlogRemArgs, itemID)
      hasRemove = true

      -- Update queue item with refill data
      local updatedData = cjson.decode(itemData)
      updatedData.rf = backlogID
      updatedData.rat = nowMS

      if updatedData.data ~= nil and updatedData.data.identifier ~= nil and updatedData.data.identifier.runID ~= nil then
        -- add item to active in run
        local runID = updatedData.data.identifier.runID
        local keyActiveRun = string.format("%s:active:run:%s", keyPrefix, runID)
        local updateTo = math.floor(itemScore / 1000)

        -- increase number of active items in run
        redis.call("INCR", keyActiveRun)

        -- if the newly-added item is earlier than existing items in the run, adjust pointer scores in the function
        -- see QueueKeyGenerator#ActivePartitionRunsIndex for reference
        local keyIndexActivePartitionRuns = string.format("%s:active-idx:runs:%s", keyPrefix, partitionID)

        redis.call("SADD", keyIndexActivePartitionRuns, runID)
      end

      table.insert(itemUpdateArgs, itemID)
      table.insert(itemUpdateArgs, cjson.encode(updatedData))

      -- Increment number of refilled items
      refilled = refilled + 1
    end
  end

  if refilled > 0 then
    -- "Refill" items to ready set
    redis.call("ZADD", keyReadySet, unpack(readyArgs))

    -- Increase active counters by number of refilled items
    redis.call("INCRBY", keyActivePartition, refilled)

    if exists_without_ending(keyActiveAccount, ":-") then
      redis.call("INCRBY", keyActiveAccount, refilled)
    end

    if exists_without_ending(keyActiveCompound, ":-") then
      redis.call("INCRBY", keyActiveCompound, refilled)
    end

    if exists_without_ending(keyActiveConcurrencyKey1, ":-") then
      redis.call("INCRBY", keyActiveConcurrencyKey1, refilled)
    end

    if exists_without_ending(keyActiveConcurrencyKey2, ":-") then
      redis.call("INCRBY", keyActiveConcurrencyKey2, refilled)
    end

    -- Update queue items with refill data
    redis.call("HSET", keyQueueItemHash, unpack(itemUpdateArgs))
  end

  if hasRemove then
    -- Remove refilled or missing items from backlog
    redis.call("ZREM", keyBacklogSet, unpack(backlogRemArgs))
  end
end

-- update gcra theoretical arrival time
if throttleLimit > 0 then
  gcraUpdate(throttleKey, nowMS, throttlePeriod * 1000, throttleLimit, throttleBurst, refill)
end

--
-- Adjust ready queue pointers
--

if refilled > 0 then
  -- Get the minimum score for the queue.
  local earliestScore = get_converted_earliest_pointer_score(keyReadySet)
  if earliestScore > 0 then
    -- Potentially update the queue of queues.
    local currentScore = redis.call("ZSCORE", keyGlobalPointer, partitionID)
    if currentScore == false or tonumber(currentScore) > earliestScore then
      update_pointer_score_to(partitionID, keyGlobalPointer, earliestScore)
      update_account_queues(keyGlobalAccountPointer, keyAccountPartitions, partitionID, accountID, earliestScore)
    end
  end
end

--
-- Adjust pointer scores for shadow scanning, potentially clean up
--

updateBacklogPointer(keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, accountID, partitionID, backlogID)

return { status, refilled, backlogCountUntil, backlogCountTotal, constraintCapacity, refill }
