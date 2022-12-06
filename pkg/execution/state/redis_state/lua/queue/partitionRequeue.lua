--[[

  Requeues a partition at a specific time.
  will take into account the new priority when weighted sampling items to
  work on.

  Return values:
  0 - Updated priority
  1 - Partition not found

]]

local partitionKey   = KEYS[1]
local partitionIndex = KEYS[2]

local workflowID = ARGV[1]
local at         = tonumber(ARGV[2]) -- time in seconds

-- $include(get_partition_item.lua)
local existing = get_partition_item(partitionKey, workflowID)
if existing == nil then
	return 1
end

existing.at = at
existing.leaseID = nil
redis.call("HSET", partitionKey, workflowID, cjson.encode(existing))
redis.call("ZADD", partitionIndex, at, workflowID)

return 0
