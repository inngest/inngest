--[[

  Leases a shadow partition and pushes pointers back to prevent
  another shadow scanner from peeking the shadow partition while it's leased.

  Return values:
  0 - Leased shadow partition
  -1 - Shadow partition not found
  -2 - Shadow partition already leased

]]

local keyShadowPartitionHash             = KEYS[1]
local keyGlobalShadowPartitionSet        = KEYS[2]
local keyGlobalAccountShadowPartitionSet = KEYS[3]
local keyAccountShadowPartitionSet       = KEYS[4]

local partitionID = ARGV[1]
local accountID   = ARGV[2]
local leaseID     = ARGV[3]
local nowMS       = tonumber(ARGV[4])
local leaseExpiry = tonumber(ARGV[5]) -- in milliseconds

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
if existing.leaseID ~= nil and existing.leaseID ~= cjson.null and decode_ulid_time(existing.leaseID) > nowMS then
  return -2
end

-- Set lease ID
existing.leaseID = leaseID
redis.call("HSET", keyShadowPartitionHash, partitionID, cjson.encode(existing))

-- Push back shadow partition in global set
update_pointer_score_to(partitionID, keyGlobalShadowPartitionSet, leaseExpiry)

-- Push back shadow partition in account set + potentially push back account in global accounts set
update_account_shadow_queues(keyGlobalAccountShadowPartitionSet, keyAccountShadowPartitionSet, partitionID, accountID, leaseExpiry)

return 0
