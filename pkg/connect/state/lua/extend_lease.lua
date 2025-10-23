--[[

Extend an existing request lease.
- Refreshes the instanceId counter TTL
- Updates lease metadata with new leaseID

Output:
  -2: Request leased by somebody else
  -1: Request not leased
  1: Successfully extended lease
  2: Successfully dropped lease
]]

local keyRequestLease = KEYS[1]
local keyInstanceCounter = KEYS[2]

local leaseID 				= ARGV[1]
local newLeaseID 			= ARGV[2]
local expiry					= tonumber(ARGV[3])
local currentTime			= tonumber(ARGV[4])
local instanceExpiry		= tonumber(ARGV[5])

-- $include(decode_ulid_time.lua)
-- $include(get_request_lease.lua)

local requestItem = get_request_lease_item(keyRequestLease)
if requestItem == nil or requestItem == false or requestItem == cjson.null then
	return -1
end

if requestItem.leaseID == nil or requestItem.leaseID == cjson.null then
	return -1
end

if requestItem.leaseID ~= leaseID and decode_ulid_time(requestItem.leaseID) > currentTime then
	return -2
end

-- If new lease expiry is in the past, drop the lease
if decode_ulid_time(newLeaseID) - currentTime <= 0 then
	redis.call("DEL", keyRequestLease)
	return 2
end

requestItem.leaseID = newLeaseID
redis.call("SET", keyRequestLease, cjson.encode(requestItem), "EX", expiry)

-- Refresh instanceId counter TTL
if keyInstanceCounter ~= nil and keyInstanceCounter ~= "" then
	redis.call("EXPIRE", keyInstanceCounter, instanceExpiry)
end

return 1
