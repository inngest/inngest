--[[

  Removes backlog pointer from normalize partition.
  Will update normalize pointers accordingly.

  Status values:

  1 - Updated pointers
  -1 - Backlog is not empty
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

-- If there are still items in the backlog, return error code
if tonumber(redis.call("ZCARD", keyBacklogSet)) > 0 then
  return -1
end

-- Remove from normalize set
redis.call("ZREM", keyPartitionNormalizeSet, backlogID)

if redis.call("ZCARD", keyPartitionNormalizeSet) == 0 then
  redis.call("ZREM", keyAccountNormalizeSet, partitionID)
end

if redis.call("ZCARD", keyAccountNormalizeSet) == 0 then
  redis.call("ZREM", keyGlobalNormalizeSet, accountID)
end

return 1
