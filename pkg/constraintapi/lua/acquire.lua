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

local envID = requestDetails.e
local functionID = requestDetails.f

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
local granted = requested

-- TODO: Can we generate a list of updates to apply in batch?
local updates = {}

for index, value in ipairs(constraints) do
	if value.k == 1 then
	-- rate limit
	elseif value.k == 2 then
	-- concurrency
	elseif value.k == 3 then
		-- throttle
	end
end

-- TODO: If missing capacity, exit early (return limiting constraint and details)
if granted == 0 then
	return { 2 }
end

-- TODO: Update constraint state with granted capacity

-- Populate request details
requestDetails.g = granted
requestDetails.a = granted

-- Store request details
-- TODO: Should this have a TTL just in case? e.g. 24h?
-- If we did this, we could not properly clean up some state, so maybe we should just trust the scavenger
redis.call("SET", keyRequestState, cjson.encode(requestDetails))

-- TODO: Bulk-Set lease idempotency keys: foreach lease: lease idempotency key -> leaseID without idempotency period (that is set on Release)

-- TODO: Bulk-Add leases to active leases for account (and account pointer to set to scavenger shard)

-- TODO: Set operation idempotency key

return { 0 }
