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

---@type { k: string, e: string, f: string, s: {}[], cv: integer?, r: integer?, g: integer?, a: integer?, l: integer?, lik: string[]?, lri: table<string, string>?, m: { ss: integer?, sl: integer?, sm: integer? }? }
local requestDetails = cjson.decode(ARGV[1])
local requestID = ARGV[2]
local accountID = ARGV[3]
local nowMS = tonumber(ARGV[4]) --[[@as integer]]
local nowNS = tonumber(ARGV[5]) --[[@as integer]]
local leaseExpiryMS = tonumber(ARGV[6])
local scopedKeyPrefix = ARGV[7]
---@type string[]
local initialLeaseIDs = cjson.decode(ARGV[8])
if not initialLeaseIDs then
	return redis.error_reply("ERR initialLeaseIDs is nil after JSON decode")
end
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

--- toInteger ensures a value is stored as an integer to prevent Redis scientific notation serialization
---@param value number
---@return integer
local function toInteger(value)
	return math.floor(value + 0.5) -- Round to nearest integer
end

-- $include(helper/gcra.lua)

---@type integer
local requested = requestDetails.r

---@type integer
local configVersion = requestDetails.cv

---@type { k: integer, c: { m: integer?, s: integer?, h: string?, eh: string?, l: integer?, ilk: string?, iik: string?, ra: integer? }?, t: { s: integer?, h: string?, k: string, eh: string?, l: integer, b: integer, p: integer }?, r: { s: integer?, h: string, eh: string, l: integer, p: integer, k: string, b: integer }? }[]
local constraints = requestDetails.s
if not constraints then
	return redis.error_reply("ERR constraints array is nil")
end

-- Get operation idempotency and check existing request state in a single call
local results = call("MGET", keyOperationIdempotency, keyRequestState)
local opIdempotency = results[1]
local existingRequestState = results[2]

if opIdempotency ~= nil and opIdempotency ~= false then
	-- Return idempotency state to user (same as initial response)
	return opIdempotency
end

-- If the same request state is still in progress (active leases), we cannot acquire more leases for the same request
-- This should never happen, as we generate a new ID for each request
if existingRequestState ~= nil and existingRequestState ~= false and existingRequestState ~= "" then
	local res = {}
	res["s"] = 4
	res["d"] = debugLogs

	return cjson.encode(res)
end

-- TODO: Is the operation related to a single idempotency key that is still valid? Return that
-- TODO: This is basically the key queues case: What if the existing lease is still valid? And if it expired, can the
-- lease idempotency key be safely reused (should be fine)

-- TODO: Verify no far newer config was seen (reduce driftt)

-- Compute constraint capacity
local availableCapacity = requested

---@type integer[]
local limitingConstraints = {}
---@type integer[]
local exhaustedConstraints = {}
---@type table<integer, integer>
local constraintCapacities = {}
---@type table<integer, boolean>
local exhaustedSet = {}
local retryAt = 0

-- Skip GCRA if constraint check idempotency key is present
local skipGCRA = call("EXISTS", keyConstraintCheckIdempotency) == 1

for index, value in ipairs(constraints) do
	-- Exit checks early if no more capacity is available (e.g. no need to check fn
	-- concurrency if account concurrency is used up)
	if availableCapacity <= 0 then
		break
	end

	-- Retrieve constraint capacity
	local constraintCapacity = 0
	local constraintRetryAfter = 0
	if skipGCRA and (value.k == 1 or value.k == 3) then
		-- noop
		constraintCapacity = availableCapacity
	elseif value.k == 1 then
		-- rate limit
		local rlRes = rateLimit(value.r.k, nowNS, value.r.p, value.r.l, value.r.b, 0)
		constraintCapacity = rlRes["remaining"]
		constraintRetryAfter = toInteger(rlRes["retry_at"] / 1000000) -- convert from ns to ms
	elseif value.k == 2 then
		-- concurrency
		local inProgressItems = getConcurrencyCount(value.c.iik)
		local inProgressLeases = getConcurrencyCount(value.c.ilk)
		local inProgressTotal = inProgressItems + inProgressLeases
		constraintCapacity = value.c.l - inProgressTotal
		constraintRetryAfter = toInteger(nowMS + value.c.ra)
	elseif value.k == 3 then
		-- throttle
		-- allow consuming all capacity in one request (for generating multiple leases)
		local maxBurst = (value.t.l or 0) + (value.t.b or 0) - 1
		local throttleRes = throttle(value.t.k, nowMS, value.t.p, value.t.l, maxBurst, 0)
		constraintCapacity = throttleRes["remaining"]
		constraintRetryAfter = toInteger(throttleRes["retry_at"]) -- already in ms
	end

	-- Store constraint capacity for later exhaustion check
	constraintCapacities[index] = constraintCapacity

	-- Track if constraint is exhausted before granting
	if constraintCapacity <= 0 then
		if not exhaustedSet[index] then
			table.insert(exhaustedConstraints, index)
			exhaustedSet[index] = true
		end
	end

	-- If index ends up limiting capacity, reduce available capacity and remember current constraint
	if constraintCapacity < availableCapacity then
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
	res["ec"] = exhaustedConstraints
	res["ra"] = retryAt
	res["d"] = debugLogs
	res["fr"] = fairnessReduction

	return cjson.encode(res)
end

local granted = availableCapacity

---@type { lid: string, lik: string }[]
local grantedLeases = {}
-- Pre-allocate array slots for grantedLeases
for i = 1, granted do
	-- Placeholder, will be replaced in loop
	grantedLeases[i] = false
end

-- Collect arguments for batched ZADD to keyAccountLeases
local accountLeasesArgs = {}
-- Pre-allocate for 2 items per lease (score + member)
for i = 1, granted * 2 do
	accountLeasesArgs[i] = false
end

-- Pre-compute key prefixes to avoid repeated string.format calls
local keyPrefixLeaseDetails = scopedKeyPrefix .. ":ld:"
local keyPrefixConstraintCheck = scopedKeyPrefix .. ":ik:cc:"

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
			-- rate limit
			rateLimit(value.r.k, nowNS, value.r.p, value.r.l, value.r.b, 1)
		elseif value.k == 2 then
			-- concurrency
			call("ZADD", value.c.ilk, tostring(leaseExpiryMS), initialLeaseID)
		elseif value.k == 3 then
			-- update throttle: consume 1 unit
			-- allow consuming all capacity in one request (for generating multiple leases)
			local maxBurst = (value.t.l or 0) + (value.t.b or 0) - 1
			throttle(value.t.k, nowMS, value.t.p, value.t.l, maxBurst, 1)
		end
	end

	local keyLeaseDetails = keyPrefixLeaseDetails .. initialLeaseID

	-- Store lease details (hashed lease idempotency key, associated run ID, operation idempotency key for request details)
	call("HSET", keyLeaseDetails, "lik", hashedLeaseIdempotencyKey, "rid", leaseRunID, "req", requestID)

	-- Collect arguments for batched ZADD to account leases (executed after loop)
	accountLeasesArgs[(i - 1) * 2 + 1] = tostring(leaseExpiryMS)
	accountLeasesArgs[(i - 1) * 2 + 2] = initialLeaseID

	-- Add constraint check idempotency for each lease (for graceful handling in rate limit, Lease, BacklogRefill, as well as Acquire in case lease expired)
	local keyLeaseConstraintCheckIdempotency = keyPrefixConstraintCheck .. hashedLeaseIdempotencyKey
	call("SET", keyLeaseConstraintCheckIdempotency, tostring(nowMS), "EX", tostring(constraintCheckIdempotencyTTL))

	---@type { lid: string, lik: string }
	local leaseObject = {}
	leaseObject["lid"] = initialLeaseID
	leaseObject["lik"] = hashedLeaseIdempotencyKey

	grantedLeases[i] = leaseObject
end

-- Batch add all leases to account leases sorted set
if #accountLeasesArgs > 0 then
	call("ZADD", keyAccountLeases, unpack(accountLeasesArgs))
end

-- Check for constraints exhausted after granting
for index, capacity in pairs(constraintCapacities) do
	if capacity - granted <= 0 then
		if not exhaustedSet[index] then
			table.insert(exhaustedConstraints, index)
			exhaustedSet[index] = true
		end
	end
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

---@type { s: integer, lc: integer[], ec: integer[], fr: integer, r: integer, g: integer, l: { lid: string, lik: string }[] }
local result = {}

result["s"] = 3
result["r"] = requested
result["g"] = granted
result["l"] = grantedLeases
result["lc"] = limitingConstraints
result["ec"] = exhaustedConstraints
result["ra"] = retryAt -- include retryAt to hint when next capacity is available
result["d"] = debugLogs
result["fr"] = fairnessReduction

local encoded = cjson.encode(result)

-- Set operation idempotency TTL
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))

return encoded
