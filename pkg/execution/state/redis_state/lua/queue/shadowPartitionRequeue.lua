--[[

  Returns an existing shadow partition lease and pushes pointers back to the next
  earliest backlog for the shadow partition. If no further backlog exists for shadow-scanning,
  clean up dangling pointers in global shadow partition set as well as account-level indexes.

  Return values:
  0 - Extended shadow partition lease or cleaned up partition with no backlogs
  -1 - Shadow partition not found
  -2 - Shadow partition already leased
  -3 - Shadow partition paused

]]

local keyShadowPartitionHash             = KEYS[1]
local keyGlobalShadowPartitionSet        = KEYS[2]
local keyGlobalAccountShadowPartitionSet = KEYS[3]
local keyAccountShadowPartitionSet       = KEYS[4]
local keyShadowPartitionSet              = KEYS[5]

local partitionID = ARGV[1]
local accountID   = ARGV[2]
local leaseID     = ARGV[3]
local nowMS       = tonumber(ARGV[4])
local requeueAt   = tonumber(ARGV[5])

-- $include(decode_ulid_time.lua)
-- $include(get_partition_item.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)

local existing = get_shadow_partition_item(keyShadowPartitionHash, partitionID)
if existing == nil or existing == false then
  return -1
end

-- Check for an existing lease.
if existing.leaseID ~= nil and existing.leaseID ~= cjson.null and existing.leaseID ~= leaseID and decode_ulid_time(existing.leaseID) > nowMS then
  return -2
end

-- Remove lease
existing.leaseID = nil
redis.call("HSET", keyShadowPartitionHash, partitionID, cjson.encode(existing))

-- Get earliest backlog score in shadow partition
local minScores = redis.call("ZRANGE", keyShadowPartitionSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")

-- No more backlogs, remove dangling pointers
if minScores == nil or minScores == false or minScores[2] == nil then
  redis.call("ZREM", keyGlobalShadowPartitionSet, partitionID)
  redis.call("ZREM", keyAccountShadowPartitionSet, partitionID)

  if tonumber(redis.call("ZCARD", keyAccountShadowPartitionSet)) == 0 then
    redis.call("ZREM", keyGlobalAccountShadowPartitionSet, accountID)
  end

  return 0
end

-- Push back to next earliest backlog
local earliestScore = tonumber(minScores[2])
local updateTo = earliestScore/1000

-- If we need to push back even further, override updateTo
if requeueAt ~= 0 and requeueAt > updateTo then
  updateTo = requeueAt
end

-- Push back shadow partition in global set
update_pointer_score_to(partitionID, keyGlobalShadowPartitionSet, updateTo)

-- Push back shadow partition in account set + potentially push back account in global accounts set
update_account_shadow_queues(keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, partitionID, accountID, updateTo)

return 0
