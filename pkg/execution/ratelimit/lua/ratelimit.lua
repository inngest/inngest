--- key specifies the rate limit key (a fully-qualified Redis key)
---@type string
local key = KEYS[1]

--- now_ns is the current time in nanoseconds
---@type integer
local now_ns = tonumber(ARGV[2])

--- period_ns is the rate limiting period in nanoseconds
---@type integer
local period_ns = tonumber(ARGV[3])

--- limit is the number of allowed requests within the period
---@type integer
local limit = tonumber(ARGV[4])

--- burst is the optional burst capacity
---@type integer
local burst = tonumber(ARGV[5])

--- gcraCapacity is the first half of a nanosecond-precision GCRA implementation. This method calculates the number of requests that can be admitted in the current period.
---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param burst integer
---@return integer[] returns a 2-tuple of remaining capacity and retry at
local function gcraCapacity(key, now_ns, period_ns, limit, burst)
	-- Handle zero limit case - immediately rate limit
	if limit == 0 then
		return { 0, now_ns + period_ns }
	end

	-- Match throttled library calculations exactly
	-- emissionInterval = quota.MaxRate.period / limit
	local emission_interval = period_ns / limit

	-- delayVariationTolerance = emission_interval * (maxBurst + 1)
	-- In throttled: immediate capacity = MaxBurst + 1
	local total_capacity = burst + 1
	local delay_variation_tolerance = emission_interval * total_capacity

	-- retrieve theoretical arrival time
	local tat = redis.call("GET", key)
	if not tat then
		tat = now_ns
	else
		tat = tonumber(tat)
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
		return { 0, allow_at }
	else
		-- Not rate limited - calculate remaining capacity
		-- next = delayVariationTolerance - ttl, where ttl = newTat.Sub(now)
		local ttl = new_tat - now_ns
		local next = delay_variation_tolerance - ttl
		local remaining = 0
		if next > -emission_interval then
			remaining = math.floor(next / emission_interval)
		end
		return { remaining, 0 }
	end
end

--- gcraUpdate is the second half of a nanosecond-precision GCRA implementation, used for updating rate limit state.
---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param capacity integer the number of requests to admit at once
local function gcraUpdate(key, now_ns, period_ns, limit, capacity)
	-- Handle zero limit case - no update needed since we always rate limit
	if limit == 0 then
		return
	end

	-- calculate emission interval (tau) - time between each token
	-- This matches throttled library: quota.MaxRate.period
	local emission_interval = period_ns / limit

	-- retrieve theoretical arrival time
	local tat = redis.call("GET", key)
	if not tat then
		tat = now_ns
	else
		tat = tonumber(tat)
	end

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
		-- Calculate TTL like throttled library: ttl = newTat.Sub(now)
		local ttl_ns = new_tat - now_ns
		local ttl_seconds = math.ceil(ttl_ns / 1000000000) -- Convert nanoseconds to seconds
		redis.call("SET", key, new_tat, "EX", ttl_seconds)
	end
end

-- Check if capacity > 0
local res = gcraCapacity(key, now_ns, period_ns, limit, burst)
if res[2] == 0 then
	-- Not rate limited, perform the update
	gcraUpdate(key, now_ns, period_ns, limit, 1)
	return { 1, 0 }
else
	-- Rate limited, return retry time
	return { 0, res[2] }
end
