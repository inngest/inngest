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

---@type string[]
local leaseIdempotencyKeys = cjson.decode(ARGV[4])
---@type string[]
local leaseRunIDs = cjson.decode(ARGV[5])

---@param key string
local function getConcurrencyCount(key)
	local count = redis.call("ZCOUNT", key, tostring(nowMS), "+inf")
	return count
end

---@param key string
---@param period integer
---@param limit integer
---@param burst integer
local function gcraCapacity(key, period, limit, burst)
	-- TODO: Implement GCRA capacity (reuse existing)
	return 0
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
	if skipGCRA then
		-- noop
		constraintCapacity = availableCapacity
	elseif value.k == 1 then
		-- rate limit
		constraintCapacity = gcraCapacity(value.r.h, value.r.p, value.r.l, 0)
	elseif value.k == 2 then
		-- concurrency
		local inProgressItems = getConcurrencyCount(value.c.iik)
		local inProgressLeases = getConcurrencyCount(value.c.ilk)
		local inProgressTotal = inProgressItems + inProgressLeases
		constraintCapacity = value.c.l - inProgressTotal
	elseif value.k == 3 then
		-- throttle
		constraintCapacity = gcraCapacity(value.t.h, value.t.p, value.t.l, value.t.b)
	end

	-- If index ends up limiting capacity, reduce available capacity and remember current constraint
	if constraintCapacity < availableCapacity then
		availableCapacity = constraintCapacity
		limitingConstraint = index
	end
end

-- TODO: Handle fairness between other lease sources! Don't allow consuming entire capacity unfairly

-- TODO: If missing capacity, exit early (return limiting constraint and details)
if availableCapacity == 0 then
	return { 2, limitingConstraint }
end

local granted = availableCapacity

-- TODO: Generate leases

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

-- TODO: Bulk-Add leases to active leases for account (and account pointer to set to scavenger shard)

-- TODO: Set operation idempotency key

return { 0 }
