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

	local expiry = string.format("%d", period_ms / 1000)
	redis.call("SET", key, new_tat, "EX", expiry)

	return true
end

local function gcraUpdate(key, now_ms, period_ms, limit, burst, capacity)
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
    local expiry = (period_ms / 1000)
    redis.call("SET", key, new_tat, "EX", expiry)
  end
end

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

  -- convert time capacity to token capacity
  local capacity = math.floor(time_capacity_remain / emission)

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
