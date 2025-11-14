local cjson = cjson
local function call(command, ...)
	return redis.call(command, unpack(arg))
end
local KEYS = KEYS
local ARGV = ARGV
local keyOperationIdempotency = KEYS[1]
local keyScavengerShard = KEYS[2]
local keyAccountLeases = KEYS[3]
local keyLeaseDetails = KEYS[4]
local keyPrefix = ARGV[1]
local accountID = ARGV[2]
local currentLeaseID = ARGV[3]
local operationIdempotencyTTL = tonumber(ARGV[4])
local enableDebugLogs = tonumber(ARGV[5]) == 1
local debugLogs = {}
local function debug(...)
	if enableDebugLogs then
		table.insert(debugLogs, table.concat(arg, " "))
	end
end
local opIdempotency = call("GET", keyOperationIdempotency)
if opIdempotency ~= nil and opIdempotency ~= false then
	debug("hit operation idempotency")
	return opIdempotency
end
local leaseDetails = call("HMGET", keyLeaseDetails, "lik", "oik", "rid")
if
	leaseDetails == false
	or leaseDetails == nil
	or leaseDetails[1] == nil
	or leaseDetails[1] == ""
	or leaseDetails[2] == nil
	or leaseDetails[2] == ""
then
	local res = {}
	res["s"] = 1
	res["d"] = debugLogs
	return cjson.encode(res)
end
local leaseIdempotencyKey = leaseDetails[1]
local leaseOperationIdempotencyKey = leaseDetails[2]
local leaseRunID = leaseDetails[3]
local keyRequestState = string.format("{%s}:%s:rs:%s", keyPrefix, accountID, leaseOperationIdempotencyKey)
local requestStateStr = call("GET", keyRequestState)
if requestStateStr == nil or requestStateStr == false or requestStateStr == "" then
	debug(keyRequestState)
	local res = {}
	res["s"] = 2
	res["d"] = debugLogs
	return cjson.encode(res)
end
local requestDetails = cjson.decode(requestStateStr)
local constraints = requestDetails.s
for _, value in ipairs(constraints) do
	if value.k == 2 then
		call("ZREM", value.c.ilk, leaseIdempotencyKey)
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
end
local res = {}
res["s"] = 3
res["d"] = debugLogs
res["r"] = requestDetails.a
local encoded = cjson.encode(res)
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
return encoded