--[[
Output:
  -3: Request leased by a different worker instance
  -2: Request leased by somebody else
  -1: Request not leased
  1: Successfully extended lease
  2: Successfully dropped lease

ARGV[1]: Current lease ID
ARGV[2]: New lease ID
ARGV[3]: Lease key expiry in seconds
ARGV[4]: Current time in milliseconds
ARGV[5]: Set TTL in seconds
ARGV[6]: Request lease duration in seconds
ARGV[7]: Instance ID
ARGV[8]: Worker capacity unlimited flag
ARGV[9]: Request ID

]]

local keyRequestLease = KEYS[1]
local workerRequestsKey = KEYS[2]
local requestWorkerKey = KEYS[3]

local leaseID 				= ARGV[1]
local newLeaseID 			= ARGV[2]
local expiry					= tonumber(ARGV[3])
local currentTime			= tonumber(ARGV[4])
local setTTL			= tonumber(ARGV[5])
local requestLeaseDuration			= tonumber(ARGV[6])
local instanceID			= ARGV[7]
local workerCapacityUnlimited = ARGV[8]
local requestID             = ARGV[9]
local workerCapUnlimitedBool = (workerCapacityUnlimited == "true")

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

-- this field is only set if worker capacity is limited
if workerCapUnlimitedBool == false then
	local workerInstanceID = redis.call("GET", requestWorkerKey)
	if workerInstanceID ~= instanceID then
		return -3
	end

    -- Remove the old request from worker's set
	redis.call("ZREM", workerRequestsKey, requestID)
end

-- If new lease expiry is in the past, drop the lease
if decode_ulid_time(newLeaseID) - currentTime <= 0 then
	redis.call("DEL", keyRequestLease)

	-- Clean up the request-worker mapping
	-- Refresh TTL on the set
	if workerCapUnlimitedBool == false then
		redis.call("DEL", requestWorkerKey)
	    redis.call("EXPIRE", workerRequestsKey, setTTL)
	end
	return 2
end

-- Update the request lease item with the new lease ID
requestItem.leaseID = newLeaseID
redis.call("SET", keyRequestLease, cjson.encode(requestItem), "EX", expiry)

-- If worker capacity is unlimited, we don't need to manage the set
if workerCapUnlimitedBool == true then
	return 1
end

-- Add to the new lease to the worker's set
redis.call("ZADD", workerRequestsKey, currentTime + requestLeaseDuration, requestID)
redis.call("EXPIRE", workerRequestsKey, setTTL)

-- Update the request-worker mapping
redis.call("SET", requestWorkerKey, instanceID, "EX", requestLeaseDuration)

return 1
