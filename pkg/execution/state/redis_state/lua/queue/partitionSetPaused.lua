--[[

  Sets the "off" boolean on the given partition.

  Return values:
  0 - Updated "off" boolean
  1 - Partition not found

]]

local partitionKey = KEYS[1]

local workflowID   = ARGV[1]
local isPaused     = tonumber(ARGV[2])

-- $include(get_partition_item.lua)
local existing = get_partition_item(partitionKey, workflowID)
if existing == nil then
	return 1
end

if isPaused == 1 then
    existing.off = true
else
    existing.off = false
end
redis.call("HSET", partitionKey, workflowID, cjson.encode(existing))

return 0
