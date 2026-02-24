--[[

Output:
  0: Successfully obtained, renewed, or released lease for key
// Renewal, Release Failures:
  1: existingLease does not exist
  2: existingLease already expired, cannot renew
// New Lease Failures:
  3: no leases available
]]

local leaseKey    = KEYS[1]

local currentTime     = tonumber(ARGV[1]) -- Current time, in ms, to check if existing lease expired.
local newLeaseID      = ARGV[2] -- New lease ID
local existingLeaseID = ARGV[3] -- Existing lease ID (empty string if nil)
local maxLeases       = tonumber(ARGV[4]) -- Max allowed leases

-- $include(decode_ulid_time.lua)

-- If an existing lease is provided, renew it
if existingLeaseID ~= "" then
    -- Check if the lease being renewed actually exists in the set
	if redis.call("SISMEMBER", leaseKey, existingLeaseID) == 0 then
		-- Lease doesn't exist (was removed/expired), cannot renew
		return 1
	end
	-- Ignore lease expiration checks.
	-- If an expired lease is still in the set, it indicates that there has been no membership change since and therefore it is safe to renew the lease.
    -- local existingLeaseTime = decode_ulid_time(existingLeaseID)
    -- if existingLeaseTime < currentTime then
    --     -- Lease expired, remove it and deny renewal
    --     redis.call("SREM", leaseKey, existingLeaseID)
    --     return 2
    -- end
	-- Remove the old lease ID
	redis.call("SREM", leaseKey, existingLeaseID)

	-- Add the new lease ID if provided (extends the lease), otherwise just release
	if newLeaseID ~= "" then
		redis.call("SADD", leaseKey, newLeaseID)
	end
	return 0
end

-- No existing lease - check if we can grant a new one
-- First, clean up expired leases
local members = redis.call("SMEMBERS", leaseKey)
local validLeases = 0

for _, member in ipairs(members) do
	local memberTime = decode_ulid_time(member)
	if memberTime < currentTime then
		-- Expired lease, remove it
		redis.call("SREM", leaseKey, member)
	else
		-- Valid lease, count it
		validLeases = validLeases + 1
	end
end

-- Check if we can grant a new lease
if validLeases < maxLeases then
	redis.call("SADD", leaseKey, newLeaseID)
	return 0
end

-- Max leases already granted
return 3
