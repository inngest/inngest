--[[

Output:
  0: Request is not leased
  1: Lease exists but expired
 	2: Request is actively leased
]]

local keyRequestLease = KEYS[1]

local currentTime			= tonumber(ARGV[1])

-- $include(decode_ulid_time.lua)
-- $include(get_request_lease.lua)

-- Case 0: Request lease item does not exist
local requestItem = get_request_lease_item(keyRequestLease)
if requestItem == nil or requestItem == false or requestItem == cjson.null then
	return 0
end

-- Case 1: Request is actively leased
if requestItem.leaseID ~= nil and requestItem.leaseID ~= cjson.null and decode_ulid_time(requestItem.leaseID) > currentTime then
	return 2
end

-- Case 2: Lease expired
return 1
