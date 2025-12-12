-- applyGCRA runs GCRA
local function applyGCRA(key, now_ms, period_ms, limit, burst, quantity, compatibility_mode)
	---@type { allowed: boolean, limit: integer?, retry_after: integer?, reset_after: integer?, remaining: integer? }
	local result = {}

	-- limit defines the maximum number of requests that can be admitted at once (irrespective of current usage)
	result["limit"] = burst + 1

	-- emission defines the window size
	local emission = period_ms / math.max(limit, 1)
	result["ei"] = emission

	-- dvt determines how many requests can be admitted
	local dvt = emission * (burst + 1)
	if compatibility_mode then
		-- compatibility mode is used to ensure we perform the legacy gcra() implementation
		dvt = period_ms * (burst + 1)
	end
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
				result["remaining"] = math.floor(next / emission)
			end
			result["reset_after"] = ttl

			result["limited"] = true

			result["retry_at"] = now_ms + (result["retry_after"] or 0)

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

	-- retry_at is always computed under the assumption that all
	-- remaining capacity is consumed
	result["retry_at"] = now_ms + emission

	local next = dvt - ttl
	if next > -emission then
		local remaining = math.floor(next / emission)
		result["remaining"] = remaining
	end
	result["reset_after"] = ttl
	result["next"] = next

	return result
end

-- performs gcra rate limiting for a given key.
--
-- Returns true on success, false if the key has been rate limited.
local function gcra(key, now_ms, period_ms, limit, burst, enableThrottleFix)
	local allowConsumingMultiple = limit + burst

	local res = applyGCRA(key, now_ms, period_ms, limit, allowConsumingMultiple - 1, 1, not enableThrottleFix)

	local used_burst = res["tat"] > now_ms

	return { res["allowed"], used_burst }
end

local function gcraUpdate(key, now_ms, period_ms, limit, burst, quantity, enableThrottleFix)
	local allowRefillingAll = limit + burst
	local res = applyGCRA(key, now_ms, period_ms, limit, allowRefillingAll - 1, quantity, not enableThrottleFix)

	return { res["remaining"], res["retry_at"] }
end

local function gcraCapacity(key, now_ms, period_ms, limit, burst, enableThrottleFix)
	return gcraUpdate(key, now_ms, period_ms, limit, burst, 0, enableThrottleFix)
end
