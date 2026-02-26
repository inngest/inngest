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

local keyShadowPartitionMeta             = KEYS[8]
local keyBacklogMeta                     = KEYS[9]

local keyBacklogSet                      = KEYS[10]
local keyShadowPartitionSet              = KEYS[11]
local keyGlobalShadowPartitionSet        = KEYS[12]
local keyGlobalAccountShadowPartitionSet = KEYS[13]
local keyAccountShadowPartitionSet       = KEYS[14]
local keyPartitionNormalizeSet           = KEYS[15]

local keyIdempotency           = KEYS[16]
local singletonRunKey          = KEYS[17]

local keyPartitionScavengerIndex  = KEYS[18]

local keyItemIndexA            = KEYS[19]   -- custom item index 1
local keyItemIndexB            = KEYS[20]  -- custom item index 2

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
-- $include(update_active_sets.lua)

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

-- Remove item from scavenger index
redis.call("ZREM", keyPartitionScavengerIndex, queueID)

-- Get the earliest item in the new scavenger index and old partition concurrency set.  We may be dequeueing
-- the only in-progress job and should remove this from the partition concurrency
-- pointers, if this exists.
--
-- This ensures that scavengeres have updated pointer queues without the currently
-- leased job, if exists.
local scavengerIndexScores = redis.call("ZRANGE", keyPartitionScavengerIndex, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if scavengerIndexScores == false or scavengerIndexScores == nil or #scavengerIndexScores == 0 then
  redis.call("ZREM", concurrencyPointer, partitionID)
else
  local earliestLease = tonumber(scavengerIndexScores[2])

  -- Ensure that we update the score with the earliest lease
  redis.call("ZADD", concurrencyPointer, earliestLease, partitionID)
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
  updateBacklogPointer(keyShadowPartitionMeta, keyBacklogMeta, keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, keyPartitionNormalizeSet, accountID, partitionID, backlogID)
end


-- Remove singleton lock
local singletonKey = redis.call("GET", singletonRunKey)

if singletonKey ~= nil and singletonKey ~= false and keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
  local queueItemsCount = redis.call("ZCOUNT", keyItemIndexA, "-inf", "+inf")
  local singletonRunID = redis.call("GET", singletonKey)

  if tonumber(queueItemsCount) == 0 then
    -- We just dequeued the last step
     redis.call("DEL", singletonRunKey)

     if singletonRunID == runID then
        redis.call("DEL", singletonKey)
    end
  end
end

return 0
