--[[

Updates a debounce to use new data.

Return values:
- >=0 (int): OK, and the new TTL from our debounce.
- -1: Debounce is already in progress, as the queue item is leased.
- -2: Event is out of order and has no effect
- -3: Debounce queue item is not found.
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

-- This table is used when decoding ulid timestamps.
local ulidMap = { ["0"] = 0, ["1"] = 1, ["2"] = 2, ["3"] = 3, ["4"] = 4, ["5"] = 5, ["6"] = 6, ["7"] = 7, ["8"] = 8, ["9"] = 9, ["A"] = 10, ["B"] = 11, ["C"] = 12, ["D"] = 13, ["E"] = 14, ["F"] = 15, ["G"] = 16, ["H"] = 17, ["J"] = 18, ["K"] = 19, ["M"] = 20, ["N"] = 21, ["P"] = 22, ["Q"] = 23, ["R"] = 24, ["S"] = 25, ["T"] = 26, ["V"] = 27, ["W"] = 28, ["X"] = 29, ["Y"] = 30, ["Z"] = 31 }

-- decode_ulid_time decodes a ULID into a ms epoch
local function decode_ulid_time(s)
        if #s < 10 then
                return 0
        end

        -- Take first 10 characters of the ULID, which is the time portion.
        s = string.sub(s, 1, 10)
        local rev = tostring(s.reverse(s))
        local time = 0
        for i = 1, #rev do
                time = time + (ulidMap[string.sub(rev, i, i)] * math.pow(32, i-1))
        end
        return time
end


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
	-- The queue item was not found. return not found but set the debounce in the hash map
  -- for lookup
  redis.call("SETEX", keyPtr, ttl, debounceID)
  redis.call("HSET", keyDbc, debounceID, debounce)
  return -3
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
