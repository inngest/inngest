--[[

Output:
  0: Successfully dequeued item
  1: Queue item not found

]]

local keyQueueMap    = KEYS[1]
-- remove items from all outsanding queues it may be in
local keyPartitionFn  = KEYS[2]  -- queue:sorted:$workflowID - zset

local keyConcurrencyFn         = KEYS[3] -- Account concurrency level
local keyCustomConcurrency1    = KEYS[4] -- When leasing an item we need to place the lease into this key.
local keyCustomConcurrency2    = KEYS[5] -- Optional for eg. for concurrency amongst steps
local keyAcctConcurrency       = KEYS[6]

local keyIdempotency           = KEYS[7]

local concurrencyPointer       = KEYS[8]
local keyGlobalPointer         = KEYS[9]
local keyGlobalAccountPointer  = KEYS[10]           -- accounts:sorted - zset
local keyAccountPartitions     = KEYS[11]           -- accounts:$accountId:partition:sorted - zset
local keyPartitionMap          = KEYS[12]

local keyItemIndexA            = KEYS[13]   -- custom item index 1
local keyItemIndexB            = KEYS[14]  -- custom item index 2

local queueID        = ARGV[1]
local idempotencyTTL = tonumber(ARGV[2])
local partitionID    = ARGV[3]
local accountId      = ARGV[4]

-- $include(get_queue_item.lua)
-- $include(get_partition_item.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)

--
-- Fetch this item to see if it was in progress prior to deleting.
local item = get_queue_item(keyQueueMap, queueID)
if item == nil then
	return 1
end

redis.call("HDEL", keyQueueMap, queueID)

-- TODO Are these calls safe? Should we check for present keys?
redis.call("ZREM", keyPartitionFn, queueID)

if idempotencyTTL > 0 then
	redis.call("SETEX", keyIdempotency, idempotencyTTL, "")
end

-- This removes the current queue item from the concurrency/in-progress queue,
-- ensures the concurrency index/scavenger queue is updated to the next earliest in-progress item,
-- and updates the global and account partition pointers to the next earliest item score
local function handleDequeueConcurrency(keyConcurrency)
	redis.call("ZREM", keyConcurrency, item.id) -- remove from concurrency/in-progress queue
end

handleDequeueConcurrency(keyConcurrencyFn)

-- Get the earliest item in the partition concurrency set.  We may be dequeueing
-- the only in-progress job and should remove this from the partition concurrency
-- pointers, if this exists.
--
-- This ensures that scavengeres have updated pointer queues without the currently
-- leased job, if exists.
local concurrencyScores = redis.call("ZRANGE", keyConcurrencyFn, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
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
local minScores = redis.call("ZRANGE", keyPartitionFn, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
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
        update_account_queues(keyGlobalAccountPointer, keyAccountPartitions, partitionID, accountId, earliestScore)

        -- Clear the ForceAtMS from the pointer.
        local existing = get_partition_item(keyPartitionMap, partitionID)
        existing.forceAtMS = nil
        redis.call("HSET", keyPartitionMap, partitionID, cjson.encode(existing))
      end
  end
end

handleDequeueConcurrency(keyCustomConcurrency1)
handleDequeueConcurrency(keyCustomConcurrency2)

-- This does not have a scavenger queue, as it's purely an entitlement limitation. See extendLease
-- and Lease for respective ZADD calls.
redis.call("ZREM", keyAcctConcurrency, item.id)

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
	redis.call("ZREM", keyItemIndexA, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
	redis.call("ZREM", keyItemIndexB, queueID)
end

return 0
