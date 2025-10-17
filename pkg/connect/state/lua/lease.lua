--[[

Output:
  -1: Request already leased
  1: Successfully leased request
]]

local keyRequestLease = KEYS[1]

local newLeaseID 			= ARGV[1]
local expiry				= tonumber(ARGV[2])
local currentTime			= tonumber(ARGV[3])
local executorIP			= ARGV[4]

-- $include(decode_ulid_time.lua)
-- $include(get_request_lease.lua)

local requestItem = get_request_lease_item(keyRequestLease)

-- Case 1: Request is actively leased
if requestItem ~= nil and requestItem ~= false and requestItem.leaseID ~= nil and requestItem.leaseID ~= cjson.null and decode_ulid_time(requestItem.leaseID) > currentTime then
  -- Maintain same lease ID with current executor IP
  requestItem = {
    leaseID = requestItem.leaseID,
    executorIP = executorIP
  }

  -- Update request lease key, maintain TTL
  redis.call("SET", keyRequestLease, cjson.encode(requestItem), "KEEPTTL")
	return -1
end

-- Case 2: Lease does not exist
requestItem = {
	leaseID = newLeaseID,
	executorIP = executorIP
}
redis.call("SET", keyRequestLease, cjson.encode(requestItem), "EX", expiry)
return 1
