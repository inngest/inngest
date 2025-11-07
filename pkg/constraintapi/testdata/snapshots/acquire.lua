local cjson = cjson
local function call(command, ...)
	redis.call(command, unpack(arg))
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
local operationIdempotencyKey = ARGV[8]
local operationIdempotencyTTL = tonumber(ARGV[9])
local constraintCheckIdempotencyTTL = tonumber(ARGV[10])
local function getConcurrencyCount(key)
	local count = call("ZCOUNT", key, tostring(nowMS), "+inf")
	return count
end
local function rateLimitCapacity(key, now_ns, period_ns, limit, burst)
	if limit == 0 then
		return { 0, now_ns + period_ns }
	end
	local emission_interval = period_ns / limit
	local total_capacity = burst + 1
	local delay_variation_tolerance = emission_interval * total_capacity
	local tat = call("GET", key)
	if not tat then
		tat = now_ns
	else
		tat = tonumber(tat)
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
local function rateLimitUpdate(key, now_ns, period_ns, limit, capacity)
	if limit == 0 then
		return
	end
	local emission_interval = period_ns / limit
	local tat = call("GET", key)
	if not tat then
		tat = now_ns
	else
		tat = tonumber(tat)
	end
	local increment = math.max(capacity, 1) * emission_interval
	local new_tat
	if now_ns > tat then
		new_tat = now_ns + increment
	else
		new_tat = tat + increment
	end
	if capacity > 0 then
		local ttl_ns = new_tat - now_ns
		local ttl_seconds = math.ceil(ttl_ns / 1000000000) 
		call("SET", key, new_tat, "EX", ttl_seconds)
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
		call("SET", key, new_tat, "EX", expiry)
	end
end
local requested = requestDetails.r
local configVersion = requestDetails.cv
local constraints = requestDetails.s
local opIdempotency = call("GET", keyOperationIdempotency)
if opIdempotency ~= nil and opIdempotency ~= false then
	return { 1, opIdempotency }
end
local availableCapacity = requested
local limitingConstraint = -1
local retryAt = 0
local skipGCRA = false
for index, value in ipairs(constraints) do
	if availableCapacity <= 0 then
		break
	end
	local constraintCapacity = 0
	local constraintRetryAfter = 0
	if skipGCRA then
		constraintCapacity = availableCapacity
	elseif value.k == 1 then
		local gcraRes = rateLimitCapacity(value.r.h, nowNS, value.r.p, value.r.l, 0)
		constraintCapacity = gcraRes[0]
		constraintRetryAfter = gcraRes[1]
	elseif value.k == 2 then
		local inProgressItems = getConcurrencyCount(value.c.iik)
		local inProgressLeases = getConcurrencyCount(value.c.ilk)
		local inProgressTotal = inProgressItems + inProgressLeases
		constraintCapacity = value.c.l - inProgressTotal
	elseif value.k == 3 then
		local gcraRes = throttleCapacity(value.t.h, nowMS, value.t.p, value.t.l, value.t.b)
		constraintCapacity = gcraRes[0]
		constraintRetryAfter = gcraRes[1]
	end
	if constraintCapacity < availableCapacity then
		availableCapacity = constraintCapacity
		limitingConstraint = index
		if constraintRetryAfter > retryAt then
			retryAt = constraintRetryAfter
		end
	end
end
local fairnessReduction = 0
availableCapacity = availableCapacity - fairnessReduction
if availableCapacity <= 0 then
	return { 2, limitingConstraint }
end
local granted = availableCapacity
local grantedLeases = {}
for i = 1, granted, 1 do
	local leaseIdempotencyKey = requestDetails.lik[i]
	local leaseRunID = requestDetails.lri[leaseIdempotencyKey]
	local initialLeaseID = initialLeaseIDs[i]
	for _, value in ipairs(constraints) do
		if skipGCRA then
		elseif value.k == 1 then
			rateLimitUpdate(value.r.h, nowNS, value.r.p, value.r.l, 1)
		elseif value.k == 2 then
			call("ZADD", value.c.ilk, tostring(leaseExpiryMS), leaseIdempotencyKey)
		elseif value.k == 3 then
			throttleUpdate(value.t.h, nowMS, value.t.p, value.t.l, 1)
		end
	end
	local keyLeaseDetails = string.format("{%s}:%s:ld:%s", keyPrefix, accountID, leaseIdempotencyKey)
	call("HSET", keyLeaseDetails, "lid", initialLeaseID, "rid", leaseRunID, "oik", operationIdempotencyKey)
	call("ZADD", keyAccountLeases, tostring(leaseExpiryMS), leaseIdempotencyKey)
	local leaseObject = {}
	leaseObject["lid"] = initialLeaseID
	leaseObject["lik"] = leaseIdempotencyKey
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
result["r"] = requested
result["g"] = granted
result["l"] = grantedLeases
local encoded = cjson.encode(result)
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
return { 3, encoded }