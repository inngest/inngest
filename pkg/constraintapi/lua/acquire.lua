--
-- Acquire
-- - checks constraint capacity given constraints and the current configuration
-- -
--

---@module 'cjson'
local cjson = cjson

---@param command string
---@param ... string
local function call(command, ...)
	return redis.call(command, ...)
end

---@type string[]
local KEYS = KEYS

---@type string[]
local ARGV = ARGV

local keyRequestState = KEYS[1]
local keyOperationIdempotency = KEYS[2]
local keyConstraintCheckIdempotency = KEYS[3]
local keyScavengerShard = KEYS[4]
local keyAccountLeases = KEYS[5]

---@type { k: string, e: string, f: string, s: {}[], cv: integer?, r: integer?, g: integer?, a: integer?, l: integer?, lik: string[]?, lri: table<string, string>? }
local requestDetails = cjson.decode(ARGV[1])
local accountID = ARGV[2]
local nowMS = tonumber(ARGV[3]) --[[@as integer]]
local nowNS = tonumber(ARGV[4]) --[[@as integer]]
local leaseExpiryMS = tonumber(ARGV[5])
local keyPrefix = ARGV[6]
---@type string[]
local initialLeaseIDs = cjson.decode(ARGV[7])
if not initialLeaseIDs then
	return redis.error_reply("ERR initialLeaseIDs is nil after JSON decode")
end
local hashedOperationIdempotencyKey = ARGV[8]
local operationIdempotencyTTL = tonumber(ARGV[9])--[[@as integer]]
local constraintCheckIdempotencyTTL = tonumber(ARGV[10])--[[@as integer]]
local enableDebugLogs = tonumber(ARGV[11]) == 1

---@type string[]
local debugLogs = {}
---@param ... string
local function debug(...)
	if enableDebugLogs then
		local args = { ... }
		table.insert(debugLogs, table.concat(args, " "))
	end
end

---@param key string
local function getConcurrencyCount(key)
	local count = call("ZCOUNT", key, tostring(nowMS), "+inf")
	if count == nil then
		return 0
	end
	return count
end

---@return integer
local function getActiveAccountLeasesCount()
	local count = call("ZCOUNT", keyAccountLeases, tostring(nowMS), "+inf")
	if count == nil then
		return 0
	end
	return count
end

---@return integer
local function getExpiredAccountLeasesCount()
	local count = call("ZCOUNT", keyAccountLeases, "-inf", tostring(nowMS))
	if count == nil then
		return 0
	end
	return count
end

---@return integer
local function getEarliestLeaseExpiry()
	local count = call("ZRANGE", keyAccountLeases, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
	if count == nil or count == false or #count == 0 then
		return 0
	end
	return tonumber(count[2])
end

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
	local tat = call("GET", key)
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
		call("SET", key, clamped_tat, "KEEPTTL")
	end

	return clamped_tat
end

--- rateLimitCapacity is the first half of a nanosecond-precision GCRA implementation. This method calculates the number of requests that can be admitted in the current period.
---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param burst integer
---@return integer[] returns a 3-tuple of remaining capacity, retry at, and current usage
local function rateLimitCapacity(key, now_ns, period_ns, limit, burst)
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

--- rateLimitUpdate is the second half of a nanosecond-precision GCRA implementation, used for updating rate limit state.
---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param capacity integer the number of requests to admit at once
---@param burst integer
local function rateLimitUpdate(key, now_ns, period_ns, limit, capacity, burst)
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
		call("SET", key, clamped_tat, "EX", ttl_seconds)
	end
end

---@param key string
---@param now_ms integer
---@param period_ms integer
---@param limit integer
---@param burst integer
---@return integer[]
local function throttleCapacity(key, now_ms, period_ms, limit, burst)
	-- TODO: Reuse shared script

	-- calculate emission interval (tau) - time between each token
	local emission = period_ms / math.max(limit, 1)

	-- calculate total capacity in time units
	local total_capacity_time = emission * (limit + burst)

	-- retrieve theoretical arrival time
	local tat = call("GET", key)
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

---@param key string
---@param now_ms integer
---@param period_ms integer
---@param limit integer
---@param capacity integer
local function throttleUpdate(key, now_ms, period_ms, limit, capacity)
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
		call("SET", key, new_tat, "EX", expiry)
	end
end

---@type integer
local requested = requestDetails.r

---@type integer
local configVersion = requestDetails.cv

---@type { k: integer, c: { m: integer?, s: integer?, h: string?, eh: string?, l: integer?, ilk: string?, iik: string? }?, t: { s: integer?, h: string?, k: string, eh: string?, l: integer, b: integer, p: integer }?, r: { s: integer?, h: string, eh: string, l: integer, p: integer, k: string, b: integer }? }[]
local constraints = requestDetails.s
if not constraints then
	return redis.error_reply("ERR constraints array is nil")
end

-- Handle operation idempotency
local opIdempotency = call("GET", keyOperationIdempotency)
if opIdempotency ~= nil and opIdempotency ~= false then
	debug("hit operation idempotency")

	-- Return idempotency state to user (same as initial response)
	return opIdempotency
end

-- TODO: Is the operation related to a single idempotency key that is still valid? Return that
-- TODO: This is basically the key queues case: What if the existing lease is still valid? And if it expired, can the
-- lease idempotency key be safely reused (should be fine)

-- TODO: Verify no far newer config was seen (reduce driftt)

-- Compute constraint capacity
local availableCapacity = requested

---@type integer[]
local limitingConstraints = {}
local retryAt = 0

-- Skip GCRA if constraint check idempotency key is present
local skipGCRA = call("EXISTS", keyConstraintCheckIdempotency) == 1

for index, value in ipairs(constraints) do
	-- Exit checks early if no more capacity is available (e.g. no need to check fn
	-- concurrency if account concurrency is used up)
	if availableCapacity <= 0 then
		break
	end

	debug("checking constraint " .. index)

	-- Retrieve constraint capacity
	local constraintCapacity = 0
	local constraintRetryAfter = 0
	if skipGCRA and (value.k == 1 or value.k == 3) then
		-- noop
		constraintCapacity = availableCapacity
		debug("skipping gcra" .. index)
	elseif value.k == 1 then
		-- rate limit
		local burst = math.floor(value.r.l / 10) -- align with burst in ratelimit
		local rlRes = rateLimitCapacity(value.r.k, nowNS, value.r.p, value.r.l, burst)
		constraintCapacity = rlRes[1]
		constraintRetryAfter = toInteger(rlRes[2] / 1000000) -- convert from ns to ms
	elseif value.k == 2 then
		-- concurrency
		debug("evaluating concurrency")
		local inProgressItems = getConcurrencyCount(value.c.iik)
		local inProgressLeases = getConcurrencyCount(value.c.ilk)
		local inProgressTotal = inProgressItems + inProgressLeases
		constraintCapacity = value.c.l - inProgressTotal
	elseif value.k == 3 then
		-- throttle
		debug("evaluating throttle")
		local throttleRes = throttleCapacity(value.t.k, nowMS, value.t.p, value.t.l, value.t.b)
		constraintCapacity = throttleRes[1]
		constraintRetryAfter = toInteger(throttleRes[2]) -- already in ms
	end

	-- If index ends up limiting capacity, reduce available capacity and remember current constraint
	if constraintCapacity < availableCapacity then
		debug(
			"constraint has less capacity",
			"c",
			index,
			"cc",
			tostring(constraintCapacity),
			"ac",
			tostring(availableCapacity),
			"ra",
			tostring(constraintRetryAfter)
		)

		availableCapacity = constraintCapacity
		table.insert(limitingConstraints, index)

		-- if the constraint must be retried later than the initial/last constraint, update retryAfter
		if constraintRetryAfter > retryAt then
			retryAt = constraintRetryAfter
		end
	end
end

-- TODO: Handle fairness between other lease sources! Don't allow consuming entire capacity unfairly
local fairnessReduction = 0
-- TODO: How can we track and gracefully handle end to end that we ran out of capacity because for fairness?
availableCapacity = availableCapacity - fairnessReduction

-- TODO: If missing capacity, exit early (return limiting constraint and details)
if availableCapacity <= 0 then
	local res = {}
	res["s"] = 2
	res["lc"] = limitingConstraints
	res["ra"] = retryAt
	res["d"] = debugLogs
	res["fr"] = fairnessReduction
	res["aal"] = getActiveAccountLeasesCount()
	res["eal"] = getExpiredAccountLeasesCount()
	res["ele"] = getEarliestLeaseExpiry()

	return cjson.encode(res)
end

local granted = availableCapacity

---@type { lid: string, lik: string }[]
local grantedLeases = {}

-- Update constraints
for i = 1, granted, 1 do
	if not requestDetails.lik then
		return redis.error_reply("ERR requestDetails.lik is nil during update")
	end
	if not initialLeaseIDs then
		return redis.error_reply("ERR initialLeaseIDs is nil during update")
	end
	local hashedLeaseIdempotencyKey = requestDetails.lik[i]
	local leaseRunID = (requestDetails.lri ~= nil and requestDetails.lri[hashedLeaseIdempotencyKey]) or ""
	local initialLeaseID = initialLeaseIDs[i]

	for _, value in ipairs(constraints) do
		if skipGCRA then
		-- noop
		elseif value.k == 1 then
			debug("updating rate limit", value.r.h)
			-- rate limit
			rateLimitUpdate(value.r.k, nowNS, value.r.p, value.r.l, 1, value.r.b)
		elseif value.k == 2 then
			-- concurrency
			call("ZADD", value.c.ilk, tostring(leaseExpiryMS), initialLeaseID)
		elseif value.k == 3 then
			-- throttle
			throttleUpdate(value.t.k, nowMS, value.t.p, value.t.l, 1)
		end
	end

	local keyLeaseDetails = string.format("{%s}:%s:ld:%s", keyPrefix, accountID, initialLeaseID)

	-- Store lease details (hashed lease idempotency key, associated run ID, operation idempotency key for request details)
	call(
		"HSET",
		keyLeaseDetails,
		"lik",
		hashedLeaseIdempotencyKey,
		"rid",
		leaseRunID,
		"oik",
		hashedOperationIdempotencyKey
	)

	-- Add lease to scavenger set of account leases
	call("ZADD", keyAccountLeases, tostring(leaseExpiryMS), initialLeaseID)

	-- Add constraint check idempotency for each lease (for graceful handling in rate limit, Lease, BacklogRefill, as well as Acquire in case lease expired)
	local keyLeaseConstraintCheckIdempotency =
		string.format("{%s}:%s:ik:cc:%s", keyPrefix, accountID, hashedLeaseIdempotencyKey)
	call("SET", keyLeaseConstraintCheckIdempotency, tostring(nowMS), "EX", tostring(constraintCheckIdempotencyTTL))

	---@type { lid: string, lik: string }
	local leaseObject = {}
	leaseObject["lid"] = initialLeaseID
	leaseObject["lik"] = hashedLeaseIdempotencyKey

	table.insert(grantedLeases, leaseObject)
end

call("SET", keyConstraintCheckIdempotency, tostring(nowMS), "EX", tostring(constraintCheckIdempotencyTTL))

-- For step concurrency, add the lease idempotency keys to the new in progress leases sets using the lease expiry as score
-- For run concurrency, add the runID to the in progress runs set and the lease idempotency key to the dynamic run in progress leases set

-- Populate request details
requestDetails.g = availableCapacity
requestDetails.a = availableCapacity

-- Store request details
call("SET", keyRequestState, cjson.encode(requestDetails))

-- Update pointer to account leases
local accountScore = call("ZSCORE", keyScavengerShard, accountID)
if accountScore == nil or accountScore == false or tonumber(accountScore) > leaseExpiryMS then
	call("ZADD", keyScavengerShard, tonumber(leaseExpiryMS), accountID)
end

-- Construct result

---@type { s: integer, lc: integer[], fr: integer, r: integer, g: integer, l: { lid: string, lik: string }[] }
local result = {}

result["s"] = 3
result["r"] = requested
result["g"] = granted
result["l"] = grantedLeases
result["lc"] = limitingConstraints
result["ra"] = retryAt -- include retryAt to hint when next capacity is available
result["d"] = debugLogs
result["fr"] = fairnessReduction
result["aal"] = getActiveAccountLeasesCount()
result["eal"] = getExpiredAccountLeasesCount()
result["ele"] = getEarliestLeaseExpiry()

local encoded = cjson.encode(result)

-- Set operation idempotency TTL
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))

return encoded
