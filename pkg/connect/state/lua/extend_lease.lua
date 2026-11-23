--[[
Output (status, lease ID):
  -3: Request leased by a different worker instance
  -2: Request leased by somebody else
  -1: Request not leased
  1: Successfully extended lease
  2: Successfully dropped lease
  3: Idempotent retry; returning the lease from the original extension

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

-- Separate TTL variables for different components
local workerTotalCapacityTTL = setTTL
local workerRequestsSetTTL = setTTL
local requestWorkerMappingTTL = requestLeaseDuration
local workerCapUnlimitedBool = (workerCapacityUnlimited == "true")

-- $include(decode_ulid_time.lua)
-- $include(get_request_lease.lua)

local requestItem = get_request_lease_item(keyRequestLease)
if requestItem == nil or requestItem == false or requestItem == cjson.null then
	return {-1, ""}
end

if requestItem.leaseID == nil or requestItem.leaseID == cjson.null then
	return {-1, ""}
end

-- unlimited workers skip this check.  a request that an older executor
-- assigned to an unlimited worker has no request-to-worker key, and the check
-- would reject every extension of that lease with -3.
local mappedWorkerID = nil
if workerCapUnlimitedBool == false then
	mappedWorkerID = redis.call("GET", requestWorkerKey)
	if mappedWorkerID ~= instanceID then
		return {-3, ""}
	end
end

if requestItem.leaseID ~= leaseID and decode_ulid_time(requestItem.leaseID) > currentTime then
	-- A successful extension can outlive the WebSocket carrying its ACK. Keep
	-- one predecessor in the existing lease value so the same worker can replay
	-- an old-SDK extension and recover the current lease ID without another key
	-- or an unbounded history.
	-- the request-to-worker key decides only when the lease names no worker.
	-- unlimited workers also have the key, and without this check the routed
	-- worker could take over a lease that another worker already extends.
	local leaseWorkerID = requestItem.workerID
	local leaseHasWorker = leaseWorkerID ~= nil and leaseWorkerID ~= cjson.null
	local retryOwnedByWorker = leaseWorkerID == instanceID
	if retryOwnedByWorker == false and leaseHasWorker == false then
		if mappedWorkerID == nil then
			mappedWorkerID = redis.call("GET", requestWorkerKey)
		end
		retryOwnedByWorker = mappedWorkerID == instanceID
	end
	if requestItem.prevLeaseID == leaseID and retryOwnedByWorker then
		return {3, requestItem.leaseID}
	end
	return {-2, ""}
end

if workerCapUnlimitedBool == false then
	-- Remove the old request from worker's set before replacing its lease.
	redis.call("ZREM", workerRequestsKey, requestID)
end

-- If new lease expiry is in the past, drop the lease
if decode_ulid_time(newLeaseID) - currentTime <= 0 then
	redis.call("DEL", keyRequestLease)

	-- Clean up the request-worker mapping
	-- Refresh TTL on the set
	if workerCapUnlimitedBool == false then
		redis.call("DEL", requestWorkerKey)
	    redis.call("EXPIRE", workerRequestsKey, workerRequestsSetTTL)
	elseif redis.call("GET", requestWorkerKey) == instanceID then
		-- unlimited workers skip the ownership check above.  the key is deleted
		-- only when it names this instance, so a stale lease holder does not
		-- erase the mapping of the worker that now runs the request.
		redis.call("DEL", requestWorkerKey)
	end
	return {2, ""}
end

-- Update the request lease item with the new lease ID
requestItem.prevLeaseID = requestItem.leaseID
requestItem.leaseID = newLeaseID
if workerCapUnlimitedBool == true then
	-- Capacity-limited workers already have a bounded request-to-worker
	-- mapping, so only unlimited workers need ownership duplicated here.
	requestItem.workerID = instanceID
else
	requestItem.workerID = nil
end
redis.call("SET", keyRequestLease, cjson.encode(requestItem), "EX", expiry)

-- If worker capacity is unlimited, we don't need to manage the set
if workerCapUnlimitedBool == true then
	-- without this the request-to-worker key expires on runs longer than its
	-- TTL and GetAssignedWorkerID returns no worker.  EXPIRE on a missing key
	-- does nothing, so a request without the key still extends.
	redis.call("EXPIRE", requestWorkerKey, requestWorkerMappingTTL)
	return {1, newLeaseID}
end

-- Worker request scores use Unix seconds; currentTime is in milliseconds.
redis.call("ZADD", workerRequestsKey, math.floor(currentTime / 1000) + requestLeaseDuration, requestID)
redis.call("EXPIRE", workerRequestsKey, workerRequestsSetTTL)

-- Update the request-worker mapping with the new TTL
redis.call("EXPIRE", requestWorkerKey, requestWorkerMappingTTL)

return {1, newLeaseID}
