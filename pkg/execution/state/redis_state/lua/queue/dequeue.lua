--[[

Output:
  0: Successfully dequeued item
  1: Queue item not found

]]

local keyQueueMap              = KEYS[1]
local keyPartitionMap          = KEYS[2]

local concurrencyPointer       = KEYS[3]

local keyReadyQueue            = KEYS[4]  -- queue:sorted:$workflowID - zset
local keyGlobalPointer         = KEYS[5]
local keyGlobalAccountPointer  = KEYS[6]           -- accounts:sorted - zset
local keyAccountPartitions     = KEYS[7]           -- accounts:$accountID:partition:sorted - zset

local keyBacklogSet                      = KEYS[8]
local keyShadowPartitionSet              = KEYS[9]
local keyGlobalShadowPartitionSet        = KEYS[10]
local keyGlobalAccountShadowPartitionSet = KEYS[11]
local keyAccountShadowPartitionSet       = KEYS[12]

local keyInProgressAccount                  = KEYS[13]
local keyInProgressPartition                = KEYS[14] -- Account concurrency level
local keyInProgressCustomConcurrencyKey1    = KEYS[15] -- When leasing an item we need to place the lease into this key.
local keyInProgressCustomConcurrencyKey2    = KEYS[16] -- Optional for eg. for concurrency amongst steps

local keyActiveAccount             = KEYS[17]
local keyActivePartition           = KEYS[18]
local keyActiveConcurrencyKey1     = KEYS[19]
local keyActiveConcurrencyKey2     = KEYS[20]
local keyActiveCompound            = KEYS[21]
local keyActiveRun                 = KEYS[22]
local keyIndexActivePartitionRuns  = KEYS[23]

local keyIdempotency           = KEYS[24]

local singletonRunKey          = KEYS[25]

local keyItemIndexA            = KEYS[26]   -- custom item index 1
local keyItemIndexB            = KEYS[27]  -- custom item index 2

local queueID        = ARGV[1]
local partitionID    = ARGV[2]
local backlogID      = ARGV[3]
local accountID      = ARGV[4]
local runID          = ARGV[5]
local idempotencyTTL = tonumber(ARGV[6])

-- $include(get_queue_item.lua)
-- $include(get_partition_item.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(update_backlog_pointer.lua)

--
-- Fetch this item to see if it was in progress prior to deleting.
local item = get_queue_item(keyQueueMap, queueID)
if item == nil then
	return 1
end

redis.call("HDEL", keyQueueMap, queueID)

-- TODO Are these calls safe? Should we check for present keys?
redis.call("ZREM", keyReadyQueue, queueID)

if idempotencyTTL > 0 then
	redis.call("SETEX", keyIdempotency, idempotencyTTL, "")
end

-- This removes the current queue item from the concurrency/in-progress queue,
-- ensures the concurrency index/scavenger queue is updated to the next earliest in-progress item,
-- and updates the global and account partition pointers to the next earliest item score
local function handleDequeueConcurrency(keyConcurrency)
	redis.call("ZREM", keyConcurrency, item.id) -- remove from concurrency/in-progress queue
end

handleDequeueConcurrency(keyInProgressPartition)

-- Get the earliest item in the partition concurrency set.  We may be dequeueing
-- the only in-progress job and should remove this from the partition concurrency
-- pointers, if this exists.
--
-- This ensures that scavengeres have updated pointer queues without the currently
-- leased job, if exists.
local concurrencyScores = redis.call("ZRANGE", keyInProgressPartition, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if concurrencyScores == false then
  redis.call("ZREM", concurrencyPointer, partitionID)
else
  local earliestLease = tonumber(concurrencyScores[2])
  if earliestLease == nil then
    redis.call("ZREM", concurrencyPointer, partitionID)
  else
    -- Ensure that we update the score with the earliest lease
    redis.call("ZADD", concurrencyPointer, earliestLease, partitionID)
  end
end

-- For each partition, we now have an extra available capacity.  Check the partition's
-- score, and ensure that it's updated in the global pointer index.
--
local minScores = redis.call("ZRANGE", keyReadyQueue, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if minScores ~= nil and minScores ~= false and #minScores ~= 0 then
  -- If there's nothing int he partition set (no more jobs), end early, as we don't need to
  -- check partition scores.
  local currentScore = redis.call("ZSCORE", keyGlobalPointer, partitionID)
  if currentScore ~= nil and currentScore ~= false then
    local earliestScore = tonumber(minScores[2])/1000
      if tonumber(currentScore) > earliestScore then
        -- Update the global index now that there's capacity, even if we've forced, as we now
        -- have capacity.  Note the earliest score is in MS while partitions are stored in S.
        update_pointer_score_to(partitionID, keyGlobalPointer, earliestScore)
        update_account_queues(keyGlobalAccountPointer, keyAccountPartitions, partitionID, accountID, earliestScore)

        -- Clear the ForceAtMS from the pointer.
        local existing = get_partition_item(keyPartitionMap, partitionID)
        existing.forceAtMS = nil
        redis.call("HSET", keyPartitionMap, partitionID, cjson.encode(existing))
      end
  end
end

if exists_without_ending(keyInProgressCustomConcurrencyKey1, ":-") then
  handleDequeueConcurrency(keyInProgressCustomConcurrencyKey1)
end

if exists_without_ending(keyInProgressCustomConcurrencyKey2, ":-") then
  handleDequeueConcurrency(keyInProgressCustomConcurrencyKey2)
end

if exists_without_ending(keyInProgressAccount, ":-") then
  -- This does not have a scavenger queue, as it's purely an entitlement limitation. See extendLease
  -- and Lease for respective ZADD calls.
  redis.call("ZREM", keyInProgressAccount, item.id)
end

-- Decrease active counters and clean up if necessary
if redis.call("DECR", keyActivePartition) <= 0 then
  redis.call("DEL", keyActivePartition)
end

if exists_without_ending(keyActiveAccount, ":-") then
  if redis.call("DECR", keyActiveAccount) <= 0 then
    redis.call("DEL", keyActiveAccount)
  end
end

if exists_without_ending(keyActiveCompound, ":-") then
  if redis.call("DECR", keyActiveCompound) <= 0 then
    redis.call("DEL", keyActiveCompound)
  end
end

if exists_without_ending(keyActiveConcurrencyKey1, ":-") then
  if redis.call("DECR", keyActiveConcurrencyKey1) <= 0 then
    redis.call("DEL", keyActiveConcurrencyKey1)
  end
end

if exists_without_ending(keyActiveConcurrencyKey2, ":-") then
  if redis.call("DECR", keyActiveConcurrencyKey2) <= 0 then
    redis.call("DEL", keyActiveConcurrencyKey2)
  end
end

if exists_without_ending(keyActiveRun, ":-") then
  -- increase number of active items in the run
  if redis.call("DECR", keyActiveRun) <= 0 then
    redis.call("DEL", keyActiveRun)

    -- update set of active function runs
    if exists_without_ending(keyIndexActivePartitionRuns, ":-") then
      redis.call("SREM", keyIndexActivePartitionRuns, runID)
    end
  end
end

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
	redis.call("ZREM", keyItemIndexA, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
	redis.call("ZREM", keyItemIndexB, queueID)
end

-- If item is in backlog, remove
local backlogScore = tonumber(redis.call("ZSCORE", keyBacklogSet, queueID))
if backlogScore ~= nil and backlogScore ~= false and backlogScore > 0 then
  redis.call("ZREM", keyBacklogSet, queueID)

  -- update backlog pointers
  updateBacklogPointer(keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, accountID, partitionID, backlogID)
end


-- Remove singleton lock
local singletonKey = redis.call("GET", singletonRunKey)

if singletonKey ~= nil and singletonKey ~= false and keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
  local queueItemsCount = redis.call("ZCOUNT", keyItemIndexA, "-inf", "+inf")
  local singletonRunID = redis.call("GET", singletonKey)

  if tonumber(queueItemsCount) == 0 and singletonRunID == runID then
    -- We just dequeued the last step
    redis.call("DEL", singletonKey)
    redis.call("DEL", singletonRunKey)
  end
end

return 0
