local cjson = cjson
local function call(command, ...)
	return redis.call(command, ...)
end
local KEYS = KEYS
local ARGV = ARGV
local keyOperationIdempotency = KEYS[1]
local keyScavengerShard = KEYS[2]
local keyAccountLeases = KEYS[3]
local keyLeaseDetails = KEYS[4]
local scopedKeyPrefix = ARGV[1]
local accountID = ARGV[2]
local currentLeaseID = ARGV[3]
local nowMS = tonumber(ARGV[4]) 
local operationIdempotencyTTL = tonumber(ARGV[5])
local enableDebugLogs = tonumber(ARGV[6]) == 1
local forceReleaseSemaphores = tonumber(ARGV[7]) == 1
local enableCacheInvalidation = ARGV[8] == "1"
local debugLogs = {}
local function debug(...)
	if enableDebugLogs then
		local args = { ... }
		table.insert(debugLogs, table.concat(args, " "))
	end
end
local function toInteger(value)
	return math.floor(value + 0.5) 
end
local function getConcurrencyCount(key)
	local count = call("ZCOUNT", key, tostring(nowMS), "+inf")
	if count == nil then
		return 0
	end
	return count
end
local function addConcurrencyUsage(target, index, value)
	local usage = {}
	usage["i"] = index
	usage["l"] = value.c.l or 0
	usage["u"] = getConcurrencyCount(value.c.ilk)
	table.insert(target, usage)
end
local opIdempotency = call("GET", keyOperationIdempotency)
if opIdempotency ~= nil and opIdempotency ~= false then
	debug("hit operation idempotency")
	return { 1, opIdempotency }
end
local requestID = call("HGET", keyLeaseDetails, "req")
if requestID == false or requestID == nil or requestID == "" then
	local res = {}
	res["s"] = 1
	res["d"] = debugLogs
	return { 0, cjson.encode(res) }
end
local keyRequestState = string.format("%s:rs:%s", scopedKeyPrefix, requestID)
local requestStateStr = call("GET", keyRequestState)
if requestStateStr == nil or requestStateStr == false or requestStateStr == "" then
	local res = {}
	res["s"] = 2
	res["d"] = debugLogs
	return { 0, cjson.encode(res) }
end
local requestDetails = cjson.decode(requestStateStr)
if not requestDetails then
	return redis.error_reply("ERR requestDetails is nil after JSON decode")
end
local constraints = requestDetails.s
if not constraints then
	return redis.error_reply("ERR constraints array is nil")
end
local constraintUsage = {}
for index, c in ipairs(constraints) do
	if c.k == 2 then
		debug("removing in progress lease", c.c.ilk)
		call("ZREM", c.c.ilk, currentLeaseID)
		addConcurrencyUsage(constraintUsage, index, c)
	elseif c.k == 4 then
		if c.sem.rel == 0 or forceReleaseSemaphores then
			local weight = c.sem.w
			if not weight or weight <= 0 then
				weight = 1
			end
			local newVal = call("DECRBY", c.sem.k, toInteger(weight))
			if tonumber(newVal) < 0 then
				call("SET", c.sem.k, "0")
			end
		end
	end
end
call("DEL", keyLeaseDetails)
call("ZREM", keyAccountLeases, currentLeaseID)
local earliestScore = call("ZRANGE", keyAccountLeases, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if earliestScore == nil or earliestScore == false or earliestScore[2] == nil then
	call("ZREM", keyScavengerShard, accountID)
else
	call("ZADD", keyScavengerShard, tonumber(earliestScore[2]), accountID)
end
requestDetails.a = requestDetails.a - 1
if requestDetails.a == 0 then
	call("DEL", keyRequestState)
else
	call("SET", keyRequestState, cjson.encode(requestDetails))
end
local res = {}
res["s"] = 3
res["d"] = debugLogs
res["r"] = requestDetails.a
res["e"] = requestDetails.e
res["f"] = requestDetails.f
res["ai"] = requestDetails.ai
res["m"] = requestDetails.m
res["sc"] = constraints
res["cu"] = constraintUsage
local encoded = cjson.encode(res)
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
if enableCacheInvalidation then
	local cacheKeysToDelete = {}
	for _, c in ipairs(constraints) do
		if c.k == 2 and c.c then
			local scope = c.c.s or 0
			local sl = (scope == 2 and "a") or (scope == 1 and "e") or "f"
			local cacheKey
			if c.c.h ~= nil and c.c.h ~= "" then
				if scope == 0 then
					cacheKey = accountID .. ":c:" .. sl .. ":" .. requestDetails.f .. ":" .. c.c.h .. ":" .. (c.c.eh or "")
				elseif scope == 1 then
					cacheKey = accountID .. ":c:" .. sl .. ":" .. requestDetails.e .. ":" .. c.c.h .. ":" .. (c.c.eh or "")
				else
					cacheKey = accountID .. ":c:" .. sl .. ":" .. c.c.h .. ":" .. (c.c.eh or "")
				end
			elseif scope == 0 then
				cacheKey = accountID .. ":c:" .. sl .. ":" .. requestDetails.f
			elseif scope == 1 then
				cacheKey = accountID .. ":c:" .. sl .. ":" .. requestDetails.e
			else
				cacheKey = accountID .. ":c:" .. sl
			end
			table.insert(cacheKeysToDelete, scopedKeyPrefix .. ":cache:" .. cacheKey)
		end
	end
	if #cacheKeysToDelete > 0 then
		call("DEL", unpack(cacheKeysToDelete))
	end
end
return { 0, encoded }