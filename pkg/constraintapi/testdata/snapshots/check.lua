local cjson = cjson
local function call(command, ...)
	return redis.call(command, ...)
end
local KEYS = KEYS
local ARGV = ARGV
local keyAccountLeases = KEYS[1]
local keyOperationIdempotency = KEYS[2]
local requestDetails = cjson.decode(ARGV[1])
if not requestDetails then
	return redis.error_reply("ERR requestDetails is nil after JSON decode")
end
local keyPrefix = ARGV[2]
local accountID = ARGV[3]
local nowMS = tonumber(ARGV[4]) 
local nowNS = tonumber(ARGV[5]) 
local operationIdempotencyTTL = tonumber(ARGV[6])
local enableDebugLogs = tonumber(ARGV[7]) == 1
local debugLogs = {}
local function debug(...)
	if enableDebugLogs then
		local args = { ... }
		table.insert(debugLogs, table.concat(args, " "))
	end
end
local function getConcurrencyCount(key)
	local count = call("ZCOUNT", key, tostring(nowMS), "+inf")
	if count == nil then
		return 0
	end
	return count
end
local function toInteger(value)
	return math.floor(value + 0.5) 
end
local function rateLimit(key, now_ns, period_ns, limit, burst, quantity)
	limit = math.max(limit, 1)
	local result = {}
	result["remaining"] = 0
	result["limit"] = burst + 1
	local emission = period_ns / limit
	result["ei"] = emission
	result["retry_at"] = now_ns + emission
	local dvt = emission * (burst + 1)
	result["dvt"] = dvt
	local tat = redis.call("GET", key)
	if not tat then
		tat = now_ns
	else
		tat = tonumber(tat)
	end
	result["tat"] = tat
	local origQuantity = quantity
	if quantity == 0 then
		quantity = 1
	end
	local increment = quantity * emission
	result["inc"] = increment
	local new_tat = tat + increment
	if now_ns > tat then
		new_tat = now_ns + increment
	end
	result["ntat"] = new_tat
	local ttl = tat - now_ns
	result["reset_after"] = ttl
	local used_tokens = math.min(math.ceil(ttl / emission), limit)
	result["u"] = used_tokens
	local allow_at = new_tat - dvt
	result["aat"] = allow_at
	local diff = now_ns - allow_at
	result["diff"] = diff
	if diff < 0 then
		if increment <= dvt then
			result["retry_after"] = -diff
			result["retry_at"] = now_ns - diff
		end
		if origQuantity > 0 then
			local next = dvt - ttl
			result["next"] = next
			result["remaining"] = 0
			result["limited"] = true
			return result
		end
	end
	if origQuantity > 0 then
		ttl = new_tat - now_ns
		result["reset_after"] = ttl
		used_tokens = math.min(math.ceil(ttl / emission), limit)
		result["u"] = used_tokens
		local expiry = string.format("%d", math.max(ttl / 1000000000, 1))
		redis.call("SET", key, new_tat, "EX", expiry)
	end
	local next = dvt - ttl
	result["next"] = next
	if next > -emission then
		local remaining = math.floor(next / emission)
		result["remaining"] = remaining
	end
	return result
end
local function throttle(key, now_ms, period_ms, limit, burst, quantity)
	limit = math.max(limit, 1)
	local result = {}
	result["remaining"] = 0
	result["limit"] = burst + 1
	local emission = period_ms / limit
	result["ei"] = emission
	result["retry_at"] = now_ms + emission
	local dvt = emission * (burst + 1)
	result["dvt"] = dvt
	local tat = redis.call("GET", key)
	if not tat then
		tat = now_ms
	else
		tat = tonumber(tat)
	end
	result["tat"] = tat
	local origQuantity = quantity
	if quantity == 0 then
		quantity = 1
	end
	local increment = quantity * emission
	result["inc"] = increment
	local new_tat = tat + increment
	if now_ms > tat then
		new_tat = now_ms + increment
	end
	result["ntat"] = new_tat
	local ttl = tat - now_ms
	result["reset_after"] = ttl
	local used_tokens = math.min(math.ceil(ttl / emission), limit)
	result["u"] = used_tokens
	local allow_at = new_tat - dvt
	result["aat"] = allow_at
	local diff = now_ms - allow_at
	result["diff"] = diff
	if diff < 0 then
		if increment <= dvt then
			result["retry_after"] = -diff
			result["retry_at"] = now_ms - diff
		end
		if origQuantity > 0 then
			local next = dvt - ttl
			result["next"] = next
			result["remaining"] = 0
			result["limited"] = true
			return result
		end
	end
	if origQuantity > 0 then
		ttl = new_tat - now_ms
		result["reset_after"] = ttl
		used_tokens = math.min(math.ceil(ttl / emission), limit)
		result["u"] = used_tokens
		local expiry = string.format("%d", math.max(ttl / 1000, 1))
		redis.call("SET", key, new_tat, "EX", expiry)
	end
	local next = dvt - ttl
	result["next"] = next
	if next > -emission then
		local remaining = math.floor(next / emission)
		result["remaining"] = remaining
	end
	return result
end
local configVersion = requestDetails.cv
local constraints = requestDetails.s
if not constraints then
	return redis.error_reply("ERR constraints array is nil")
end
local availableCapacity = nil
local limitingConstraints = {}
local exhaustedConstraints = {}
local exhaustedSet = {}
local retryAt = 0
local constraintUsage = {}
for index, value in ipairs(constraints) do
	if availableCapacity ~= nil and availableCapacity <= 0 then
		break
	end
	debug("checking constraint " .. index)
	local constraintCapacity = 0
	local constraintRetryAfter = 0
	if value.k == 1 then
		local rlRes = rateLimit(value.r.k, nowNS, value.r.p, value.r.l, value.r.b, 0)
		constraintCapacity = rlRes["remaining"] or 0
		constraintRetryAfter = toInteger(rlRes["retry_at"] / 1000000) 
		local usage = {}
		usage["l"] = value.r.l
		usage["u"] = rlRes["u"]
		table.insert(constraintUsage, usage)
	elseif value.k == 2 then
		debug("evaluating concurrency")
		local inProgressLeases = getConcurrencyCount(value.c.ilk)
		constraintCapacity = value.c.l - inProgressLeases
		constraintRetryAfter = toInteger(nowMS + value.c.ra)
		local usage = {}
		usage["l"] = value.c.l
		usage["u"] = math.max(math.min(value.c.l - constraintCapacity, value.c.l or 0), 0)
		debug("i", index, "ipl", inProgressLeases, "cc", constraintCapacity)
		table.insert(constraintUsage, usage)
	elseif value.k == 3 then
		debug("evaluating throttle")
		local maxBurst = (value.t.l or 0) + (value.t.b or 0) - 1
		local throttleRes = throttle(value.t.k, nowMS, value.t.p, value.t.l, maxBurst, 0)
		constraintCapacity = throttleRes["remaining"] or 0
		constraintRetryAfter = toInteger(throttleRes["retry_at"]) 
		local usage = {}
		usage["l"] = value.t.l
		usage["u"] = math.max(math.min(value.t.l - constraintCapacity, value.t.l or 0), 0)
		table.insert(constraintUsage, usage)
	end
	if constraintCapacity <= 0 then
		if not exhaustedSet[index] then
			table.insert(exhaustedConstraints, index)
			exhaustedSet[index] = true
		end
		if constraintRetryAfter > retryAt then
			retryAt = constraintRetryAfter
		end
	end
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
	end
end
local fairnessReduction = 0
availableCapacity = availableCapacity - fairnessReduction
local res = {}
res["s"] = 1
res["d"] = debugLogs
res["lc"] = limitingConstraints
res["ec"] = exhaustedConstraints
res["ra"] = retryAt
res["fr"] = fairnessReduction
res["a"] = availableCapacity
res["cu"] = constraintUsage
local encoded = cjson.encode(res)
if operationIdempotencyTTL > 0 then
	call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
end
return encoded