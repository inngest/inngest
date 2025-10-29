--[[
Output:
  -3: Request leased by a different worker instance
  -2: Request leased by somebody else
  -1: Request not leased
  1: Successfully extended lease
  2: Successfully dropped lease
]]

local keyRequestLease = KEYS[1]
local workerLeasesSetKey = KEYS[2]
local leaseWorkerKey = KEYS[3]

local leaseID 				= ARGV[1]
local newLeaseID 			= ARGV[2]
local expiry					= tonumber(ARGV[3])
local currentTime			= tonumber(ARGV[4])
local setTTL			= tonumber(ARGV[5])
local instanceID			= ARGV[6]
local workerCapacityUnlimited = ARGV[7]
local workerCapUnlimited = (workerCapacityUnlimited == "true")

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
if workerCapacityUnlimited == false then
	local workerInstanceID = redis.call("GET", leaseWorkerKey)
	if workerInstanceID ~= instanceID then
		return -3
	end

    -- Remove the old request from worker's set
	redis.call("ZREM", workerLeasesSetKey, leaseID)
end

-- If new lease expiry is in the past, drop the lease
if decode_ulid_time(newLeaseID) - currentTime <= 0 then
	redis.call("DEL", keyRequestLease)

	-- Clean up the lease-worker mapping
	-- Refresh TTL on the set
	if workerCapacityUnlimited == false then
		redis.call("DEL", leaseWorkerKey)
	    redis.call("EXPIRE", workerLeasesSetKey, setTTL)
	end
	return 2
end

-- Update the request lease item with the new lease ID
requestItem.leaseID = newLeaseID
redis.call("SET", keyRequestLease, cjson.encode(requestItem), "EX", expiry)

-- If worker capacity is unlimited, we don't need to manage the set
if workerCapUnlimited == true then
	return 1
end

-- Add to the new lease to the worker's set
redis.call("ZADD", workerLeasesSetKey, decode_ulid_time(newLeaseID), newLeaseID)
redis.call("EXPIRE", workerLeasesSetKey, setTTL)

-- Update the lease-worker mapping
redis.call("SET", leaseWorkerKey, instanceID, "EX", setTTL)

return 1