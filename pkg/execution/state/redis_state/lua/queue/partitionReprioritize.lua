--[[

  Reprioritizes a partition within a queue.  This ensures that PartitionPeek
  will take into account the new priority when weighted sampling items to
  work on.

  Return values:
  0 - Updated priority
  1 - Partition not found

]]

local partitionKey = KEYS[1]

local workflowID   = ARGV[1]
local priority     = tonumber(ARGV[2])

-- $include(get_partition_item.lua)
local existing = get_partition_item(partitionKey, workflowID)
if existing == nil then
	return 1
end

existing.p = priority
redis.call("HSET", partitionKey, workflowID, cjson.encode(existing))

return 0
