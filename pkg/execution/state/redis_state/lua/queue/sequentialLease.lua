--[[

Output:
  0: Successfully leased key
  1: Lease mismatch / already leased

]]

local leaseKey    = KEYS[1]

local currentTime     = tonumber(ARGV[1]) -- Current time, in ms, to check if existing lease expired.
local newLeaseID      = ARGV[2] -- New lease ID
local existingLeaseID = ARGV[3] -- New lease ID

-- $include(decode_ulid_time.lua)

local fetched = redis.call("GET", leaseKey)
if fetched == false or decode_ulid_time(fetched) < currentTime or fetched == existingLeaseID then
	-- Either nil, an expired key, or a release, so we're okay.
	redis.call("SET", leaseKey, newLeaseID)
	return 0
end

return 1
