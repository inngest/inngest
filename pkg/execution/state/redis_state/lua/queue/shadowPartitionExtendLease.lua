--[[

  Extends an existing shadow partition lease and pushes pointers back to prevent
  another shadow scanner from peeking the shadow partition while it's leased.

  Return values:
  0 - Extended shadow partition lease
  -1 - Shadow partition not found
  -2 - Shadow partition not leased
  -3 - Shadow partition already leased

]]

local keyShadowPartitionHash             = KEYS[1]
local keyGlobalShadowPartitionSet        = KEYS[2]
local keyGlobalAccountShadowPartitionSet = KEYS[3]
local keyAccountShadowPartitionSet       = KEYS[4]

local partitionID = ARGV[1]
local accountID   = ARGV[2]
local leaseID     = ARGV[3]
local newLeaseID  = ARGV[4]
local nowMS       = tonumber(ARGV[5])
local leaseExpiry = tonumber(ARGV[6]) -- in seconds, as partition score

-- $include(decode_ulid_time.lua)
-- $include(get_partition_item.lua)
-- $include(update_pointer_score.lua)
-- $include(ends_with.lua)
-- $include(update_account_queues.lua)

local existing = get_shadow_partition_item(keyShadowPartitionHash, partitionID)
if existing == nil or existing == false then
  return -1
end

-- If shadow partition is not leased, exit early
if existing.leaseID == false or existing.leaseID == nil or existing.leaseID == cjson.null then
  return -2
end

-- If shadow partition is actively leased by another process, exit early
if existing.leaseID ~= leaseID and decode_ulid_time(existing.leaseID) > nowMS then
  return -3
end


-- Update to new lease ID
existing.leaseID = newLeaseID
redis.call("HSET", keyShadowPartitionHash, partitionID, cjson.encode(existing))

-- Push back shadow partition in global set
update_pointer_score_to(partitionID, keyGlobalShadowPartitionSet, leaseExpiry)

-- Push back shadow partition in account set + potentially push back account in global accounts set
update_account_shadow_queues(keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, partitionID, accountID, leaseExpiry)

return 0
