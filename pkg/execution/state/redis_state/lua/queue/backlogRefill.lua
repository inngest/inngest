--[[

  Checks available capacity by validating concurrency and other queue capacity constraints.
  Moves as many queue items from backlog into ready queue as capacity.

  backlogRefill will always attempt to move queue items from backlogs into ready queues up to
  hitting concurrency.

  Returns a tuple of {
    status,               -- See status section below
    items_refilled,       -- Number of items refilled to ready queue
    total_items,          -- Total number of items in backlog before refilling
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
local keyQueueItemHash                   = KEYS[7]

-- Constraint-related accounting keys
local keyConcurrencyAccount      = KEYS[8]
local keyConcurrencyFn  				 = KEYS[9] -- Account concurrency level
local keyCustomConcurrencyKey1   = KEYS[10] -- When leasing an item we need to place the lease into this key.
local keyCustomConcurrencyKey2   = KEYS[11] -- Optional for eg. for concurrency amongst steps
local keyActiveCounter           = KEYS[12]

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

-- $include(check_concurrency.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(gcra.lua)

-- Helper method to clean up backlog pointers
local function cleanupBacklogPointer()
  redis.call("ZREM", keyShadowPartitionSet, backlogID)

  -- If shadow partition has no more backlogs, update global/account pointers
  if tonumber(redis.call("ZCARD", keyShadowPartitionSet)) == 0 then
    redis.call("ZREM", keyGlobalShadowPartitionSet, partitionID)
    redis.call("ZREM", keyAccountShadowPartitionSet, partitionID)

    if tonumber(redis.call("ZCARD", keyAccountShadowPartitionSet)) == 0 then
      redis.call("ZREM", keyGlobalAccountShadowPartitionSet, accountID)
    end
  end
end

--
-- Retrieve current backlog size
--

local backlogCount = redis.call("ZCOUNT", keyBacklogSet, "-inf", refillUntilMS)
if backlogCount == false or backlogCount == nil then
  backlogCount = 0
end

-- If backlog is empty, immediately clean up pointers and return
if backlogCount == 0 then
  cleanupBacklogPointer()
  return { 0, 0, 0, 0, 0 }
end

--
-- Calculate initial number of items to refill
--

-- Set items to refill to number of items in backlog
local refill = backlogCount

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

-- Check throttle capacity
if (constraintCapacity == nil or constraintCapacity > 0) and throttleLimit > 0 then
  local remainingThrottleCapacity = gcraCapacity(throttleKey, nowMS, throttlePeriod * 1000, throttleLimit, throttleBurst)
  if constraintCapacity == nil or remainingThrottleCapacity < constraintCapacity then
    constraintCapacity = remainingThrottleCapacity
    status = 5
  end
end

-- Check custom concurrency key 2 capacity
if (constraintCapacity == nil or constraintCapacity > 0) and exists_without_ending(keyCustomConcurrencyKey2, ":-") == true and customConcurrencyKey2 > 0 then
  local remainingCustomConcurrencyCapacityKey2 = check_concurrency(nowMS, keyCustomConcurrencyKey2, customConcurrencyKey2)
  if constraintCapacity == nil or remainingCustomConcurrencyCapacityKey2 < constraintCapacity then
    -- Custom concurrency key 2 imposes limits
    constraintCapacity = remainingCustomConcurrencyCapacityKey2
    status = 4
  end
end

-- Check custom concurrency key 1 capacity
if (constraintCapacity == nil or constraintCapacity > 0) and exists_without_ending(keyCustomConcurrencyKey1, ":-") == true and customConcurrencyKey1 > 0 then
  local remainingCustomConcurrencyCapacityKey1 = check_concurrency(nowMS, keyCustomConcurrencyKey1, customConcurrencyKey1)
  if constraintCapacity == nil or remainingCustomConcurrencyCapacityKey1 < constraintCapacity then
    -- Custom concurrency key 1 imposes limits
    constraintCapacity = remainingCustomConcurrencyCapacityKey1
    status = 3
  end
end

-- Check function concurrency capacity
if (constraintCapacity == nil or constraintCapacity > 0) and exists_without_ending(keyConcurrencyFn, ":-") == true and concurrencyFn > 0 then
  local remainingFunctionCapacity = check_concurrency(nowMS, keyConcurrencyFn, concurrencyFn)
  if constraintCapacity == nil or remainingFunctionCapacity < constraintCapacity then
    -- Function concurrency imposes limits
    constraintCapacity = remainingFunctionCapacity
    status = 2
  end
end

-- Check account concurrency capacity
if (constraintCapacity == nil or constraintCapacity > 0) and exists_without_ending(keyConcurrencyAccount, ":-") == true and concurrencyAcct > 0 then
  local remainingAccountCapacity = check_concurrency(nowMS, keyConcurrencyAccount, concurrencyAcct)

  if constraintCapacity == nil or remainingAccountCapacity < constraintCapacity then
    -- Account concurrency imposes limits
    constraintCapacity = remainingAccountCapacity
    status = 1
  end
end

-- If we have limited capacity, reduce refill by ready to prevent over-filling
-- Put differently: Assume items in the ready queue are as good as running
if constraintCapacity > 0 then
  -- Note: This is _not_ the active count, but instead only counts items in the ready set!
  local readyCount = redis.call("ZCARD", keyReadySet)
  if readyCount ~= nil and readyCount ~= false then
    -- If we have capacity constraints, we must subtract the number of items
    -- in the ready queue from the number of items to refill, otherwise
    -- we would ignore items not yet running but ready to start at any moment.
    refill = refill - readyCount
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

  local items = redis.call("ZRANGE", keyBacklogSet, "-inf", refillUntilMS, "BYSCORE", "LIMIT", 0, refill, "WITHSCORES")

  local potentiallyMissingQueueItems = redis.call("HMGET", keyQueueItemHash, unpack(items))

  -- Reverse the items to be added to the ready set
  local args = {}
  local remArgs = {}

  local itemCleanupArgs = {}
  local hasCleanup = false

  -- advance by two as items is essentially a tuple of (item ID, score)[]
  for i = 1, #items, 2 do
    if potentiallyMissingQueueItems[i] == false or potentiallyMissingQueueItems[i] == nil or potentiallyMissingQueueItems[i] == "" then
      table.insert(itemCleanupArgs, items[i])
      table.insert(remArgs, items[i])  -- item for removal
      hasCleanup = true
      goto continue
    end

    table.insert(args, items[i + 1]) -- score
    table.insert(args, items[i])     -- item
    table.insert(remArgs, items[i])  -- item for removal
    refilled = refilled + 1

    ::continue::
  end

  redis.call("ZADD", keyReadySet, unpack(args))
  redis.call("INCRBY", keyActiveCounter, refilled)
  redis.call("ZREM", keyBacklogSet, unpack(remArgs))


  if hasCleanup == true then
    redis.call("HDEL", keyQueueItemHash, unpack(itemCleanupArgs))
  end
end

-- update gcra theoretical arrival time
if throttleLimit > 0 then
  gcraUpdate(throttleKey, nowMS, throttlePeriod * 1000, throttleLimit, throttleBurst, refill)
end

--
-- Adjust pointer scores for shadow scanning, potentially clean up
--

-- Retrieve the earliest item score in the backlog
local minScores = redis.call("ZRANGE", keyBacklogSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")

-- If backlog is empty, update dangling pointers in shadow partition
if minScores == nil or minScores == false or minScores[2] == nil then
  cleanupBacklogPointer()

  return { status, refilled, backlogCount, constraintCapacity, refill }
end

local earliestScoreBacklog = tonumber(minScores[2])
local updateTo = earliestScoreBacklog/1000

-- If backlog has more items, update pointer in shadow partition
update_pointer_score_to(backlogID, keyShadowPartitionSet, updateTo)

-- In case the backlog is the new earliest item in the shadow partition,
-- update pointers to shadow partition in global indexes
local minScores = redis.call("ZRANGE", keyShadowPartitionSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
local earliestScoreShadowPartition = tonumber(minScores[2])

if earliestScoreBacklog < earliestScoreShadowPartition then
  -- Push back shadow partition in global set
  update_pointer_score_to(partitionID, keyGlobalShadowPartitionSet, updateTo)

  -- Push back shadow partition in account set + potentially push back account in global accounts set
  update_account_shadow_queues(keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, partitionID, accountID, updateTo)
end

return { status, refilled, backlogCount, constraintCapacity, refill }
