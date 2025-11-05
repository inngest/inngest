--
-- Acquire
-- - checks constraint capacity given constraints and the current configuration
-- -
--

local keyRequestState = KEYS[1]
local keyOperationIdempotency = KEYS[2]
local keyScavengerShard = KEYS[3]
local keyAccountLeases = KEYS[4]

local accountScopedKeyPrefix = ARGV[1]

local requestDetails = cjson.decode(ARGV[2])

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

-- TODO: If missing capacity, exit early (return limiting constraint and details)

-- TODO: Update constraint state with granted capacity

-- TODO: Store request details, granted capacity

-- TODO: Bulk-Set lease idempotency keys: foreach lease: lease idempotency key -> leaseID without idempotency period (that is set on Release)

-- TODO: Bulk-Add leases to active leases for account (and account pointer to set to scavenger shard)

-- TODO: Set operation idempotency key

return { 0 }
