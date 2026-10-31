local keyRequestLease = KEYS[1]
local leaseID = ARGV[1]

-- $include(get_request_lease.lua)

local requestItem = get_request_lease_item(keyRequestLease)
if requestItem == nil or requestItem == false or requestItem == cjson.null then
	return 0
end

-- Routing may outlive the initial lease. Never remove a replacement or renewal.
if requestItem.leaseID ~= leaseID then
	return 0
end

return redis.call("DEL", keyRequestLease)
