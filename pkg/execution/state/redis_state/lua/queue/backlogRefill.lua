--[[

  Checks available capacity by validating concurrency and other queue capacity constraints.
  Moves as many queue items from backlog into ready queue as capacity.

  backlogRefill will always attempt to move queue items from backlogs into ready queues up to
  hitting concurrency.

  Returns a tuple of {status, items_refilled}

  Status values:

  0 - Did not hit constraint
  1 - Account concurrency limit reached
  2 - Function concurrency limit reached
  3 - Custom concurrency key 1 limit reached
  4 - Custom concurrency key 2 limit reached
]]

local keyBacklogSet                      = KEYS[1]
local keyShadowPartitionSet              = KEYS[2]
local keyGlobalShadowPartitionSet        = KEYS[3]
local keyGlobalAccountShadowPartitionSet = KEYS[4]
local keyAccountShadowPartitionSet       = KEYS[5]
local keyReadySet                        = KEYS[6]

local backlogID   = ARGV[1]
local partitionID = ARGV[2]
local accountID   = ARGV[3]
local refillUntil = tonumber(ARGV[4])
local refillLimit = tonumber(ARGV[5])

-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)

-- Start with full capacity: max(number of items in the backlog, hard limit, e.g. 100)
local backlogCount = redis.call("ZCOUNT", keyBacklogSet, "-inf", refillUntil)
local capacity = backlogCount
if backlogCount > refillLimit then
  capacity = refillLimit
end

--
-- Check constraints and adjust capacity
--

-- Set initial status to success, progressively add more specific capacity constraints
local status = 0

-- TODO Retrieve account capacity
local remainingAccountCapacity = 100
if remainingAccountCapacity < capacity then
  -- Account concurrency imposes limits
  capacity = remainingAccountCapacity
  status = 1
end

-- TODO Retrieve function capacity
local remainingFunctionCapacity = 100
if remainingFunctionCapacity < capacity then
  -- Function concurrency imposes limits
  capacity = remainingFunctionCapacity
  status = 2
end

-- TODO Retrieve concurrency key 1 capacity
local remainingCustomConcurrencyCapacityKey1 = 100
if remainingCustomConcurrencyCapacityKey1 < capacity then
  -- Custom concurrency key 1 imposes limits
  capacity = remainingCustomConcurrencyCapacityKey1
  status = 3
end

-- TODO Retrieve concurrency key 2 capacity
local remainingCustomConcurrencyCapacityKey2 = 100
if remainingCustomConcurrencyCapacityKey2 < capacity then
  -- Custom concurrency key 2 imposes limits
  capacity = remainingCustomConcurrencyCapacityKey2
  status = 4
end

--
-- Refill to match capacity
--

local refilled = 0

-- Only attempt to refill if we have capacity
if capacity > 0 then
  -- Move item(s) out of backlog and into partition

  local items = redis.call("ZRANGE", keyBacklogSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, capacity, "WITHSCORES")

  -- Reverse the items to be added to the ready set
  local args = {}
  local remArgs = {}
  -- advance by two as items is essentially a tuple of (item ID, score)[]
  for i = 1, #items, 2 do
    table.insert(args, items[i + 1]) -- score
    table.insert(args, items[i])     -- item
    table.insert(remArgs, items[i])  -- item for removal
    refilled = refilled + 1
  end
  redis.call("ZADD", keyReadySet, unpack(args))
  redis.call("ZREM", keyBacklogSet, unpack(remArgs))
end

--
-- Adjust pointer scores for shadow scanning, potentially clean up
--

-- Retrieve the earliest item score in the backlog
local minScores = redis.call("ZRANGE", keyBacklogSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")

-- If backlog is empty, update dangling pointers in shadow partition
if minScores == nil or minScores == false or minScores[2] == nil then
  redis.call("ZREM", keyShadowPartitionSet, backlogID)

  -- If shadow partition has no more backlogs, update global/account pointers
  if tonumber(redis.call("ZCARD", keyShadowPartitionSet)) == 0 then
    redis.call("ZREM", keyGlobalShadowPartitionSet, partitionID)
    redis.call("ZREM", keyAccountShadowPartitionSet, partitionID)

    if tonumber(redis.call("ZCARD", keyAccountShadowPartitionSet)) == 0 then
      redis.call("ZREM", keyGlobalAccountShadowPartitionSet, accountID)
    end
  end

  return {status,refilled}
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

return {status,refilled}
