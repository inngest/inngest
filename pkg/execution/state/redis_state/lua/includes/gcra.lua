-- applyGCRA runs GCRA
local function applyGCRA(key, now_ms, period_ms, limit, burst, quantity)
	-- Ensure limit is always >= 1
	limit = math.max(limit, 1)

	---@type { limit: integer, ei: number, retry_at: number, dvt: number, tat: number, inc: number, ntat: number, aat: number, diff: number, retry_after: integer?, u: number, next: number?, remaining: integer?, reset_after: integer?, limited: boolean? }
	local result = {}

	-- limit defines the maximum number of requests that can be admitted at once (irrespective of current usage)
	result["limit"] = burst + 1

	-- emission defines the window size
	local emission = period_ms / limit
	result["ei"] = emission

	-- retry_at is always computed under the assumption that all
	-- remaining capacity is consumed
	result["retry_at"] = now_ms + emission

	-- dvt determines how many requests can be admitted
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
	-- When called with quantity 0, we simulate a call with quantity=1 to
	-- calculate retry after, remaining, etc. values
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

	-- ttl represents the current time until the full "limit" is allowed again
	local ttl = tat - now_ms
	result["reset_after"] = ttl

	-- currently used tokens must be calculated without burst
	local used_tokens = math.min(math.ceil(ttl / emission), limit)
	result["u"] = used_tokens

	-- requests should be allowed from the new_tat on, burst
	-- decreases the time to allowing a new request even if the original period received the maximum number of requests
	local allow_at = new_tat - dvt
	result["aat"] = allow_at

	-- allow_at must be in the past to allow the request (diff >= 0)
	local diff = now_ms - allow_at
	result["diff"] = diff

	if diff < 0 then
		if increment <= dvt then
			-- retry_after outlines when the next request would be accepted
			result["retry_after"] = -diff
			result["retry_at"] = now_ms - diff
		end

		if origQuantity > 0 then
			-- if we did want to update, we got limited
			local next = dvt - ttl
			result["next"] = next
			result["remaining"] = 0
			result["limited"] = true

			return result
		end
	end

	if origQuantity > 0 then
		-- update state to new_tat
		ttl = new_tat - now_ms
		result["reset_after"] = ttl

		used_tokens = math.min(math.ceil(ttl / emission), limit)
		result["u"] = used_tokens

		local expiry = string.format("%d", math.max(ttl / 1000, 1))
		redis.call("SET", key, new_tat, "EX", expiry)
	end

	local next = dvt - ttl
	result["next"] = next

	if next > -emission then
		local remaining = math.floor(next / emission)
		result["remaining"] = remaining
	end

	return result
end
-- performs gcra rate limiting for a given key.
--
-- Returns true on success, false if the key has been rate limited.
local function gcra(key, now_ms, period_ms, limit, burst)
	-- NOTE: we need to admit more than a single item every emission interval, as the queue
	-- does not follow a uniform arrival rate and we would throttle the majority of queue items,
	-- leading to significantly lower queue throughput
	local maxBurst = limit + burst - 1

	local res = applyGCRA(key, now_ms, period_ms, limit, maxBurst, 1)

	local used_burst = res["tat"] > now_ms

	return { not res["limited"], used_burst }
end

local function gcraUpdate(key, now_ms, period_ms, limit, burst, quantity)
	local allowRefillingAll = limit + burst - 1
	local res = applyGCRA(key, now_ms, period_ms, limit, allowRefillingAll, quantity)

	return { res["remaining"], res["retry_at"] }
end

local function gcraCapacity(key, now_ms, period_ms, limit, burst)
	return gcraUpdate(key, now_ms, period_ms, limit, burst, 0)
end
