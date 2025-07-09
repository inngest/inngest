--[[

  Removes backlog pointer from shadow partition and into normalize partition.
  Will update shadow partition pointers accordingly.

  Return status values:

  1 - Moved backlog to normalize set
  -1 - Garbage-collected empty backlog
]]

local keyBacklogMeta                     = KEYS[1]
local keyShadowPartitionMeta             = KEYS[2]

local keyBacklogSet                      = KEYS[3]
local keyShadowPartitionSet              = KEYS[4]
local keyGlobalShadowPartitionSet        = KEYS[5]
local keyGlobalAccountShadowPartitionSet = KEYS[6]
local keyAccountShadowPartitionSet       = KEYS[7]

local keyGlobalNormalizeSet              = KEYS[8]
local keyAccountNormalizeSet             = KEYS[9]
local keyPartitionNormalizeSet           = KEYS[10]

local backlogID             = ARGV[1]
local partitionID           = ARGV[2]
local accountID             = ARGV[3]
local normalizeTime         = tonumber(ARGV[4])
local normalizeAsyncMinimum = tonumber(ARGV[5])

-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)
-- $include(update_backlog_pointer.lua)

local backlogCount = redis.call("ZCARD", keyBacklogSet)

-- If backlog is empty, garbage-collect it from shadow partition
if backlogCount == nil or backlogCount == false or backlogCount == 0 then
  -- Update pointers
  updateBacklogPointer(keyShadowPartitionMeta, keyBacklogMeta, keyGlobalShadowPartitionSet, keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, keyShadowPartitionSet, keyBacklogSet, keyPartitionNormalizeSet, accountID, partitionID, backlogID)

  return -1
end

-- Add to normalize sets
local currentScore = redis.call("ZSCORE", keyPartitionNormalizeSet, backlogID)
if currentScore == false or tonumber(currentScore) > normalizeTime then
  redis.call("ZADD", keyPartitionNormalizeSet, normalizeTime, backlogID)
end

local currentScore = redis.call("ZSCORE", keyAccountNormalizeSet, partitionID)
if currentScore == false or tonumber(currentScore) > normalizeTime then
  redis.call("ZADD", keyAccountNormalizeSet, normalizeTime, partitionID)
end

local currentScore = redis.call("ZSCORE", keyGlobalNormalizeSet, accountID)
if currentScore == false or tonumber(currentScore) > normalizeTime then
  redis.call("ZADD", keyGlobalNormalizeSet, normalizeTime, accountID)
end

-- Remove from backlog and update pointers
-- Note: The backlog is not yet empty, but we don't want to process it,
-- as it is outdated. That's why we don't call updateBacklogPointer which would
-- use the earliest item score as pointer instead of dropping it altogether.
redis.call("ZREM", keyShadowPartitionSet, backlogID)

-- If shadow partition has no more backlogs, update global/account pointers
if tonumber(redis.call("ZCARD", keyShadowPartitionSet)) == 0 then
  -- do not clean up shadow partition metadata yet, as we may still normalize

  redis.call("ZREM", keyGlobalShadowPartitionSet, partitionID)
  redis.call("ZREM", keyAccountShadowPartitionSet, partitionID)

  if tonumber(redis.call("ZCARD", keyAccountShadowPartitionSet)) == 0 then
    redis.call("ZREM", keyGlobalAccountShadowPartitionSet, accountID)
  end
end

return 1
