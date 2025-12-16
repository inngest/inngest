--[[

  Requeues a backlog in the future and updates pointers in shadow partitions.

  Return values:
  1 - Empty backlog cleaned up
  0 - Requeued backlog
  -1 - Backlog not found
]]


local keyShadowPartitionHash             = KEYS[1]
local keyBacklogMeta                     = KEYS[2]
local keyShadowPartitionMeta             = KEYS[3]

local keyGlobalShadowPartitionSet        = KEYS[4]
local keyGlobalAccountShadowPartitionSet = KEYS[5]
local keyAccountShadowPartitionSet       = KEYS[6]
local keyShadowPartitionSet              = KEYS[7]
local keyBacklogSet                      = KEYS[8]
local keyPartitionNormalizeSet           = KEYS[9]

local accountID   = ARGV[1]
local partitionID = ARGV[2]
local backlogID   = ARGV[3]
local requeueAtMS = tonumber(ARGV[4])

-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(update_backlog_pointer.lua)

if redis.call("HEXISTS", keyBacklogMeta, backlogID) == 0 then
  return -1
end

-- Clean up empty backlog
if tonumber(redis.call("ZCARD", keyBacklogSet)) == 0 then
  redis.call("HDEL", keyBacklogMeta, backlogID)

  -- Update pointers
  updateBacklogPointer(keyShadowPartitionMeta, keyBacklogMeta, keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, keyPartitionNormalizeSet, accountID, partitionID, backlogID)

  return 1
end

-- If backlog has more items, update pointer in shadow partition
update_pointer_score_to(backlogID, keyShadowPartitionSet, requeueAtMS)

-- In case the backlog is the new earliest item in the shadow partition,
-- update pointers to shadow partition in global indexes
local earliestShadowPartitionScore = get_earliest_score(keyShadowPartitionSet)

-- Push back shadow partition in global set
update_pointer_score_to(partitionID, keyGlobalShadowPartitionSet, earliestShadowPartitionScore)

-- Push back shadow partition in account set + potentially push back account in global accounts set
update_account_shadow_queues(keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, partitionID, accountID, earliestShadowPartitionScore)

return 0
