local keyBacklogSet                      = KEYS[1]
local keyShadowPartitionSet              = KEYS[2]
local keyGlobalShadowPartitionSet        = KEYS[3]
local keyGlobalAccountShadowPartitionSet = KEYS[4]
local keyAccountShadowPartitionSet       = KEYS[5]

local backlogID   = ARGV[1]
local partitionID = ARGV[2]
local accountID   = ARGV[3]

-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)

-- TODO Enforce constraints

-- TODO Move item(s) out of backlog and into partition

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

  return 0
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

return 0
