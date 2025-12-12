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
local requestID = ARGV[2]
local accountID = ARGV[3]
local nowMS = tonumber(ARGV[4]) 
local nowNS = tonumber(ARGV[5]) 
local leaseExpiryMS = tonumber(ARGV[6])
local scopedKeyPrefix = ARGV[7]
local initialLeaseIDs = cjson.decode(ARGV[8])
if not initialLeaseIDs then
	return redis.error_reply("ERR initialLeaseIDs is nil after JSON decode")
end
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
local function getActiveAccountLeasesCount()
	local count = call("ZCOUNT", keyAccountLeases, tostring(nowMS), "+inf")
	if count == nil then
		return 0
	end
	return count
end
local function getExpiredAccountLeasesCount()
	local count = call("ZCOUNT", keyAccountLeases, "-inf", tostring(nowMS))
	if count == nil then
		return 0
	end
	return count
end
local function getEarliestLeaseExpiry()
	local count = call("ZRANGE", keyAccountLeases, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
	if count == nil or count == false or #count == 0 then
		return 0
	end
	return tonumber(count[2])
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
	local tat = call("GET", key)
	if not tat then
		return now_ns
	end
	local raw_tat = tonumber(tat)
	if not raw_tat then
		return now_ns 
	end
	local clamped_tat = clampTat(raw_tat, now_ns, period_ns, delay_variation_tolerance)
	if raw_tat ~= clamped_tat then
		call("SET", key, clamped_tat, "KEEPTTL")
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
		call("SET", key, clamped_tat, "EX", ttl_seconds)
	end
end
local function throttle(key, now_ms, period_ms, limit, burst, quantity)
	local result = {}
	result["limit"] = burst + 1
	local emission = period_ms / math.max(limit, 1)
	result["ei"] = emission
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
	local allow_at = new_tat - dvt
	result["aat"] = allow_at
	local diff = now_ms - allow_at
	result["diff"] = diff
	local ttl = 0
	if diff < 0 then
		if increment <= dvt then
			result["retry_after"] = -diff
			ttl = tat - now_ms
			result["ttl"] = ttl
		end
		if origQuantity > 0 then
			local next = dvt - ttl
			result["next"] = next
			if next > -emission then
				result["remaining"] = math.floor(next / emission)
			end
			result["reset_after"] = ttl
			result["limited"] = true
			result["retry_at"] = now_ms + (result["retry_after"] or 0)
			return result
		end
	end
	ttl = tat - now_ms
	if origQuantity > 0 then
		ttl = new_tat - now_ms
		local expiry = string.format("%d", math.max(ttl / 1000, 1))
		redis.call("SET", key, new_tat, "EX", expiry)
	end
	result["ttl"] = ttl
	result["retry_at"] = now_ms + emission
	local next = dvt - ttl
	if next > -emission then
		local remaining = math.floor(next / emission)
		result["remaining"] = remaining
	end
	result["reset_after"] = ttl
	result["next"] = next
	return result
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
		local maxBurst = (value.t.l or 0) + (value.t.b or 0) - 1
		local throttleRes =
			throttle(value.t.k, nowMS, value.t.p, value.t.l, maxBurst, 0)
		constraintCapacity = throttleRes["remaining"]
		constraintRetryAfter = toInteger(throttleRes["retry_at"]) 
	end
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
	res["aal"] = getActiveAccountLeasesCount()
	res["eal"] = getExpiredAccountLeasesCount()
	res["ele"] = getEarliestLeaseExpiry()
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
			local maxBurst = (value.t.l or 0) + (value.t.b or 0) - 1
			throttle(value.t.k, nowMS, value.t.p, value.t.l, maxBurst, 1)
		end
	end
	local keyLeaseDetails = string.format("%s:ld:%s", scopedKeyPrefix, initialLeaseID)
	call("HSET", keyLeaseDetails, "lik", hashedLeaseIdempotencyKey, "rid", leaseRunID, "req", requestID)
	call("ZADD", keyAccountLeases, tostring(leaseExpiryMS), initialLeaseID)
	local keyLeaseConstraintCheckIdempotency = string.format("%s:ik:cc:%s", scopedKeyPrefix, hashedLeaseIdempotencyKey)
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
result["ra"] = retryAt 
result["d"] = debugLogs
result["fr"] = fairnessReduction
result["aal"] = getActiveAccountLeasesCount()
result["eal"] = getExpiredAccountLeasesCount()
result["ele"] = getEarliestLeaseExpiry()
local encoded = cjson.encode(result)
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
return encoded