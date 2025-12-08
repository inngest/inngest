--[[

Re-enqueus a queue item within its queue, removing any lease.

Output:
  0: Successfully re-enqueued item
  1: Queue item not found
  2: Successfully re-queued to backlog -- TODO: this should be a temporary status
]]

local queueKey                = KEYS[1] -- queue:item - hash: { $itemID: $item }
local keyPartitionMap         = KEYS[2] -- partition:item - hash: { $workflowID: $partition }
local concurrencyPointer      = KEYS[3]

local keyGlobalPointer        = KEYS[4] -- partition:sorted - zset
local keyGlobalAccountPointer = KEYS[5] -- accounts:sorted - zset
local keyAccountPartitions    = KEYS[6] -- accounts:$accountID:partition:sorted

local keyReadyQueue           = KEYS[7] -- queue:sorted:$workflowID - zset

local keyInProgressAccount                  = KEYS[8]
local keyInProgressPartition                = KEYS[9]
local keyInProgressCustomConcurrencyKey1    = KEYS[10]
local keyInProgressCustomConcurrencyKey2    = KEYS[11]

local keyActiveAccount             = KEYS[12]
local keyActivePartition           = KEYS[13]
local keyActiveConcurrencyKey1     = KEYS[14]
local keyActiveConcurrencyKey2     = KEYS[15]
local keyActiveCompound            = KEYS[16]

local keyActiveRun                        = KEYS[17]
local keyActiveRunsAccount                = KEYS[18]
local keyActiveRunsPartition              = KEYS[19]
local keyActiveRunsCustomConcurrencyKey1  = KEYS[20]
local keyActiveRunsCustomConcurrencyKey2  = KEYS[21]

local keyBacklogSet                      = KEYS[22]          -- backlog:sorted:<backlogID> - zset
local keyBacklogMeta                     = KEYS[23]          -- backlogs - hash
local keyGlobalShadowPartitionSet        = KEYS[24]          -- shadow:sorted
local keyShadowPartitionSet              = KEYS[25]          -- shadow:sorted:<fnID|queueName> - zset
local keyShadowPartitionMeta             = KEYS[26]          -- shadows
local keyGlobalAccountShadowPartitionSet = KEYS[27]
local keyAccountShadowPartitionSet       = KEYS[28]

local keyPartitionScavengerIndex  = KEYS[29]

local keyItemIndexA           = KEYS[30]          -- custom item index 1
local keyItemIndexB           = KEYS[31]          -- custom item index 2

local queueID             = ARGV[1]           -- id
local queueItem           = ARGV[2]
local queueScore          = tonumber(ARGV[3]) -- vesting time, in ms
local accountID           = ARGV[4]
local runID               = ARGV[5]
local partitionID         = ARGV[6]
local partitionItem       = ARGV[7]

local nowMS               = tonumber(ARGV[8]) -- now in ms

-- Key queues v2
local requeueToBacklog				= tonumber(ARGV[9])
local shadowPartitionItem     = ARGV[10]
local backlogID               = ARGV[11]
local backlogItem             = ARGV[12]

local updateConstraintState = tonumber(ARGV[13])

-- $include(get_queue_item.lua)
-- $include(get_partition_item.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(enqueue_to_partition.lua)
-- $include(update_active_sets.lua)

local item = get_queue_item(queueKey, queueID)
if item == nil then
    return 1
end

-- Update the queue item with a nil lease, at, atMS, etc.
redis.call("HSET", queueKey, queueID, queueItem)

-- Remove item from ready queue
redis.call("ZREM", keyReadyQueue, item.id)

if updateConstraintState == 1 then
  -- This removes the queue item from the concurrency/in-progress queue and ensures that the concurrency
  -- index/scavenger queue is updated to the next earliest item.
  -- This is the first half of requeueing: Removing the in-progress item, which must be followed up
  -- by enqueueing back to the partition queues
  local function handleRequeueConcurrency(keyConcurrency)
    redis.call("ZREM", keyConcurrency, item.id) -- Remove from in-progress queue
  end

  --
  -- Concurrency
  --

  handleRequeueConcurrency(keyInProgressPartition)

  if exists_without_ending(keyInProgressCustomConcurrencyKey1, ":-") then
    handleRequeueConcurrency(keyInProgressCustomConcurrencyKey1)
  end

  if exists_without_ending(keyInProgressCustomConcurrencyKey2, ":-") then
    handleRequeueConcurrency(keyInProgressCustomConcurrencyKey2)
  end

  if exists_without_ending(keyInProgressAccount, ":-") then
      -- Remove item from the account concurrency queue
      -- This does not have a scavenger queue, as it's purely an entitlement limitation. See extendLease
      -- and Lease for respective ZADD calls.
      redis.call("ZREM", keyInProgressAccount, item.id)
  end

  -- Remove item from active sets
  removeFromActiveSets(keyActivePartition, keyActiveAccount, keyActiveCompound, keyActiveConcurrencyKey1, keyActiveConcurrencyKey2, queueID)
  removeFromActiveRunSets(keyActiveRun, keyActiveRunsPartition, keyActiveRunsAccount, keyActiveRunsCustomConcurrencyKey1, keyActiveRunsCustomConcurrencyKey2, runID, queueID)
end

-- Remove item from scavenger index
redis.call("ZREM", keyPartitionScavengerIndex, item.id)

-- Get the earliest item in the new scavenger index and old partition concurrency set.  We may be dequeueing
-- the only in-progress job and should remove this from the partition concurrency
-- pointers, if this exists.
--
-- This ensures that scavengeres have updated pointer queues without the currently
-- leased job, if exists.
local concurrencyScores = redis.call("ZRANGE", keyInProgressPartition, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
local scavengerIndexScores = redis.call("ZRANGE", keyPartitionScavengerIndex, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if scavengerIndexScores == false and concurrencyScores == false then
  redis.call("ZREM", concurrencyPointer, partitionID)
else
  -- Either scavenger index or partition in progress set includes more items

  local earliestLease = nil
  if scavengerIndexScores ~= false and scavengerIndexScores ~= nil then
    earliestLease = tonumber(scavengerIndexScores[2])
  end

  -- Fall back to in progress set
  if earliestLease == nil or (
    concurrencyScores ~= false and
    concurrencyScores ~= nil and
    #concurrencyScores > 0 and
    tonumber(concurrencyScores[2]) < earliestLease
  ) then
    earliestLease = tonumber(concurrencyScores[2])
  end

  if earliestLease == nil then
    redis.call("ZREM", concurrencyPointer, partitionID)
  else
    -- Ensure that we update the score with the earliest lease
    redis.call("ZADD", concurrencyPointer, earliestLease, partitionID)
  end
end

if requeueToBacklog == 1 then
	--
	-- Requeue item to backlog queues again
	--
  requeue_to_backlog(keyBacklogSet, backlogID, backlogItem, partitionID, shadowPartitionItem, partitionItem, keyPartitionMap, keyBacklogMeta, keyGlobalShadowPartitionSet, keyShadowPartitionMeta, keyShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, queueScore, queueID, accountID)
else
  --
  -- Enqueue item to partition queues again
  --
  requeue_to_partition(keyReadyQueue, partitionID, partitionItem, keyPartitionMap, keyGlobalPointer, keyGlobalAccountPointer, keyAccountPartitions, queueScore, queueID, nowMS, accountID)
end

-- Add optional indexes.
if keyItemIndexA ~= "" and keyItemIndexA ~= false and keyItemIndexA ~= nil then
    redis.call("ZADD", keyItemIndexA, queueScore, queueID)
end
if keyItemIndexB ~= "" and keyItemIndexB ~= false and keyItemIndexB ~= nil then
    redis.call("ZADD", keyItemIndexB, queueScore, queueID)
end

return 0
