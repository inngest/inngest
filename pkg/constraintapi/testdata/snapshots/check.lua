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
local function clampTat(tat, now_ns, period_ns, delay_variation_tolerance)
	local max_reasonable_tat = now_ns + period_ns + delay_variation_tolerance
	local min_reasonable_tat = now_ns - period_ns 
	if tat > max_reasonable_tat then
		return toInteger(max_reasonable_tat)
	elseif tat < min_reasonable_tat then
		return toInteger(now_ns) 
	else
		return toInteger(tat)
	end
end
local function retrieveAndNormalizeTat(key, now_ns, period_ns, delay_variation_tolerance)
	local tat = redis.call("GET", key)
	if not tat then
		return now_ns
	end
	local raw_tat = tonumber(tat)
	if not raw_tat then
		return now_ns 
	end
	local clamped_tat = clampTat(raw_tat, now_ns, period_ns, delay_variation_tolerance)
	if raw_tat ~= clamped_tat then
		redis.call("SET", key, clamped_tat, "KEEPTTL")
	end
	return clamped_tat
end
local function rateLimitCapacity(key, now_ns, period_ns, limit, burst)
	if limit == 0 then
		return { 0, now_ns + period_ns, 0 }
	end
	local emission_interval = period_ns / limit
	local total_capacity = burst + 1
	local delay_variation_tolerance = emission_interval * total_capacity
	local tat = retrieveAndNormalizeTat(key, now_ns, period_ns, delay_variation_tolerance)
	local used_tokens = 0
	if tat > now_ns then
		local consumed_time = tat - now_ns
		used_tokens = math.min(math.ceil(consumed_time / emission_interval), limit)
	end
	local increment = 1 * emission_interval
	local new_tat
	if now_ns > tat then
		new_tat = now_ns + increment
	else
		new_tat = tat + increment
	end
	local allow_at = new_tat - delay_variation_tolerance
	local diff = now_ns - allow_at
	if diff < 0 then
		return { 0, allow_at, used_tokens }
	else
		local current_ttl = math.max(tat - now_ns, 0)
		local next = delay_variation_tolerance - current_ttl
		local remaining = 0
		if next > -emission_interval then
			remaining = math.floor(next / emission_interval)
		end
		local new_tat_after_consumption = math.max(tat, now_ns) + remaining * emission_interval
		local next_available_at_ns = new_tat_after_consumption - delay_variation_tolerance + emission_interval
		return { remaining, toInteger(next_available_at_ns), used_tokens }
	end
end
local function throttleCapacity(key, now_ms, period_ms, limit, burst)
	local emission = period_ms / math.max(limit, 1)
	local total_capacity_time = emission * (limit + burst)
	local tat = call("GET", key)
	if not tat then
		tat = now_ms
	else
		tat = tonumber(tat)
	end
	local time_capacity_remain = now_ms + total_capacity_time - tat
	local capacity = math.floor(time_capacity_remain / emission)
	local final_capacity = math.min(capacity, limit + burst)
	local new_tat_after_consumption = math.max(tat, now_ms) + final_capacity * emission
	local next_available_at_ms = math.ceil(new_tat_after_consumption - total_capacity_time + emission)
	return { final_capacity, next_available_at_ms }
end
local configVersion = requestDetails.cv
local constraints = requestDetails.s
if not constraints then
	return redis.error_reply("ERR constraints array is nil")
end
local availableCapacity = nil
local limitingConstraints = {}
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
		local burst = math.floor(value.r.l / 10) 
		local rlRes = rateLimitCapacity(value.r.k, nowNS, value.r.p, value.r.l, burst)
		constraintCapacity = rlRes[1]
		constraintRetryAfter = toInteger(rlRes[2] / 1000000) 
		local usage = {}
		usage["l"] = value.r.l
		usage["u"] = rlRes[3] 
		table.insert(constraintUsage, usage)
	elseif value.k == 2 then
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
		debug("evaluating throttle")
		local throttleRes = throttleCapacity(value.t.k, nowMS, value.t.p, value.t.l, value.t.b)
		constraintCapacity = throttleRes[1]
		constraintRetryAfter = toInteger(throttleRes[2]) 
		local usage = {}
		usage["l"] = value.t.l
		usage["u"] = math.max(math.min(value.t.l - constraintCapacity, value.t.l or 0), 0)
		table.insert(constraintUsage, usage)
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
		if constraintRetryAfter > retryAt then
			retryAt = constraintRetryAfter
		end
	end
end
local fairnessReduction = 0
availableCapacity = availableCapacity - fairnessReduction
local res = {}
res["s"] = 1
res["d"] = debugLogs
res["lc"] = limitingConstraints
res["ra"] = retryAt
res["fr"] = fairnessReduction
res["a"] = availableCapacity
res["cu"] = constraintUsage
local encoded = cjson.encode(res)
if operationIdempotencyTTL > 0 then
	call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
end
return encoded