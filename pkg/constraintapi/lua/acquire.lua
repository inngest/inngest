--
-- Acquire
-- - checks constraint capacity given constraints and the current configuration
-- -
--

---@module 'cjson'
local cjson = cjson

---@module 'redis'
local redis = redis

---@type string[]
local KEYS = KEYS

---@type string[]
local ARGV = ARGV

---@type string
local keyRequestState = KEYS[1]
---@type string
local keyOperationIdempotency = KEYS[2]
---@type string
local keyScavengerShard = KEYS[3]
---@type string
local keyAccountLeases = KEYS[4]

---@type { k: string, e: string, f: string, s: {}[], cv: integer?, r: integer?, g: integer?, a: integer?, l: integer?, lik: string[]?, lri: table<string, string>? }
local requestDetails = cjson.decode(ARGV[1])

local nowMS = tonumber(ARGV[2])

local leaseExpiryMS = tonumber(ARGV[3])

local keyPrefix = ARGV[4]

---@type string[]
local initialLeaseIDs = cjson.decode(ARGV[5])

local operationIdempotencyKey = ARGV[6]

local accountID = ARGV[7]

---@param key string
local function getConcurrencyCount(key)
	local count = redis.call("ZCOUNT", key, tostring(nowMS), "+inf")
	return count
end

---@param key string
---@param period integer
---@param limit integer
---@param burst integer
---@return integer[]
local function gcraCapacity(key, period, limit, burst)
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

---@param key string
---@param period integer
---@param limit integer
---@param burst integer
---@param capacity integer
local function gcraUpdate(key, period, limit, burst, capacity)
	return 0
end

---@type integer
local requested = requestDetails.r

---@type integer
local configVersion = requestDetails.cv

---@type { k: integer, c: { m: integer?, s: integer?, h: string?, eh: string?, l: integer?, ilk: string?, iik: string? }?, t: { s: integer?, h: string?, eh: string?, l: integer?, b: integer?, p: integer? }?, r: { s: integer?, h: string?, eh: string?, l: integer?, p: integer? }? }[]
local constraints = requestDetails.s

-- TODO: Handle operation idempotency (was this request seen before?)
local opIdempotency = redis.call("GET", keyOperationIdempotency)
if opIdempotency ~= nil and opIdempotency ~= false then
	-- Return idempotency state to user (same as initial response)
	return { 1, opIdempotency }
end

-- TODO: Is the operation related to a single idempotency key that is still valid? Return that

-- TODO: Verify no far newer config was seen (reduce driftt)

-- TODO: Compute constraint capacity
local availableCapacity = requested
local limitingConstraint = -1
local retryAt = 0

-- TODO: Can we generate a list of updates to apply in batch?
-- local updates = {}

-- TODO: Handle constraint idempotency (do we need to skip GCRA? only for single leases with valid idempotency)
local skipGCRA = false

-- TODO: Extract constraint capacity calculation into testable function
for index, value in ipairs(constraints) do
	-- Exit checks early if no more capacity is available (e.g. no need to check fn
	-- concurrency if account concurrency is used up)
	if availableCapacity <= 0 then
		break
	end

	-- Retrieve constraint capacity
	local constraintCapacity = 0
	local constraintRetryAfter = 0
	if skipGCRA then
		-- noop
		constraintCapacity = availableCapacity
	elseif value.k == 1 then
		-- rate limit
		local gcraRes = gcraCapacity(value.r.h, value.r.p, value.r.l, 0)
		constraintCapacity = gcraRes[0]
		constraintRetryAfter = gcraRes[1]
	elseif value.k == 2 then
		-- concurrency
		local inProgressItems = getConcurrencyCount(value.c.iik)
		local inProgressLeases = getConcurrencyCount(value.c.ilk)
		local inProgressTotal = inProgressItems + inProgressLeases
		constraintCapacity = value.c.l - inProgressTotal
	elseif value.k == 3 then
		-- throttle
		local gcraRes = gcraCapacity(value.t.h, value.t.p, value.t.l, value.t.b)
		constraintCapacity = gcraRes[0]
		constraintRetryAfter = gcraRes[1]
	end

	-- If index ends up limiting capacity, reduce available capacity and remember current constraint
	if constraintCapacity < availableCapacity then
		availableCapacity = constraintCapacity
		limitingConstraint = index

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
	return { 2, limitingConstraint }
end

local granted = availableCapacity

-- Update constraints
for i = 1, granted, 1 do
	local leaseIdempotencyKey = requestDetails.lik[i]
	local leaseRunID = requestDetails.lri[leaseIdempotencyKey]
	local initialLeaseID = initialLeaseIDs[i]

	for _, value in ipairs(constraints) do
		-- Retrieve constraint capacity
		local constraintCapacity = 0
		if skipGCRA then
		-- noop
		elseif value.k == 1 then
			-- rate limit
			gcraUpdate(value.r.h, value.r.p, value.r.l, 0, 1)
		elseif value.k == 2 then
			-- concurrency
			redis.call("ZADD", value.c.ilk, tostring(leaseExpiryMS), leaseIdempotencyKey)
		elseif value.k == 3 then
			-- throttle
			gcraUpdate(value.t.h, value.t.p, value.t.l, value.t.b, 1)
		end
	end

	local keyLeaseDetails = string.format("{%s}:%s:ld:%s", keyPrefix, accountID, leaseIdempotencyKey)

	-- Store lease details (current lease ID, associated run ID, operation idempotency key for request details)
	redis.call("HSET", keyLeaseDetails, "lid", initialLeaseID, "rid", leaseRunID, "oik", operationIdempotencyKey)

	-- Add lease to scavenger set of account leases
	redis.call("ZADD", keyAccountLeases, tostring(leaseExpiryMS), leaseIdempotencyKey)
end

-- For step concurrency, add the lease idempotency keys to the new in progress leases sets using the lease expiry as score
-- For run concurrency, add the runID to the in progress runs set and the lease idempotency key to the dynamic run in progress leases set

-- Populate request details
requestDetails.g = availableCapacity
requestDetails.a = availableCapacity

-- Store request details
-- TODO: Should this have a TTL just in case? e.g. 24h?
-- If we did this, we could not properly clean up some state, so maybe we should just trust the scavenger
redis.call("SET", keyRequestState, cjson.encode(requestDetails))

-- TODO: Bulk-Set lease idempotency keys: foreach lease: lease idempotency key -> leaseID without idempotency period (that is set on Release)

local accountScore = redis.call("ZSCORE", keyScavengerShard, accountID)
if accountScore == nil or accountScore == false or tonumber(accountScore) > leaseExpiryMS then
	redis.call("ZADD", keyScavengerShard, tonumber(leaseExpiryMS), accountID)
end

-- TODO: Set operation idempotency key

return { 0 }
