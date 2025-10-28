--[[
Output:
  -2: Request leased by somebody else
  -1: Request not leased
  1: Successfully extended lease
  2: Successfully dropped lease
]]

local keyRequestLease = KEYS[1]
local leasesSetKey = KEYS[2]
local leaseWorkerKey = KEYS[3]

local leaseID 				= ARGV[1]
local newLeaseID 			= ARGV[2]
local expiry					= tonumber(ARGV[3])
local currentTime			= tonumber(ARGV[4])
local setTTL			= tonumber(ARGV[5])
local instanceID			= ARGV[6]
local isWorkerCapacityLimited = ARGV[7]

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
	-- If worker capacity is limited, also clean up the set and mapping
	if isWorkerCapacityLimited == "true" then
		-- Get the old worker instance ID from the lease mapping
		local oldInstanceID = redis.call("GET", leaseWorkerKey)
		if oldInstanceID then
			-- Remove from the old instance's set
			local oldLeasesSetKey = string.gsub(leasesSetKey, instanceID, oldInstanceID)

			-- Get request ID from the request item
			local requestIDToRemove = requestItem.requestID
			if not requestIDToRemove and requestItem.Request then
				requestIDToRemove = requestItem.Request.id
			end

			if requestIDToRemove then
				-- Remove the request from old worker's set
				redis.call("ZREM", oldLeasesSetKey, requestIDToRemove)

				-- Check if old set is now empty
				local oldRemainingCount = redis.call("ZCARD", oldLeasesSetKey)
				if oldRemainingCount == 0 then
					redis.call("DEL", oldLeasesSetKey)
				end
			end
			redis.call("DEL", leaseWorkerKey)
		end
	end
	return 2
end

-- Get the old worker instance ID from the lease mapping (for potential set transfer)
local oldInstanceID = nil
if isWorkerCapacityLimited == "true" then
	oldInstanceID = redis.call("GET", leaseWorkerKey)
end

requestItem.leaseID = newLeaseID
redis.call("SET", keyRequestLease, cjson.encode(requestItem), "EX", expiry)

-- Handle worker capacity set management
if isWorkerCapacityLimited == "true" then
	-- Get request ID from the request item
	local requestIDToManage = requestItem.requestID
	if not requestIDToManage and requestItem.Request then
		requestIDToManage = requestItem.Request.id
	end

	if not requestIDToManage then
		-- This shouldn't happen, but handle gracefully
		return -1
	end

	-- If there was an old instance ID and it's different from the new one
	if oldInstanceID and oldInstanceID ~= instanceID then
		-- Remove from the old instance's set
		local oldLeasesSetKey = string.gsub(leasesSetKey, instanceID, oldInstanceID)

		-- Remove the request from old worker's set
		redis.call("ZREM", oldLeasesSetKey, requestIDToManage)

		-- Check if old set is now empty
		local oldRemainingCount = redis.call("ZCARD", oldLeasesSetKey)
		if oldRemainingCount == 0 then
			redis.call("DEL", oldLeasesSetKey)
		else
			-- Refresh TTL on the old set
			redis.call("EXPIRE", oldLeasesSetKey, setTTL)
		end

		-- Add to the new instance's set
		local newExpirationTime = decode_ulid_time(newLeaseID)
		redis.call("ZADD", leasesSetKey, newExpirationTime, requestIDToManage)
		redis.call("EXPIRE", leasesSetKey, setTTL)
	else
		-- Same worker, just update the expiration time in the set
		local newExpirationTime = decode_ulid_time(newLeaseID)
		redis.call("ZADD", leasesSetKey, newExpirationTime, requestIDToManage)
		redis.call("EXPIRE", leasesSetKey, setTTL)
	end

	-- Always update the mapping
	redis.call("SET", leaseWorkerKey, instanceID, "EX", setTTL)
end

return 1