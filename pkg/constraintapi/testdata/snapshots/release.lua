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
local operationIdempotencyTTL = tonumber(ARGV[4])
local enableDebugLogs = tonumber(ARGV[5]) == 1
local debugLogs = {}
local function debug(...)
	if enableDebugLogs then
		local args = { ... }
		table.insert(debugLogs, table.concat(args, " "))
	end
end
local opIdempotency = call("GET", keyOperationIdempotency)
if opIdempotency ~= nil and opIdempotency ~= false then
	debug("hit operation idempotency")
	return opIdempotency
end
local requestID = call("HGET", keyLeaseDetails, "req")
if requestID == false or requestID == nil or requestID == "" then
	local res = {}
	res["s"] = 1
	res["d"] = debugLogs
	return cjson.encode(res)
end
local keyRequestState = string.format("%s:rs:%s", scopedKeyPrefix, requestID)
local requestStateStr = call("GET", keyRequestState)
if requestStateStr == nil or requestStateStr == false or requestStateStr == "" then
	local res = {}
	res["s"] = 2
	res["d"] = debugLogs
	return cjson.encode(res)
end
local requestDetails = cjson.decode(requestStateStr)
if not requestDetails then
	return redis.error_reply("ERR requestDetails is nil after JSON decode")
end
local constraints = requestDetails.s
if not constraints then
	return redis.error_reply("ERR constraints array is nil")
end
for _, c in ipairs(constraints) do
	if c.k == 2 then
		debug("removing in progress lease", c.c.ilk)
		call("ZREM", c.c.ilk, currentLeaseID)
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
res["m"] = requestDetails.m
local encoded = cjson.encode(res)
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
return encoded