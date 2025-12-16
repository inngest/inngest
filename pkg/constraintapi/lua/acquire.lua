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

---@param key string
---@param now_ns integer
---@param period_ns integer
---@param limit integer
---@param burst integer
---@param quantity integer
local function rateLimit(key, now_ns, period_ns, limit, burst, quantity)
	---@type { limit: integer, ei: number, retry_at: number, dvt: number, tat: number, inc: number, ntat: number, aat: number, diff: number, retry_after: integer?, u: number, next: number?, remaining: integer?, reset_after: integer?, limited: boolean? }
	local result = {}

	-- limit defines the maximum number of requests that can be admitted at once (irrespective of current usage)
	result["limit"] = burst + 1

	-- emission defines the window size
	local emission = period_ns / math.max(limit, 1)
	result["ei"] = emission

	-- retry_at is always computed under the assumption that all
	-- remaining capacity is consumed
	result["retry_at"] = now_ns + emission

	-- dvt determines how many requests can be admitted
	local dvt = emission * (burst + 1)
	result["dvt"] = dvt

	-- use existing tat or start at now_ms
	local tat = call("GET", key)
	if not tat then
		tat = now_ns
	else
		tat = tonumber(tat)
	end

	result["tat"] = tat
	-- When called with quantity 0, we simulate a call with quantity=1 to
	-- calculate retry after, remaining, etc. values
	local origQuantity = quantity
	if quantity == 0 then
		quantity = 1
	end

	-- increment based on quantity
	local increment = quantity * emission
	result["inc"] = increment

	-- if existing tat is in the past, increment from now_ms
	local new_tat = tat + increment
	if now_ns > tat then
		new_tat = now_ns + increment
	end
	result["ntat"] = new_tat

	-- ttl represents the current time until the full "limit" is allowed again
	local ttl = tat - now_ns
	result["reset_after"] = ttl

	-- currently used tokens must be calculated without burst
	local used_tokens = math.min(math.ceil(ttl / emission), limit)
	result["u"] = used_tokens

	-- requests should be allowed from the new_tat on, burst
	-- decreases the time to allowing a new request even if the original period received the maximum number of requests
	local allow_at = new_tat - dvt
	result["aat"] = allow_at

	-- allow_at must be in the past to allow the request (diff >= 0)
	local diff = now_ns - allow_at
	result["diff"] = diff

	if diff < 0 then
		if increment <= dvt then
			-- retry_after outlines when the next request would be accepted
			result["retry_after"] = -diff
			result["retry_at"] = now_ns - diff
		end

		if origQuantity > 0 then
			-- if we did want to update, we got limited
			local next = dvt - ttl
			result["next"] = next
			result["remaining"] = 0
			result["limited"] = true

			return result
		end
	end

	if origQuantity > 0 then
		-- update state to new_tat
		ttl = new_tat - now_ns
		local expiry = string.format("%d", math.max(ttl / 1000000000, 1))
		call("SET", key, new_tat, "EX", expiry)
	end
	result["reset_after"] = ttl

	local next = dvt - ttl
	result["next"] = next

	if next > -emission then
		local remaining = math.floor(next / emission)
		result["remaining"] = remaining
	end

	return result
end

---@param key string
---@param now_ms integer
---@param period_ms integer
---@param limit integer
---@param burst integer
---@param quantity integer
local function throttle(key, now_ms, period_ms, limit, burst, quantity)
	---@type { limit: integer, ei: number, retry_at: number, dvt: number, tat: number, inc: number, ntat: number, aat: number, diff: number, retry_after: integer?, u: number, next: number?, remaining: integer?, reset_after: integer?, limited: boolean? }
	local result = {}

	-- limit defines the maximum number of requests that can be admitted at once (irrespective of current usage)
	result["limit"] = burst + 1

	-- emission defines the window size
	local emission = period_ms / math.max(limit, 1)
	result["ei"] = emission

	-- retry_at is always computed under the assumption that all
	-- remaining capacity is consumed
	result["retry_at"] = now_ms + emission

	-- dvt determines how many requests can be admitted
	local dvt = emission * (burst + 1)
	result["dvt"] = dvt

	-- use existing tat or start at now_ms
	local tat = call("GET", key)
	if not tat then
		tat = now_ms
	else
		tat = tonumber(tat)
	end

	result["tat"] = tat
	-- When called with quantity 0, we simulate a call with quantity=1 to
	-- calculate retry after, remaining, etc. values
	local origQuantity = quantity
	if quantity == 0 then
		quantity = 1
	end

	-- increment based on quantity
	local increment = quantity * emission
	result["inc"] = increment

	-- if existing tat is in the past, increment from now_ms
	local new_tat = tat + increment
	if now_ms > tat then
		new_tat = now_ms + increment
	end
	result["ntat"] = new_tat

	-- ttl represents the current time until the full "limit" is allowed again
	local ttl = tat - now_ms
	result["reset_after"] = ttl

	-- currently used tokens must be calculated without burst
	local used_tokens = math.min(math.ceil(ttl / emission), limit)
	result["u"] = used_tokens

	-- requests should be allowed from the new_tat on, burst
	-- decreases the time to allowing a new request even if the original period received the maximum number of requests
	local allow_at = new_tat - dvt
	result["aat"] = allow_at

	-- allow_at must be in the past to allow the request (diff >= 0)
	local diff = now_ms - allow_at
	result["diff"] = diff

	if diff < 0 then
		if increment <= dvt then
			-- retry_after outlines when the next request would be accepted
			result["retry_after"] = -diff
			result["retry_at"] = now_ms - diff
		end

		if origQuantity > 0 then
			-- if we did want to update, we got limited
			local next = dvt - ttl
			result["next"] = next
			result["remaining"] = 0
			result["limited"] = true

			return result
		end
	end

	if origQuantity > 0 then
		-- update state to new_tat
		ttl = new_tat - now_ms
		local expiry = string.format("%d", math.max(ttl / 1000, 1))
		call("SET", key, new_tat, "EX", expiry)
	end
	result["reset_after"] = ttl

	local next = dvt - ttl
	result["next"] = next

	if next > -emission then
		local remaining = math.floor(next / emission)
		result["remaining"] = remaining
	end

	return result
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

-- If the same request state is still in progress (active leases), we cannot acquire more leases for the same request
-- This should never happen, as we generate a new ID for each request
local existingRequestState = call("GET", keyRequestState)
if existingRequestState ~= nil and existingRequestState ~= false and existingRequestState ~= "" then
	local res = {}
	res["s"] = 4
	res["d"] = debugLogs
	res["aal"] = getActiveAccountLeasesCount()
	res["eal"] = getExpiredAccountLeasesCount()
	res["ele"] = getEarliestLeaseExpiry()

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
		local rlRes = rateLimit(value.r.k, nowNS, value.r.p, value.r.l, value.r.b, 0)
		constraintCapacity = rlRes["remaining"]
		constraintRetryAfter = toInteger(rlRes["retry_at"] / 1000000) -- convert from ns to ms
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
		-- allow consuming all capacity in one request (for generating multiple leases)
		local maxBurst = (value.t.l or 0) + (value.t.b or 0) - 1
		local throttleRes = throttle(value.t.k, nowMS, value.t.p, value.t.l, maxBurst, 0)
		constraintCapacity = throttleRes["remaining"]
		constraintRetryAfter = toInteger(throttleRes["retry_at"]) -- already in ms
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

	local keyLeaseDetails = string.format("%s:ld:%s", scopedKeyPrefix, initialLeaseID)

	-- Store lease details (hashed lease idempotency key, associated run ID, operation idempotency key for request details)
	call("HSET", keyLeaseDetails, "lik", hashedLeaseIdempotencyKey, "rid", leaseRunID, "req", requestID)

	-- Add lease to scavenger set of account leases
	call("ZADD", keyAccountLeases, tostring(leaseExpiryMS), initialLeaseID)

	-- Add constraint check idempotency for each lease (for graceful handling in rate limit, Lease, BacklogRefill, as well as Acquire in case lease expired)
	local keyLeaseConstraintCheckIdempotency = string.format("%s:ik:cc:%s", scopedKeyPrefix, hashedLeaseIdempotencyKey)
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
