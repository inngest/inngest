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

local keyAccountLeases = KEYS[1]
local keyOperationIdempotency = KEYS[2]

---@type { e: string, f: string, s: {}[], cv: integer? }
local requestDetails = cjson.decode(ARGV[1])
if not requestDetails then
	return redis.error_reply("ERR requestDetails is nil after JSON decode")
end
local keyPrefix = ARGV[2]
local accountID = ARGV[3]
local nowMS = tonumber(ARGV[4]) --[[@as integer]]
local nowNS = tonumber(ARGV[5]) --[[@as integer]]
local operationIdempotencyTTL = tonumber(ARGV[6])--[[@as integer]]
local enableDebugLogs = tonumber(ARGV[7]) == 1

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
local configVersion = requestDetails.cv

---@type { k: integer, c: { m: integer?, s: integer?, h: string?, eh: string?, l: integer?, ilk: string?, iik: string? }?, t: { s: integer?, h: string?, k: string, eh: string?, l: integer?, b: integer?, p: integer? }?, r: { s: integer?, h: string, eh: string, l: integer, p: integer, k: string, b: integer }? }[]
local constraints = requestDetails.s
if not constraints then
	return redis.error_reply("ERR constraints array is nil")
end

-- Compute constraint capacity
---@type integer?
local availableCapacity = nil

---@type integer[]
local limitingConstraints = {}
local retryAt = 0

local constraintUsage = {}
for index, value in ipairs(constraints) do
	-- Exit checks early if no more capacity is available (e.g. no need to check fn
	-- concurrency if account concurrency is used up)
	if availableCapacity ~= nil and availableCapacity <= 0 then
		break
	end

	debug("checking constraint " .. index)

	-- Retrieve constraint capacity
	local constraintCapacity = 0
	local constraintRetryAfter = 0
	if value.k == 1 then
		-- rate limit
		local rlRes = rateLimit(value.r.k, nowNS, value.r.p, value.r.l, value.r.b, 0)
		constraintCapacity = rlRes["remaining"]
		constraintRetryAfter = toInteger(rlRes["retry_at"] / 1000000) -- convert from ns to ms

		local usage = {}
		usage["l"] = value.r.l
		usage["u"] = rlRes["u"]
		table.insert(constraintUsage, usage)
	elseif value.k == 2 then
		-- concurrency
		debug("evaluating concurrency")
		local inProgressItems = getConcurrencyCount(value.c.iik)
		local inProgressLeases = getConcurrencyCount(value.c.ilk)
		local inProgressTotal = inProgressItems + inProgressLeases
		constraintCapacity = value.c.l - inProgressTotal

		local usage = {}
		usage["l"] = value.c.l
		usage["u"] = math.max(math.min(value.c.l - constraintCapacity, value.c.l or 0), 0)
		debug(
			"i",
			index,
			"ipi",
			inProgressItems,
			"ipl",
			inProgressLeases,
			"ipt",
			inProgressTotal,
			"cc",
			constraintCapacity
		)
		table.insert(constraintUsage, usage)
	elseif value.k == 3 then
		-- throttle
		debug("evaluating throttle")
		-- allow consuming all capacity in one request (for generating multiple leases)
		local maxBurst = (value.t.l or 0) + (value.t.b or 0) - 1
		local throttleRes = throttle(value.t.k, nowMS, value.t.p, value.t.l, maxBurst, 0)
		constraintCapacity = throttleRes["remaining"]
		constraintRetryAfter = toInteger(throttleRes["retry_at"]) -- already in ms

		local usage = {}
		usage["l"] = value.t.l
		usage["u"] = math.max(math.min(value.t.l - constraintCapacity, value.t.l or 0), 0)
		table.insert(constraintUsage, usage)
	end

	-- If index ends up limiting capacity, reduce available capacity and remember current constraint
	if availableCapacity == nil or constraintCapacity < availableCapacity then
		debug(
			"constraint has less capacity",
			"c",
			index,
			"cc",
			tostring(constraintCapacity),
			"ac",
			tostring(availableCapacity)
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

---@type { s: integer, d: string[], lc: integer[], ra: integer, fr: integer, a: integer, constraintUsage: {}[] }
local res = {}
res["s"] = 1
res["d"] = debugLogs
res["lc"] = limitingConstraints
res["ra"] = retryAt
res["fr"] = fairnessReduction
res["a"] = availableCapacity
res["cu"] = constraintUsage

local encoded = cjson.encode(res)

-- Set operation idempotency TTL
if operationIdempotencyTTL > 0 then
	call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
end

return encoded
