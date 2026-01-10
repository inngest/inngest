--- key specifies the rate limit key (a fully-qualified Redis key)
---@type string
local key = KEYS[1]

--- idempotencyKey is always defined but may be empty in case no ttl is set
---@type string
local idempotencyKey = KEYS[2]

--- now_ns is the current time in nanoseconds
---@type integer
local now_ns = tonumber(ARGV[1])

--- period_ns is the rate limiting period in nanoseconds
---@type integer
local period_ns = tonumber(ARGV[2])

--- limit is the number of allowed requests within the period
---@type integer
local limit = tonumber(ARGV[3])

--- burst is the optional burst capacity
---@type integer
local burst = tonumber(ARGV[4])

--- idempotencyTTL is an optional idempotency period
---@type integer
local idempotencyTTL = tonumber(ARGV[5])

---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param burst integer
---@param quantity integer
local function rateLimit(key, now_ns, period_ns, limit, burst, quantity)
	-- Ensure limit is always >= 1
	limit = math.max(limit, 1)

	---@type { limit: integer, ei: number, retry_at: number, dvt: number, tat: number, inc: number, ntat: number, aat: number, diff: number, retry_after: integer?, u: number, next: number?, remaining: integer?, reset_after: integer?, limited: boolean? }
	local result = {}

	-- limit defines the maximum number of requests that can be admitted at once (irrespective of current usage)
	result["limit"] = burst + 1

	-- emission defines the window size
	local emission = period_ns / limit
	result["ei"] = emission

	-- retry_at is always computed under the assumption that all
	-- remaining capacity is consumed
	result["retry_at"] = now_ns + emission

	-- dvt determines how many requests can be admitted
	local dvt = emission * (burst + 1)
	result["dvt"] = dvt

	-- use existing tat or start at now_ms
	local tat = redis.call("GET", key)
	if not tat then
		tat = now_ns
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
	if now_ns > tat then
		new_tat = now_ns + increment
	end
	result["ntat"] = new_tat

	-- ttl represents the current time until the full "limit" is allowed again
	local ttl = tat - now_ns
	result["reset_after"] = ttl

	-- currently used tokens must be calculated without burst
	local used_tokens = math.min(math.ceil(ttl / emission), limit)
	result["u"] = used_tokens

	-- requests should be allowed from the new_tat on, burst
	-- decreases the time to allowing a new request even if the original period received the maximum number of requests
	local allow_at = new_tat - dvt
	result["aat"] = allow_at

	-- allow_at must be in the past to allow the request (diff >= 0)
	local diff = now_ns - allow_at
	result["diff"] = diff

	if diff < 0 then
		if increment <= dvt then
			-- retry_after outlines when the next request would be accepted
			result["retry_after"] = -diff
			result["retry_at"] = now_ns - diff
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
		ttl = new_tat - now_ns
		result["reset_after"] = ttl

		used_tokens = math.min(math.ceil(ttl / emission), limit)
		result["u"] = used_tokens

		local expiry = string.format("%d", math.max(ttl / 1000000000, 1))
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

--- gcraUpdate is the second half of a nanosecond-precision GCRA implementation, used for updating rate limit state.
---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param burst integer
---@param quantity integer the number of requests to admit at once
local function gcraUpdate(key, now_ns, period_ns, limit, burst, quantity)
	local res = rateLimit(key, now_ns, period_ns, limit, burst, quantity)

	return { res["remaining"], res["retry_at"] }
end

--- gcraCapacity is the first half of a nanosecond-precision GCRA implementation. This method calculates the number of requests that can be admitted in the current period.
---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param burst integer
---@return integer[] returns a 3-tuple of remaining capacity, retry at, and current usage
local function gcraCapacity(key, now_ns, period_ns, limit, burst)
	return gcraUpdate(key, now_ns, period_ns, limit, burst, 0)
end

-- If idempotency key is set, do not perform check again
if idempotencyTTL > 0 and redis.call("EXISTS", idempotencyKey) == 1 then
	return { 2, 0 }
end

-- Check if capacity > 0
local res = gcraCapacity(key, now_ns, period_ns, limit, burst)
if res[1] > 0 then
	-- Not rate limited, perform the update
	gcraUpdate(key, now_ns, period_ns, limit, burst, 1)

	if idempotencyTTL > 0 then
		redis.call("SET", idempotencyKey, tostring(now_ns), "EX", idempotencyTTL)
	end

	return { 1, 0 }
else
	-- Rate limited, return retry time
	return { 0, res[2] }
end
