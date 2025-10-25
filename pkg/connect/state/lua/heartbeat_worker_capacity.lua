--[[

Refreshes the TTL on worker capacity and counter keys during heartbeat.

Output:
  0: No capacity set, nothing to refresh
  1: Successfully refreshed capacity TTL (and counter TTL if counter exists)

ARGV[1]: TTL in seconds for both capacity and counter keys
]]

local capacityKey = KEYS[1]
local counterKey = KEYS[2]
local ttl = tonumber(ARGV[1])

-- Check if capacity key exists
local capacityExists = redis.call("EXISTS", capacityKey)

-- If no capacity limit is set, nothing to refresh
if capacityExists == 0 then
  return 0
end

-- Refresh capacity key TTL
redis.call("EXPIRE", capacityKey, ttl)

-- Refresh counter key TTL if it exists
-- this check can be skipped but it will create the value if it doesn't exist
local counterExists = redis.call("EXISTS", counterKey)
if counterExists == 1 then
  redis.call("EXPIRE", counterKey, ttl)
end

return 1
