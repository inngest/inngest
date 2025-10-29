--[[

Refreshes the TTL on worker capacity and leases set during heartbeat.

Output:
  0: No capacity set, nothing to refresh
  1: Successfully refreshed capacity TTL (and set TTL if it exists)

ARGV[1]: TTL in seconds for both capacity and set keys
]]

local capacityKey = KEYS[1]
local workerLeasesKey = KEYS[2]
local ttl = tonumber(ARGV[1])

-- Check if capacity key exists
local capacityExists = redis.call("EXISTS", capacityKey)

-- If no capacity limit is set, nothing to refresh
if capacityExists == 0 then
  return 0
end

-- Refresh capacity key TTL
redis.call("EXPIRE", capacityKey, ttl)

-- Refresh set key TTL
--local setExists = redis.call("EXISTS", workerLeasesKey)
-- if setExists == 1 then
redis.call("EXPIRE", workerLeasesKey, ttl) -- incase the set doesn't exist, it' returns -2 but we ignore output
--end

return 1
