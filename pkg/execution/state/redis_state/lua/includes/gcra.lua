-- performs gcra rate limiting for a given key.
--
-- Returns true on success, false if the key has been rate limited.
local function gcra(key, now_ms, period_ms, limit, burst, enableThrottleFix)
	-- Calculate the basic variables for GCRA.
	local cost = 1                            -- everything counts as a single rqeuest

	local emission  = period_ms / math.max(limit, 1)   -- how frequently we can admit new requests
	local increment = emission * cost         -- this request's time delta
  -- BUG: The variance is calculated incorrectly. We should use emission instead of period_ms.
	local variance  = period_ms * (math.max(burst, 1)) -- variance takes into account bursts
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
	local allow_at_ms = new_tat - variance            -- handle bursts.
	local diff_ms = now_ms - allow_at_ms

	if diff_ms < 0 then
		return {false, false}
	end

	local expiry = string.format("%d", period_ms / 1000)
	redis.call("SET", key, new_tat, "EX", expiry)

  local new_tat_no_burst = tat + emission
  local allow_at_no_burst = new_tat_no_burst - variance
  local used_burst = (now_ms - allow_at_no_burst) < 0

	return {true, used_burst}
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
  local new_tat
  if now_ms > tat then
    new_tat = now_ms + (math.max(capacity, 1) * emission)
  else
    new_tat = tat + (math.max(capacity, 1) * emission)
  end

  if capacity > 0 then
    local expiry = string.format("%d", period_ms / 1000)
    if expiry == "0" then
      expiry = "1"
    end
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

  -- Convert the remaining time budget back into a number of tokens.
  local capacity = math.floor(time_capacity_remain / emission)

  -- The capacity cannot exceed the defined limit + burst.
  local final_capacity = math.min(capacity, limit + burst)

  -- Calculate when the next unit will be available after consuming all remaining capacity.
  -- The current TAT represents when the bucket becomes "full" if no requests are made.
  -- If we consume final_capacity tokens now, we need to advance TAT by final_capacity * emission.
  -- The next token after consumption will be available when: new_tat + emission - total_capacity_time
  local new_tat_after_consumption = math.max(tat, now_ms) + final_capacity * emission
  local next_available_at_ms = math.ceil(new_tat_after_consumption - total_capacity_time + emission)

  return { final_capacity, next_available_at_ms }
end
