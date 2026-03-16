--[[

  Moves as many queue items from backlog into ready queue as capacity.

  backlogRefill will always attempt to move queue items from backlogs into ready queues.

  Returns a tuple of {
    status,               -- Always 0 (constraint checking handled by Constraint API)
    items_refilled,       -- Number of items refilled to ready queue
    items_until,          -- Number of items within provided time range in backlog before refilling
    items_total,          -- Total number of items in backlog before refilling
    constraintCapacity,   -- Equal to refill (no inline constraint limits)
    refill,               -- Number of items to refill (may include missing items)
    refilled_item_ids,    -- Set of refilled item IDs
    retry_after           -- Always 0 (no inline constraint retries)
  }
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

local keyBacklogActiveCheckSet       = KEYS[13]
local keyBacklogActiveCheckCooldown  = KEYS[14]

local keyPartitionNormalizeSet       = KEYS[15]

local backlogID     = ARGV[1]
local partitionID   = ARGV[2]
local accountID     = ARGV[3]
local refillUntilMS = tonumber(ARGV[4])
local refillItems   = cjson.decode(ARGV[5])
local nowMS         = tonumber(ARGV[6])

local keyPrefix = ARGV[7]

-- $include(update_pointer_score.lua)
-- $include(update_account_queues.lua)
-- $include(update_backlog_pointer.lua)

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

  return { 0, 0, 0, backlogCountTotal, 0, 0, {}, 0 }
end

local backlogCountUntil = redis.call("ZCOUNT", keyBacklogSet, "-inf", refillUntilMS)
if backlogCountUntil == false or backlogCountUntil == nil then
  backlogCountUntil = 0
end

if backlogCountUntil == 0 then
  -- update backlog pointers
  updateBacklogPointer(keyShadowPartitionMeta, keyBacklogMeta, keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, keyPartitionNormalizeSet, accountID, partitionID, backlogID)

  return { 0, 0, backlogCountUntil, backlogCountTotal, 0, 0, {}, 0 }
end

--
-- Calculate initial number of items to refill
--

-- Set items to refill to number of items provided
local refill = #refillItems

-- No inline constraint checking; capacity equals refill
local constraintCapacity = refill
local status = 0
local retryAt = 0

--
-- Refill to match capacity
--

local refilled = 0

-- Only attempt to refill if we have capacity
local refilledItemIDs = {}
if refill > 0 then
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

      table.insert(itemUpdateArgs, itemID)
      table.insert(itemUpdateArgs, cjson.encode(updatedData))

      table.insert(refilledItemIDs, itemID)

      -- Increment number of refilled items
      refilled = refilled + 1
    end
  end

  if refilled > 0 then
    -- "Refill" items to ready set
    redis.call("ZADD", keyReadySet, unpack(readyArgs))

    -- Update queue items with refill data
    redis.call("HSET", keyQueueItemHash, unpack(itemUpdateArgs))
  end

  if hasRemove then
    -- Remove refilled or missing items from backlog
    redis.call("ZREM", keyBacklogSet, unpack(backlogRemArgs))
  end
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

local function update_backlog_successive_constrained_counters(keyBacklogMeta, backlogID, status)
  --
  -- Update successive constrained metrics in backlog meta
  --

  local existing = cjson.decode(redis.call("HGET", keyBacklogMeta, backlogID))

  -- If not constrained, reset counters
  if status == 0 then
    existing.stc = 0
    existing.sccc = 0
  end

  -- If custom concurrency limits hit, increase counter
  if status == 3 or status == 4 then
    local previousSuccessiveCustomConcurrencyConstrained = existing.sccc
    if previousSuccessiveCustomConcurrencyConstrained == false or previousSuccessiveCustomConcurrencyConstrained == nil then
      previousSuccessiveCustomConcurrencyConstrained = 0
    end

    existing.sccc = previousSuccessiveCustomConcurrencyConstrained + 1
  end

  -- If throttled, increase counter
  if status == 5 then
    local previousSuccessiveThrottleConstrained = existing.stc
    if previousSuccessiveThrottleConstrained == false or previousSuccessiveThrottleConstrained == nil then
      previousSuccessiveThrottleConstrained = 0
    end

    existing.stc = previousSuccessiveThrottleConstrained + 1
  end

  redis.call("HSET", keyBacklogMeta, backlogID, cjson.encode(existing))
end

-- Clean up backlog meta if we refilled the last item (or dropped all dangling item pointers)
if tonumber(redis.call("ZCARD", keyBacklogSet)) == 0 then
  redis.call("HDEL", keyBacklogMeta, backlogID)
else
  update_backlog_successive_constrained_counters(keyBacklogMeta, backlogID, status)
end

-- Always update pointers
updateBacklogPointer(keyShadowPartitionMeta, keyBacklogMeta, keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, keyPartitionNormalizeSet, accountID, partitionID, backlogID)

return { status, refilled, backlogCountUntil, backlogCountTotal, constraintCapacity, refill, refilledItemIDs, retryAt }
