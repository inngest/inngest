--[[

Removes a lease from the worker's sorted set and manages TTL/cleanup.

Output:
  0: Successfully removed, set deleted (empty)
  1: Successfully removed, set still active
  2: Set doesn't exist, nothing to remove

ARGV[1]: TTL in seconds for the set key
ARGV[2]: Request ID to remove
]]

local leasesSetKey = KEYS[1]
local leaseWorkerKey = KEYS[2]
local setTTL = tonumber(ARGV[1])
local requestID = ARGV[2]

-- Check if set exists
local setExists = redis.call("EXISTS", leasesSetKey)

-- If set doesn't exist, nothing to remove
if setExists == 0 then
  -- Still clean up the mapping in case it exists
  redis.call("DEL", leaseWorkerKey)
  return 2
end

-- Remove the specific request ID from the set
redis.call("ZREM", leasesSetKey, requestID)

-- Check if set is now empty and delete it
-- local remainingCount = redis.call("ZCARD", leasesSetKey)
-- if remainingCount == 0 then
  -- Set is empty, delete it and the mapping
--  redis.call("DEL", leasesSetKey)
--  redis.call("DEL", leaseWorkerKey)
--  return 0
--end

-- Set still has leases, refresh TTL
redis.call("EXPIRE", leasesSetKey, setTTL)
-- Delete the specific request's worker mapping
redis.call("DEL", leaseWorkerKey)
return 1
