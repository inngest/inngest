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
		return false
	end

	local expiry = string.format("%d", period_ms / 1000)
	redis.call("SET", key, new_tat, "EX", expiry)

	return true
end

local function gcraUpdate(key, now_ms, period_ms, limit, burst, capacity)
  -- calculate emission interval (tau) - time between each token
  local emission = period_ms / math.max(limit, 1)

  -- delay variation tolerance: total time budget (burst + 1 tokens worth)
  local dvt = emission * (burst + 1)

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

  -- check if request should be blocked
  local allow_at = new_tat - dvt
  if now_ms < allow_at then
    -- request would exceed rate limit, don't update
    return false
  end

  if capacity > 0 then
    -- TTL is time from now until the new TAT
    local ttl_ms = new_tat - now_ms
    local ttl_sec = math.ceil(ttl_ms / 1000)
    if ttl_sec < 1 then
      ttl_sec = 1
    end
    redis.call("SET", key, string.format("%.0f", new_tat), "EX", ttl_sec)
  end

  return true
end

local function gcraCapacity(key, now_ms, period_ms, limit, burst)
  -- calculate emission interval (tau) - time between each token
  local emission = period_ms / math.max(limit, 1)

  -- delay variation tolerance: total time budget (burst + 1 tokens worth)
  local dvt = emission * (burst + 1)

  -- total limit: maximum instantaneous capacity
  local total_limit = burst + 1

  -- retrieve theoretical arrival time
  local tat = redis.call("GET", key)
  if not tat then
    tat = now_ms
  else
    tat = tonumber(tat)
  end

  -- Calculate TTL (time until TAT)
  local ttl
  if now_ms > tat then
    ttl = 0
  else
    ttl = tat - now_ms
  end

  -- Remaining capacity calculation (matches Go: (dvt - ttl) / emission)
  local next = dvt - ttl
  local remaining = 0
  if next > -emission then
    remaining = math.floor(next / emission)
  end

  -- Cap at total limit
  remaining = math.min(remaining, total_limit)

  -- Calculate retry_after: time until a request of quantity=1 would be allowed
  -- Simulates what would happen if we tried to make a request now
  local simulated_new_tat
  if now_ms > tat then
    simulated_new_tat = now_ms + emission
  else
    simulated_new_tat = tat + emission
  end
  local allow_at = simulated_new_tat - dvt
  local retry_after = -1
  if now_ms < allow_at then
    retry_after = math.ceil(allow_at - now_ms)
  end

  -- Reset after: time until bucket is fully replenished
  local reset_after = ttl

  return { remaining, retry_after, reset_after, total_limit }
end
