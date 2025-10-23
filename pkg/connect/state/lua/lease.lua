--[[

Lease a request to an executor instance.
- Increments the instanceId counter with configurable TTL
- Stores lease metadata (leaseID, executorIP, instanceID)

Output:
  -1: Request already leased
  1: Successfully leased request
]]

local keyRequestLease = KEYS[1]
local keyInstanceCounter = KEYS[2]

local newLeaseID 			= ARGV[1]
local expiry				= tonumber(ARGV[2])
local currentTime			= tonumber(ARGV[3])
local executorIP			= ARGV[4]
local instanceID    =ARGV[5]
local instanceExpiry			= ARGV[6]

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
	executorIP = executorIP,
	instanceID = instanceID
}
redis.call("SET", keyRequestLease, cjson.encode(requestItem), "EX", expiry)

-- Increment instanceId counter with 60 second TTL
if keyInstanceCounter ~= nil and keyInstanceCounter ~= "" then
	redis.call("INCR", keyInstanceCounter)
	redis.call("EXPIRE", keyInstanceCounter, instanceExpiry)
end

return 1
