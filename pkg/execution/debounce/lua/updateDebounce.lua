--[[

Updates a debounce to use new data.

Return values:
- >=0 (int): OK, and the new TTL from our debounce.
- -1: Debounce is already in progress, as the queue item is leased.
- -2: Event is out of order and has no effect

]]--

local keyPtr = KEYS[1] -- fn -> debounce ptr
local keyDbc = KEYS[2] -- debounce info key
-- We need queue details to check if the debounce job is in progress (leased).  If so, we fail
-- and create a new debounce job.
local keyQueueHash = KEYS[3]

local debounceID  = ARGV[1] 
local debounce    = ARGV[2]
local ttl         = tonumber(ARGV[3])
local queueJobID  = ARGV[4]
local currentTime = tonumber(ARGV[5]) -- in ms
local eventTime   = tonumber(ARGV[6]) -- The `event.ts` value.  If this is less than the event stored in the debounce, we
                                      -- will not update the debounce as it violates the debounce order.


-- copied from get_queue_item.lua
local function get_queue_item(queueKey, queueID)
	local fetched = redis.call("HGET", queueKey, queueID)
	if fetched ~= false then
		return cjson.decode(fetched)
	end
	return nil
end

-- Check that the queue item is not leased (ie. this debounce is not in progress)
local item = get_queue_item(keyQueueHash, queueJobID)
if item == nil then
	-- The queue item was not found.  Return a new debounce.
	return -1
end
if item.leaseID ~= nil and item.leaseID ~= cjson.null and decode_ulid_time(item.leaseID) > currentTime then
	-- The debounce queue item is leased. 
	return -1
end

-- Get the debounce
local existing = redis.call("HGET", keyDbc, debounceID)
if existing ~= false then
	-- Decode the debounce, and check whether the existing event ID is > the current event ID.  If so,
	-- don't update the debounce.
	local item = cjson.decode(existing)
	if item ~= nil and item.e ~= nil and item.e.ts > eventTime then
		-- The stored event occurs after the event we're updating, so do nothing.
		return -2
	end

	-- Also, if there's an existing debounce, ensure that we respect the max timeout
	-- for the debounce.  We don't want to keep pushing a debounce out indefinitely,
	-- so if (now + new TTL in seconds) > the debounce's max time, use the debounce's
	-- max time instead.
	if item ~= nil and item.t ~= nil and item.t > 0 then
		local nextTTL = currentTime + (ttl  * 1000)
		if nextTTL > item.t then
			ttl = math.floor((item.t - currentTime) / 1000)
			if ttl <= 0 then
				-- Ensure we always use a minimum.
				ttl = 1
			end
			ttl = tonumber(ttl)
		end

		-- Also set the max within the updated debounce item.  We have to decode
		-- then re-encode the item to keep the max timeout consistent,
		-- as we do not know the max when calling update.
		--
		-- This makes updates transactional.
		local next = cjson.decode(debounce)
		next.t = item.t
		debounce = cjson.encode(next)
	end
end

-- Set the fn -> debounce ID pointer
redis.call("SETEX", keyPtr, ttl, debounceID)
redis.call("HSET", keyDbc, debounceID, debounce)

-- TODO: This should also reschedule the job directly in an atomic transaction.

return ttl
