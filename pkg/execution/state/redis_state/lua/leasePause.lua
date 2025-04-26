--[[

Leases a pause atomically, if the pause is not already leased or the lease has expired.

Output:
  0: Successfully leased
  1: Already leased

]]

local leaseID = KEYS[1]

-- The current time and lease time are provided as unix timestamps in MS
local currentTime = tonumber(ARGV[1])
local leaseTTL = tonumber(ARGV[2])

if redis.call("EXISTS", leaseID) == 1 then
	-- Lease exists;  check if the lease has expired.
	local lease = tonumber(redis.call("GET", leaseID))
	if lease ~= nil and lease > currentTime then
		-- unable to lease as the lease is valid.
		return 1
	end
end

-- Add a marker denoting this item as leased.  Use second precision
-- for the expiry (leaseTTL) and as the value store millisecond precision data.
redis.call("SETEX", leaseID, leaseTTL, currentTime + (leaseTTL * 1000))

return 0
