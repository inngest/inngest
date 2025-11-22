local cjson = cjson
local function call(command, ...)
	return redis.call(command, ...)
end
local KEYS = KEYS
local ARGV = ARGV
local keyRequestState = KEYS[1]
local keyOperationIdempotency = KEYS[2]
local keyConstraintCheckIdempotency = KEYS[3]
local keyScavengerShard = KEYS[4]
local keyAccountLeases = KEYS[5]
local requestDetails = cjson.decode(ARGV[1])
local accountID = ARGV[2]
local nowMS = tonumber(ARGV[3]) 
local nowNS = tonumber(ARGV[4]) 
local leaseExpiryMS = tonumber(ARGV[5])
local keyPrefix = ARGV[6]
local initialLeaseIDs = cjson.decode(ARGV[7])
if not initialLeaseIDs then
	return redis.error_reply("ERR initialLeaseIDs is nil after JSON decode")
end
local hashedOperationIdempotencyKey = ARGV[8]
local operationIdempotencyTTL = tonumber(ARGV[9])
local constraintCheckIdempotencyTTL = tonumber(ARGV[10])
local enableDebugLogs = tonumber(ARGV[11]) == 1
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
		return { 0, now_ns + period_ns }
	end
	local emission_interval = period_ns / limit
	local total_capacity = burst + 1
	local delay_variation_tolerance = emission_interval * total_capacity
	local tat = retrieveAndNormalizeTat(key, now_ns, period_ns, delay_variation_tolerance)
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
		return { 0, allow_at }
	else
		local ttl = new_tat - now_ns
		local next = delay_variation_tolerance - ttl
		local remaining = 0
		if next > -emission_interval then
			remaining = math.floor(next / emission_interval)
		end
		return { remaining, 0 }
	end
end
local function rateLimitUpdate(key, now_ns, period_ns, limit, capacity, burst)
	if limit == 0 then
		return
	end
	local emission_interval = period_ns / limit
	local total_capacity = (burst or 0) + 1
	local delay_variation_tolerance = emission_interval * total_capacity
	local tat = retrieveAndNormalizeTat(key, now_ns, period_ns, delay_variation_tolerance)
	local increment = math.max(capacity, 1) * emission_interval
	local new_tat
	if now_ns > tat then
		new_tat = now_ns + increment
	else
		new_tat = tat + increment
	end
	if capacity > 0 then
		local clamped_tat = clampTat(new_tat, now_ns, period_ns, delay_variation_tolerance)
		local ttl_ns = clamped_tat - now_ns
		local ttl_seconds = math.ceil(ttl_ns / 1000000000) 
		redis.call("SET", key, clamped_tat, "EX", ttl_seconds)
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
	if final_capacity < 1 then
		local next_available_at_ms = tat - total_capacity_time + emission
		return { final_capacity, math.ceil(next_available_at_ms) }
	else
		return { final_capacity, 0 }
	end
end
local function throttleUpdate(key, now_ms, period_ms, limit, capacity)
	local emission = period_ms / math.max(limit, 1)
	local tat = redis.call("GET", key)
	if not tat then
		tat = now_ms
	else
		tat = tonumber(tat)
	end
	local new_tat = tat + (math.max(capacity, 1) * emission)
	if capacity > 0 then
		local expiry = string.format("%d", period_ms / 1000)
		if expiry == "0" then
			expiry = "1"
		end
		call("SET", key, new_tat, "EX", expiry)
	end
end
local requested = requestDetails.r
local configVersion = requestDetails.cv
local constraints = requestDetails.s
if not constraints then
	return redis.error_reply("ERR constraints array is nil")
end
local opIdempotency = call("GET", keyOperationIdempotency)
if opIdempotency ~= nil and opIdempotency ~= false then
	debug("hit operation idempotency")
	return opIdempotency
end
local availableCapacity = requested
local limitingConstraints = {}
local retryAt = 0
local skipGCRA = call("EXISTS", keyConstraintCheckIdempotency) == 1
for index, value in ipairs(constraints) do
	if availableCapacity <= 0 then
		break
	end
	debug("checking constraint " .. index)
	local constraintCapacity = 0
	local constraintRetryAfter = 0
	if skipGCRA and (value.k == 1 or value.k == 3) then
		constraintCapacity = availableCapacity
		debug("skipping gcra" .. index)
	elseif value.k == 1 then
		local burst = math.floor(value.r.l / 10) 
		local rlRes = rateLimitCapacity(value.r.k, nowNS, value.r.p, value.r.l, burst)
		constraintCapacity = rlRes[1]
		constraintRetryAfter = toInteger(rlRes[2] / 1000000) 
	elseif value.k == 2 then
		debug("evaluating concurrency")
		local inProgressItems = getConcurrencyCount(value.c.iik)
		local inProgressLeases = getConcurrencyCount(value.c.ilk)
		local inProgressTotal = inProgressItems + inProgressLeases
		constraintCapacity = value.c.l - inProgressTotal
	elseif value.k == 3 then
		debug("evaluating throttle")
		local throttleRes = throttleCapacity(value.t.k, nowMS, value.t.p, value.t.l, value.t.b)
		constraintCapacity = throttleRes[1]
		constraintRetryAfter = toInteger(throttleRes[2]) 
	end
	if constraintCapacity < availableCapacity then
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
if availableCapacity <= 0 then
	local res = {}
	res["s"] = 2
	res["lc"] = limitingConstraints
	res["ra"] = retryAt
	res["d"] = debugLogs
	res["fr"] = fairnessReduction
	return cjson.encode(res)
end
local granted = availableCapacity
local grantedLeases = {}
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
		elseif value.k == 1 then
			debug("updating rate limit", value.r.h)
			rateLimitUpdate(value.r.k, nowNS, value.r.p, value.r.l, 1, value.r.b)
		elseif value.k == 2 then
			call("ZADD", value.c.ilk, tostring(leaseExpiryMS), initialLeaseID)
		elseif value.k == 3 then
			throttleUpdate(value.t.k, nowMS, value.t.p, value.t.l, 1)
		end
	end
	local keyLeaseDetails = string.format("{%s}:%s:ld:%s", keyPrefix, accountID, initialLeaseID)
	call(
		"HSET",
		keyLeaseDetails,
		"lik",
		hashedLeaseIdempotencyKey,
		"rid",
		leaseRunID,
		"oik",
		hashedOperationIdempotencyKey
	)
	call("ZADD", keyAccountLeases, tostring(leaseExpiryMS), initialLeaseID)
	local keyLeaseConstraintCheckIdempotency =
		string.format("{%s}:%s:ik:cc:%s", keyPrefix, accountID, hashedLeaseIdempotencyKey)
	call("SET", keyLeaseConstraintCheckIdempotency, tostring(nowMS), "EX", tostring(constraintCheckIdempotencyTTL))
	local leaseObject = {}
	leaseObject["lid"] = initialLeaseID
	leaseObject["lik"] = hashedLeaseIdempotencyKey
	table.insert(grantedLeases, leaseObject)
end
call("SET", keyConstraintCheckIdempotency, tostring(nowMS), "EX", tostring(constraintCheckIdempotencyTTL))
requestDetails.g = availableCapacity
requestDetails.a = availableCapacity
call("SET", keyRequestState, cjson.encode(requestDetails))
local accountScore = call("ZSCORE", keyScavengerShard, accountID)
if accountScore == nil or accountScore == false or tonumber(accountScore) > leaseExpiryMS then
	call("ZADD", keyScavengerShard, tonumber(leaseExpiryMS), accountID)
end
local result = {}
result["s"] = 3
result["r"] = requested
result["g"] = granted
result["l"] = grantedLeases
result["lc"] = limitingConstraints
result["d"] = debugLogs
result["fr"] = fairnessReduction
local encoded = cjson.encode(result)
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
return encoded