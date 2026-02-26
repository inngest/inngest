--[[

  BacklogRefill moves the specified items from backlogs into the ready queue.

  If items do not exist, 

  Returns a tuple of {
    items_total,          -- Total number of items in backlog before refilling
    items_until,          -- Number of items within provided time range in backlog before refilling
    refilled_item_ids,    -- Set of refilled item IDs
  }

  Status values:

  0 - Did not hit constraint
  1 - Account concurrency limit reached
  2 - Function concurrency limit reached
  3 - Custom concurrency key 1 limit reached
  4 - Custom concurrency key 2 limit reached
  5 - Throttled
]]

local keyShadowPartitionMeta             = KEYS[1]
local keyBacklogMeta                     = KEYS[2]

local keyBacklogSet                      = KEYS[3]
local keyShadowPartitionSet              = KEYS[4]
local keyGlobalShadowPartitionSet        = KEYS[5]
local keyGlobalAccountShadowPartitionSet = KEYS[6]
local keyAccountShadowPartitionSet       = KEYS[7]

local keyReadySet                        = KEYS[8]
local keyGlobalPointer        	         = KEYS[9] -- partition:sorted - zset
local keyGlobalAccountPointer 	         = KEYS[10] -- accounts:sorted - zset
local keyAccountPartitions    	         = KEYS[11] -- accounts:$accountID:partition:sorted - zset

local keyQueueItemHash                   = KEYS[12]

local keyPartitionNormalizeSet       = KEYS[13]

local backlogID     = ARGV[1]
local partitionID   = ARGV[2]
local accountID     = ARGV[3]
local refillUntilMS = tonumber(ARGV[4])
local refillItems   = cjson.decode(ARGV[5])
local nowMS         = tonumber(ARGV[6])

-- Constraint API rollout
local itemCapacityLeases = {}
if ARGV[18] ~= nil and ARGV[18] ~= "" and ARGV[18] ~= "null" then
  local success, result = pcall(cjson.decode, ARGV[18])
  if success and type(result) == "table" then
    itemCapacityLeases = result
  end
end

-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(gcra.lua)
-- $include(update_backlog_pointer.lua)
-- $include(update_active_sets.lua)
-- $include(active_check.lua)

--
-- Retrieve current backlog size
--

local backlogCountTotal = redis.call("ZCARD", keyBacklogSet)
if backlogCountTotal == false or backlogCountTotal == nil then
  backlogCountTotal = 0
end

if backlogCountTotal == 0 then
  -- Clean up metadata if the backlog is empty
  redis.call("HDEL", keyBacklogMeta, backlogID)

  -- update backlog pointers
  updateBacklogPointer(keyShadowPartitionMeta, keyBacklogMeta, keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, keyPartitionNormalizeSet, accountID, partitionID, backlogID)

  return { 0, 0, {} }
end

local backlogCountUntil = redis.call("ZCOUNT", keyBacklogSet, "-inf", refillUntilMS)
if backlogCountUntil == false or backlogCountUntil == nil then
  backlogCountUntil = 0
end

if backlogCountUntil == 0 then
  -- update backlog pointers
  updateBacklogPointer(keyShadowPartitionMeta, keyBacklogMeta, keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, keyPartitionNormalizeSet, accountID, partitionID, backlogID)

  return { backlogCountTotal, 0, {} }
end

--
-- Calculate initial number of items to refill
--

-- Set items to refill to number of items provided
local refill = #refillItems

--
-- Refill to match capacity
--

local refilledItemIDs = {}

-- Move item(s) out of backlog and into partition

-- Use provided item IDs, limited by final refill count
local itemIDs = {}
for i = 1, math.min(refill, #refillItems) do
  table.insert(itemIDs, refillItems[i])
end

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

  -- If queue item does not exist in backlog, skip
  local missingInBacklog = itemScore == nil

  -- If queue item does not exist in hash, delete from backlog
  local missingInHash = itemData == false or itemData == nil or itemData == ""

  if missingInBacklog then
    -- no-op
  elseif missingInHash then
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

    -- Update item with Capacity Lease if lease acquired
    if itemCapacityLeases ~= nil and #itemCapacityLeases > 0 then
      updatedData.cl = itemCapacityLeases[i]
    end

    table.insert(itemUpdateArgs, itemID)
    table.insert(itemUpdateArgs, cjson.encode(updatedData))

    table.insert(refilledItemIDs, itemID)
  end
end

  if #refilledItemIDs > 0 then
    -- "Refill" items to ready set
    redis.call("ZADD", keyReadySet, unpack(readyArgs))

    -- Update queue items with refill data
    redis.call("HSET", keyQueueItemHash, unpack(itemUpdateArgs))
  end

  if hasRemove then
    -- Remove refilled or missing items from backlog
    redis.call("ZREM", keyBacklogSet, unpack(backlogRemArgs))
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

-- Clean up backlog meta if we refilled the last item (or dropped all dangling item pointers)
if tonumber(redis.call("ZCARD", keyBacklogSet)) == 0 then
  redis.call("HDEL", keyBacklogMeta, backlogID)
end

-- Always update pointers
updateBacklogPointer(keyShadowPartitionMeta, keyBacklogMeta, keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, keyPartitionNormalizeSet, accountID, partitionID, backlogID)

return { backlogCountTotal, backlogCountUntil, refilledItemIDs }
