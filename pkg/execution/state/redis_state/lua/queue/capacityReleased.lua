--[[

Wakes a partition after the Constraint API confirmed a capacity release.

Output:
  0: Partition marked as released
  1: Partition not found

]]

local keyPartitionMap         = KEYS[1]
local keyReadyQueue           = KEYS[2]
local keyGlobalPointer        = KEYS[3]
local keyGlobalAccountPointer = KEYS[4]
local keyAccountPartitions    = KEYS[5]

local partitionID             = ARGV[1]
local accountID               = ARGV[2]

-- $include(get_partition_item.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)

local existing = get_partition_item(keyPartitionMap, partitionID)
if existing == nil then
  return 1
end

-- The next partition lease consumes this flag and skips cached constraint
-- exhaustion, which predates the release. If the partition is currently
-- leased, partitionRequeue sees the flag and won't force a constrained backoff.
existing.capacityReleased = true

-- Same as the dequeue-time wakeup: bring the partition forward to its
-- earliest ready item, even if it was forced into the future.
local minScores = redis.call("ZRANGE", keyReadyQueue, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if minScores ~= nil and minScores ~= false and #minScores ~= 0 then
  local currentScore = redis.call("ZSCORE", keyGlobalPointer, partitionID)
  if currentScore ~= nil and currentScore ~= false then
    local earliestScore = tonumber(minScores[2])/1000
    if tonumber(currentScore) > earliestScore then
      update_pointer_score_to(partitionID, keyGlobalPointer, earliestScore)
      update_account_queues(keyGlobalAccountPointer, keyAccountPartitions, partitionID, accountID, earliestScore)
      existing.forceAtMS = nil
    end
  end
end

redis.call("HSET", keyPartitionMap, partitionID, cjson.encode(existing))

return 0
