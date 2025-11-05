--
-- Acquire
-- - checks constraint capacity given constraints and the current configuration
-- -
--

---@type string
local keyRequestState = KEYS[1]
---@type string
local keyOperationIdempotency = KEYS[2]
---@type string
local keyScavengerShard = KEYS[3]
---@type string
local keyAccountLeases = KEYS[4]

---@type string
local accountScopedKeyPrefix = ARGV[1]

---@type { k: string, e: string, f: string, s: {}[], cv: integer?, r: integer?, g: integer?, a: integer?, l: integer? }
local requestDetails = cjson.decode(ARGV[2])

---@type integer
local requested = requestDetails.r

---@type integer
local configVersion = requestDetails.cv

---@type { k: integer, c: { m: integer?, s: integer?, h: string?, eh: string?, l: integer?, ilk: string?, iik: string? }?, t: { s: integer?, h: string?, eh: string?, l: integer?, b: integer?, p: integer? }?, r: { s: integer?, h: string?, eh: string?, l: integer?, p: string? }? }[]
local constraints = requestDetails.s

-- TODO: Handle operation idempotency (was this request seen before?)
local opIdempotency = redis.call("GET", keyOperationIdempotency)
if opIdempotency ~= nil and opIdempotency ~= false then
	-- Return idempotency state to user (same as initial response)
	return { 1, opIdempotency }
end

-- TODO: Is the operation related to a single idempotency key that is still valid? Return that

-- TODO: Verify no far newer config was seen (reduce driftt)

-- TODO: Handle constraint idempotency (do we need to skip GCRA? only for single leases with valid idempotency)

-- TODO: Compute constraint capacity
local availableCapacity = requested
local limitingConstraint = -1

-- TODO: Can we generate a list of updates to apply in batch?
-- local updates = {}

-- TODO: Extract constraint capacity calculation into testable function
for index, value in ipairs(constraints) do
	-- Exit checks early if no more capacity is available (e.g. no need to check fn
	-- concurrency if account concurrency is used up)
	if availableCapacity <= 0 then
		break
	end

	-- Retrieve constraint capacity
	local constraintCapacity = 0
	if value.k == 1 then
	-- rate limit
	-- TODO: Check GCRA capacity against value.r.eh
	elseif value.k == 2 then
	-- concurrency
	-- TODO: Check value.c.iik
	-- TODO: Check value.c.ilk
	elseif value.k == 3 then
		-- throttle
		-- TODO: Check GCRA capacity against value.t.eh
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

-- TODO: Update constraint state with granted capacity
-- For throttle and rate limit, update the same GCRA key
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
