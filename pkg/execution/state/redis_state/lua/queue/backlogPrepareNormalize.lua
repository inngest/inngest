--[[

  Removes backlog pointer from shadow partition and into normalize partition.
  Will update shadow partition pointers accordingly.

  Returns a tuple of {status, items_in_backlog}

  Status values:

  1 - Moved backlog to normalize set
  -1 - Fewer items than minimum
]]

local keyBacklogSet                      = KEYS[1]
local keyShadowPartitionSet              = KEYS[2]
local keyGlobalShadowPartitionSet        = KEYS[3]
local keyGlobalAccountShadowPartitionSet = KEYS[4]
local keyAccountShadowPartitionSet       = KEYS[5]

local keyGlobalNormalizeSet              = KEYS[6]
local keyAccountNormalizeSet             = KEYS[7]
local keyPartitionNormalizeSet           = KEYS[8]

local backlogID             = ARGV[1]
local partitionID           = ARGV[2]
local accountID             = ARGV[3]
local normalizeTime         = tonumber(ARGV[4])
local normalizeAsyncMinimum = tonumber(ARGV[5])

-- If there's a minimum number of backlog items required to normalize asynchronously,
-- we do not need to move backlog pointers to the normalization ZSETs but can just normalize
-- in the same shadow scanner loop iteration.
local backlogCount = redis.call("ZCARD", keyBacklogSet)
if normalizeAsyncMinimum > 0 and backlogCount ~= false and backlogCount ~= nil and backlogCount < normalizeAsyncMinimum then
  return { -1, backlogCount }
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

redis.call("ZREM", keyShadowPartitionSet, backlogID)

-- If shadow partition has no more backlogs, update global/account pointers
if tonumber(redis.call("ZCARD", keyShadowPartitionSet)) == 0 then
  redis.call("ZREM", keyGlobalShadowPartitionSet, partitionID)
  redis.call("ZREM", keyAccountShadowPartitionSet, partitionID)

  if tonumber(redis.call("ZCARD", keyAccountShadowPartitionSet)) == 0 then
    redis.call("ZREM", keyGlobalAccountShadowPartitionSet, accountID)
  end
end

return { 1, backlogCount }
