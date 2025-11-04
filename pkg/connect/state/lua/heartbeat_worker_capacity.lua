--[[

Refreshes the TTL on worker capacity and leases set during heartbeat.

Output:
  0: No capacity set, nothing to refresh
  1: Successfully refreshed capacity TTL (and set TTL if it exists)

ARGV[1]: TTL in seconds for both capacity and set keys
]]

local workerTotalCapacityKey = KEYS[1]
local workerRequestsKey = KEYS[2]
local ttl = tonumber(ARGV[1])

-- Separate TTL variables for different components
local workerTotalCapacityTTL = ttl
local workerRequestsSetTTL = ttl

-- Check if capacity key exists
local capacityExists = redis.call("EXISTS", workerTotalCapacityKey)

-- If no capacity limit is set, nothing to refresh
if capacityExists == 0 then
  return 0
end

-- Refresh capacity key TTL
redis.call("EXPIRE", workerTotalCapacityKey, workerTotalCapacityTTL)

-- Refresh set key TTL
redis.call("EXPIRE", workerRequestsKey, workerRequestsSetTTL) -- incase the set doesn't exist, it' returns -2 but we ignore output

return 1
