---@type string
local key = ARGV[1]

---@type integer
local now_ns = tonumber(ARGV[2])

---@type integer
local period_ns = tonumber(ARGV[3])

---@type integer
local limit = tonumber(ARGV[4])

---@type integer
local burst = tonumber(ARGV[5])

---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param capacity integer
local function gcraUpdate(key, now_ns, period_ns, limit, capacity)
	-- calculate emission interval (tau) - time between each token
	-- This matches throttled library: quota.MaxRate.period
	local emission_interval = period_ns / math.max(limit, 1)

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

---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param burst integer
---@return integer[]
local function gcraCapacity(key, now_ns, period_ns, limit, burst)
	-- Match throttled library calculations exactly
	-- emissionInterval = quota.MaxRate.period
	local emission_interval = period_ns / math.max(limit, 1)
	
	-- delayVariationTolerance = quota.MaxRate.period * (quota.MaxBurst + 1)
	-- In throttled library: limit = quota.MaxBurst + 1, so burst = limit - 1
	-- But we receive burst as the actual burst value, so we use burst + 1 to match
	local delay_variation_tolerance = emission_interval * (burst + 1)

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
		-- We are rate limited
		-- Calculate retry after time: -diff (when allowAt becomes <= now)
		local retry_after_ns = -diff
		return { 0, now_ns + retry_after_ns }
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

local res = gcraCapacity(key, now_ns, period_ns, limit, burst)
if res[1] == 0 then
	-- Not rate limited, perform the update
	gcraUpdate(key, now_ns, period_ns, limit, 1)
	return { 1, 0 }
else
	-- Rate limited, return retry time
	return { 0, res[2] }
end
