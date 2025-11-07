---@type string
local key = ARGV[1]

---@type integer
local now_ms = tonumber(ARGV[2])

---@type integer
local period_ms = tonumber(ARGV[3])

---@type integer
local limit = tonumber(ARGV[4])

---@type integer
local burst = tonumber(ARGV[5])

---@param key string
---@param now_ms integer
---@param period_ms integer
---@param limit integer
---@param capacity integer
local function gcraUpdate(key, now_ms, period_ms, limit, capacity)
	-- calculate emission interval (tau) - time between each token
	local emission = period_ms / math.max(limit, 1)

	-- retrieve theoretical arrival time
	local tat = redis.call("GET", key)
	if not tat then
		tat = now_ms
	else
		tat = tonumber(tat)
	end

	-- calculate next theoretical arrival time
	local new_tat = tat + (math.max(capacity, 1) * emission)

	if capacity > 0 then
		local expiry = string.format("%d", period_ms / 1000)
		redis.call("SET", key, new_tat, "EX", expiry)
	end
end

---@param key string
---@param now_ms integer
---@param period_ms integer
---@param limit integer
---@param burst integer
---@return integer[]
local function gcraCapacity(key, now_ms, period_ms, limit, burst)
	-- calculate emission interval (tau) - time between each token
	local emission = period_ms / math.max(limit, 1)

	-- calculate total capacity in time units
	local total_capacity_time = emission * (limit + burst)

	-- retrieve theoretical arrival time
	local tat = redis.call("GET", key)
	if not tat then
		tat = now_ms
	else
		tat = tonumber(tat)
	end

	-- remaining capacity in time units
	local time_capacity_remain = now_ms + total_capacity_time - tat

	-- Convert the remaining time budget back into a number of tokens.
	local capacity = math.floor(time_capacity_remain / emission)

	-- The capacity cannot exceed the defined limit + burst.
	local final_capacity = math.min(capacity, limit + burst)

	if final_capacity < 1 then
		-- We are throttled. Calculate the time when the capacity will be >= 1.
		-- This is the point where enough time has passed to "earn" one token.
		-- The formula is derived from solving for the future time `t` where capacity becomes 1.
		local next_available_at_ms = tat - total_capacity_time + emission
		return { final_capacity, math.ceil(next_available_at_ms) }
	else
		-- Not throttled, so there is no "next available time" to report.
		return { final_capacity, 0 }
	end
end

local res = gcraCapacity(key, now_ms, period_ms, limit, burst)
if res[0] == 0 then
	return { 0, res[1] }
end

gcraUpdate(key, now_ms, period_ms, limit, 1)

return { 1, 0 }
