-- performs gcra rate limiting for a given key.
--
-- Returns true on success, false if the key has been rate limited.
local function gcra(key, now_ms, period_ms, limit, burst, enableThrottleFix)
	-- Calculate the basic variables for GCRA.
	local cost = 1 -- everything counts as a single rqeuest

	local emission = period_ms / math.max(limit, 1) -- how frequently we can admit new requests
	local increment = emission * cost -- this request's time delta
	-- BUG: The variance is calculated incorrectly. We should use emission instead of period_ms.
	local variance = period_ms * (math.max(burst, 1)) -- variance takes into account bursts
	if enableThrottleFix then
		-- NOTE: This fixes the delay variation tolerance calculation
		variance = emission * (math.max(burst, 1))
	end

	-- fetch the theoretical arrival time for equally spaced requests
	-- at exactly the rate limit
	local tat = redis.call("GET", key)
	if not tat then
		tat = now_ms
	else
		tat = tonumber(tat)
	end

	local new_tat = math.max(tat, now_ms) + increment -- add the request's cost to the theoretical arrival time.
	local allow_at_ms = new_tat - variance -- handle bursts.
	local diff_ms = now_ms - allow_at_ms

	if diff_ms < 0 then
		return { false, false }
	end

	local expiry = string.format("%d", period_ms / 1000)
	redis.call("SET", key, new_tat, "EX", expiry)

	local used_burst = tat > now_ms

	return { true, used_burst }
end

local function gcraUpdate(key, now_ms, period_ms, limit, burst, quantity)
	---@type { allowed: boolean, limit: integer?, retry_after: integer?, reset_after: integer?, remaining: integer? }
	local result = {}

	-- limit defines the maximum number of requests that can be admitted at once (irrespective of current usage)
	result["limit"] = burst + 1

	-- emission defines the window size
	local emission = period_ms / math.max(limit, 1)
	result["ei"] = emission

	-- dvt defines how much burst to apply
	local dvt = emission * (burst + 1)
	result["dvt"] = dvt

	-- use existing tat or start at now_ms
	local tat = redis.call("GET", key)
	if not tat then
		tat = now_ms
	else
		tat = tonumber(tat)
	end

	result["tat"] = tat

	local origQuantity = quantity
	if quantity == 0 then
		quantity = 1
	end

	-- increment based on quantity
	local increment = quantity * emission
	result["inc"] = increment

	-- if existing tat is in the past, increment from now_ms
	local new_tat = tat + increment
	if now_ms > tat then
		new_tat = now_ms + increment
	end
	result["ntat"] = new_tat

	-- requests should be allowed from the new_tat on, burst
	-- decreases the time to allowing a new request even if the original period received the maximum number of requests
	local allow_at = new_tat - dvt
	result["aat"] = allow_at

	-- allow_at must be in the past to allow the request (diff >= 0)
	local diff = now_ms - allow_at
	result["diff"] = diff

	local ttl = 0

	if diff < 0 then
		if increment <= dvt then
			-- retry_after outlines when the next request would be accepted
			result["retry_after"] = -diff
			-- ttl represents the current time until the full "limit" is allowed again
			ttl = tat - now_ms
			result["ttl"] = ttl
		end

		if origQuantity > 0 then
			-- if we did want to update, we got limited
			local next = dvt - ttl
			result["next"] = next
			if next > -emission then
				result["remaining"] = math.floor((dvt - ttl) / emission)
			end
			result["reset_after"] = ttl

			result["limited"] = true

			return result
		end
	end

	ttl = tat - now_ms
	if origQuantity > 0 then
		-- update state to new_tat
		ttl = new_tat - now_ms
		local expiry = string.format("%d", math.max(ttl / 1000, 1))
		redis.call("SET", key, new_tat, "EX", expiry)
	end
	result["ttl"] = ttl

	local next = dvt - ttl
	if next > -emission then
		result["remaining"] = math.floor((dvt - ttl) / emission)
	end
	result["reset_after"] = ttl
	result["next"] = next

	return result
end

local function gcraCapacity(key, now_ms, period_ms, limit, burst)
	return gcraUpdate(key, now_ms, period_ms, limit, burst, 0)
end
