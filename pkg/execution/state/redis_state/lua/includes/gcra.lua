-- performs gcra rate limiting for a given key.
-- 
-- Returns true on success, false if the key has been rate limited.
local function gcra(key, now_ms, period_ms, limit, burst)
	-- Calculate the basic variables for GCRA.
	local cost = 1                            -- everything counts as a single rqeuest

	local emission  = period_ms / math.max(limit, 1)   -- how frequently we can admit new requests
	local increment = emission * cost         -- this request's time delta
	local variance  = period_ms * (math.max(burst, 1)) -- variance takes into account bursts

	-- fetch the theoretical arrival time for equally spaced requests
	-- at exactly the rate limit
	local tat = redis.call("GET", key)
	if not tat then
		tat = now_ms
	else
		tat = tonumber(tat)
	end

	local new_tat = math.max(tat, now_ms) + increment -- add the request's cost to the theoretical arrival time.
	local allow_at_ms = new_tat - variance            -- handle bursts.
	local diff_ms = now_ms - allow_at_ms

	if diff_ms < 0 then
		return false
	end

	local expiry = (period_ms / 1000)
	redis.call("SET", key, new_tat, "EX", expiry)

	return true
end
