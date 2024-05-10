--[[

Output:
  0: Successfully extended lease
  1: Partition not found
  2: Partition has no lease
  3: Lease ID doesn't match (indicating someone else took the lease)
]]

local partitionKey            = KEYS[1]
local keyGlobalPartitionPtr   = KEYS[2]
local keyShardPartitionPtr    = KEYS[3]
local partitionConcurrencyKey = KEYS[4]

local partitionID    = ARGV[1]
local currentLeaseID = ARGV[2]
local newLeaseID     = ARGV[3]

-- $include(check_concurrency.lua)
-- $include(get_partition_item.lua)
-- $include(decode_ulid_time.lua)
-- $include(update_pointer_score.lua)
-- $include(has_shard_key.lua)

local nextTime = decode_ulid_time(newLeaseID)

-- check if the partition exists
local existing = get_partition_item(partitionKey, partitionID)
if existing == nil or existing == false then
  return 1
end

-- check if lease already exists or not
if existing.leaseID == nil or existing.leaseID == cjson.null then
  return 2
end
-- check if the leaseID matches or not
if existing.leaseID ~= currentLeaseID then
  return 3
end

-- update the lease
-- NOTE: should `last` also be updated???
existing.leaseID = newLeaseID

-- update item and index score
redis.call("HSET", partitionKey, partitionID, cjson.encode(existing))
-- update the score for the partition
redis.call("ZADD", keyGlobalPartitionPtr, nextTime, partitionID)

if has_shard_key(keyShardPartitionPtr) then
	update_pointer_score_to(partitionID, keyShardPartitionPtr, nextTime)
end

return 0
