--[[
Output:
	-2: Request leased by somebody else
  -1: Request not leased
  1: Successfully extended lease
  2: Successfully dropped lease
]]

local keyRequestLease = KEYS[1]
local counterKey = KEYS[2]
local leaseWorkerKey = KEYS[3]

local leaseID 				= ARGV[1]
local newLeaseID 			= ARGV[2]
local expiry					= tonumber(ARGV[3])
local currentTime			= tonumber(ARGV[4])
local counterTTL			= tonumber(ARGV[5])
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
	-- If worker capacity is limited, also clean up the counter and mapping
	if isWorkerCapacityLimited == "true" then
		-- Get the old worker instance ID from the lease mapping
		local oldInstanceID = redis.call("GET", leaseWorkerKey)
		if oldInstanceID then
			-- Decrement the old counter and clean up the mapping
			local currentValue = redis.call("GET", counterKey)
			if currentValue then
				local newValue = redis.call("DECR", counterKey)
				if newValue <= 0 then
					redis.call("DEL", counterKey)
				end
			end
			redis.call("DEL", leaseWorkerKey)
		end
	end
	return 2
end

-- Get the old worker instance ID from the lease mapping (for potential counter transfer)
local oldInstanceID = nil
if isWorkerCapacityLimited == "true" then
	oldInstanceID = redis.call("GET", leaseWorkerKey)
end

requestItem.leaseID = newLeaseID
redis.call("SET", keyRequestLease, cjson.encode(requestItem), "EX", expiry)

-- Handle worker capacity counter management
if isWorkerCapacityLimited == "true" then
	-- If there was an old instance ID and it's different from the new one
	if oldInstanceID and oldInstanceID ~= instanceID then
		-- Decrement the old instance's counter
		local oldCounterKey = string.gsub(counterKey, instanceID, oldInstanceID)
		local currentValue = redis.call("GET", oldCounterKey)
		if currentValue then
			local newValue = redis.call("DECR", oldCounterKey)
			if newValue <= 0 then
				redis.call("DEL", oldCounterKey)
			else
				-- Refresh TTL on the old counter
				redis.call("EXPIRE", oldCounterKey, counterTTL)
			end
		end

		-- Increment the new instance's counter
		redis.call("INCR", counterKey)
	end

	-- Always refresh the counter TTL and update the mapping
	redis.call("EXPIRE", counterKey, counterTTL)
	redis.call("SET", leaseWorkerKey, instanceID, "EX", counterTTL)
end

return 1
