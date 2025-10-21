--[[

Add a lease for a request to a worker instance.

Output:
  0: Successfully added lease
  1: Worker at capacity

]]

local capacityKey = KEYS[1]
local leasesKey = KEYS[2]

local requestID = ARGV[1]

-- Get the capacity limit
local capacity = tonumber(redis.call("GET", capacityKey))
if not capacity then
	-- No limit set, add the lease
	redis.call("SADD", leasesKey, requestID)
	redis.call("EXPIRE", leasesKey, 86400)
	return 0
end

-- Check current number of leases
local current = redis.call("SCARD", leasesKey)
if current >= capacity then
	return 1  -- At capacity
end

-- Add the lease
redis.call("SADD", leasesKey, requestID)
redis.call("EXPIRE", leasesKey, 86400)
return 0
