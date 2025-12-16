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

--- toInteger ensures a value is stored as an integer to prevent Redis scientific notation serialization
---@param value number
---@return integer
local function toInteger(value)
	return math.floor(value + 0.5) -- Round to nearest integer
end

--- clampTat ensures tat value is within reasonable bounds to prevent corruption issues
---@param tat number
---@param now_ns integer
---@param period_ns integer
---@param delay_variation_tolerance number
---@return integer
local function clampTat(tat, now_ns, period_ns, delay_variation_tolerance)
	local max_reasonable_tat = now_ns + period_ns + delay_variation_tolerance
	local min_reasonable_tat = now_ns - period_ns -- Allow some past values for clock skew

	if tat > max_reasonable_tat then
		return toInteger(max_reasonable_tat)
	elseif tat < min_reasonable_tat then
		return toInteger(now_ns) -- Reset to current time if too far in past
	else
		return toInteger(tat)
	end
end

--- retrieveAndNormalizeTat gets the TAT value from Redis and normalizes it if needed
---@param key string
---@param now_ns integer
---@param period_ns integer
---@param delay_variation_tolerance number
---@return integer
local function retrieveAndNormalizeTat(key, now_ns, period_ns, delay_variation_tolerance)
	local tat = redis.call("GET", key)
	if not tat then
		return now_ns
	end

	local raw_tat = tonumber(tat)
	if not raw_tat then
		return now_ns -- Reset if tonumber failed
	end

	local clamped_tat = clampTat(raw_tat, now_ns, period_ns, delay_variation_tolerance)
	-- If value was clamped, commit the normalization immediately
	if raw_tat ~= clamped_tat then
		redis.call("SET", key, clamped_tat, "KEEPTTL")
	end

	return clamped_tat
end

--- gcraCapacity is the first half of a nanosecond-precision GCRA implementation. This method calculates the number of requests that can be admitted in the current period.
---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param burst integer
---@return integer[] returns a 3-tuple of remaining capacity, retry at, and current usage
local function gcraCapacity(key, now_ns, period_ns, limit, burst)
	-- Handle zero limit case - immediately rate limit
	if limit == 0 then
		return { 0, now_ns + period_ns, 0 }
	end

	-- Match throttled library calculations exactly
	-- emissionInterval = quota.MaxRate.period / limit
	local emission_interval = period_ns / limit

	-- delayVariationTolerance = emission_interval * (maxBurst + 1)
	-- In throttled: immediate capacity = MaxBurst + 1
	local total_capacity = burst + 1
	local delay_variation_tolerance = emission_interval * total_capacity

	-- retrieve and normalize theoretical arrival time
	local tat = retrieveAndNormalizeTat(key, now_ns, period_ns, delay_variation_tolerance)

	-- Calculate current usage (consumed tokens) independently of burst capacity
	local used_tokens = 0
	if tat > now_ns then
		local consumed_time = tat - now_ns
		used_tokens = math.min(math.ceil(consumed_time / emission_interval), limit)
	end

	-- Calculate what the next TAT would be if we processed this request (quantity = 1)
	local increment = 1 * emission_interval
	local new_tat
	if now_ns > tat then
		new_tat = now_ns + increment
	else
		new_tat = tat + increment
	end

	-- Block the request if the next permitted time is in the future
	-- allowAt = newTat.Add(-delayVariationTolerance)
	local allow_at = new_tat - delay_variation_tolerance
	local diff = now_ns - allow_at

	if diff < 0 then
		-- We are rate limited - calculate retry time
		-- RetryAfter = -diff (when diff is negative)
		return { 0, allow_at, used_tokens }
	else
		-- Not rate limited - calculate remaining capacity
		-- Use current TAT instead of new_tat since we haven't consumed the token yet
		-- next = delayVariationTolerance - ttl, where ttl = currentTat.Sub(now)
		local current_ttl = math.max(tat - now_ns, 0)
		local next = delay_variation_tolerance - current_ttl
		local remaining = 0
		if next > -emission_interval then
			remaining = math.floor(next / emission_interval)
		end

		-- Calculate when the next unit will be available after consuming all remaining capacity
		local new_tat_after_consumption = math.max(tat, now_ns) + remaining * emission_interval
		local next_available_at_ns = new_tat_after_consumption - delay_variation_tolerance + emission_interval

		return { remaining, toInteger(next_available_at_ns), used_tokens }
	end
end

--- gcraUpdate is the second half of a nanosecond-precision GCRA implementation, used for updating rate limit state.
---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param capacity integer the number of requests to admit at once
---@param burst integer
local function gcraUpdate(key, now_ns, period_ns, limit, capacity, burst)
	-- Handle zero limit case - no update needed since we always rate limit
	if limit == 0 then
		return
	end

	-- calculate emission interval (tau) - time between each token
	-- This matches throttled library: quota.MaxRate.period
	local emission_interval = period_ns / limit

	-- Calculate delay_variation_tolerance for bounds checking
	local total_capacity = (burst or 0) + 1
	local delay_variation_tolerance = emission_interval * total_capacity

	-- retrieve and normalize theoretical arrival time
	local tat = retrieveAndNormalizeTat(key, now_ns, period_ns, delay_variation_tolerance)

	-- calculate next theoretical arrival time
	-- This matches throttled library logic: tat.Add(increment) where increment = quantity * emissionInterval
	local increment = math.max(capacity, 1) * emission_interval
	local new_tat
	if now_ns > tat then
		new_tat = now_ns + increment
	else
		new_tat = tat + increment
	end

	if capacity > 0 then
		-- Clamp new_tat to reasonable bounds and ensure integer storage
		local clamped_tat = clampTat(new_tat, now_ns, period_ns, delay_variation_tolerance)

		-- Calculate TTL like throttled library: ttl = newTat.Sub(now)
		local ttl_ns = clamped_tat - now_ns
		local ttl_seconds = math.ceil(ttl_ns / 1000000000) -- Convert nanoseconds to seconds
		redis.call("SET", key, clamped_tat, "EX", ttl_seconds)
	end
end

-- If idempotency key is set, do not perform check again
if idempotencyTTL > 0 and redis.call("EXISTS", idempotencyKey) == 1 then
	return { 2, 0 }
end

-- Check if capacity > 0
local res = gcraCapacity(key, now_ns, period_ns, limit, burst)
if res[1] > 0 then
	-- Not rate limited, perform the update
	gcraUpdate(key, now_ns, period_ns, limit, 1, burst)

	if idempotencyTTL > 0 then
		redis.call("SET", idempotencyKey, tostring(now_ns), "EX", idempotencyTTL)
	end

	return { 1, 0 }
else
	-- Rate limited, return retry time
	return { 0, res[2] }
end
