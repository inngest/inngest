--[[

  Drops stale normalization pointers bottom-up:
  backlog -> partition normalize set -> account normalize set -> global normalize set.

  Each pointer is only removed once the set it points to is empty, so this is safe to
  call concurrently with backlogPrepareNormalize. When force is 1, the partition pointer
  is dropped from the account normalize set regardless (e.g. partition metadata is gone).
  Account-level steps are skipped when accountID is empty.

  Return values:

  0 - Done
]]

local keyBacklogSet            = KEYS[1]
local keyPartitionNormalizeSet = KEYS[2]
local keyAccountNormalizeSet   = KEYS[3]
local keyGlobalNormalizeSet    = KEYS[4]

local backlogID   = ARGV[1]
local partitionID = ARGV[2]
local accountID   = ARGV[3]
local force       = tonumber(ARGV[4])

if backlogID ~= "" and tonumber(redis.call("ZCARD", keyBacklogSet)) == 0 then
  redis.call("ZREM", keyPartitionNormalizeSet, backlogID)
end

if accountID == "" then
  return 0
end

if partitionID ~= "" and (force == 1 or tonumber(redis.call("ZCARD", keyPartitionNormalizeSet)) == 0) then
  redis.call("ZREM", keyAccountNormalizeSet, partitionID)
end

if tonumber(redis.call("ZCARD", keyAccountNormalizeSet)) == 0 then
  redis.call("ZREM", keyGlobalNormalizeSet, accountID)
end

return 0
