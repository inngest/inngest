local cjson = cjson
local function call(command, ...)
	return redis.call(command, ...)
end
local ulidMap = {
	["0"] = 0,
	["1"] = 1,
	["2"] = 2,
	["3"] = 3,
	["4"] = 4,
	["5"] = 5,
	["6"] = 6,
	["7"] = 7,
	["8"] = 8,
	["9"] = 9,
	["A"] = 10,
	["B"] = 11,
	["C"] = 12,
	["D"] = 13,
	["E"] = 14,
	["F"] = 15,
	["G"] = 16,
	["H"] = 17,
	["J"] = 18,
	["K"] = 19,
	["M"] = 20,
	["N"] = 21,
	["P"] = 22,
	["Q"] = 23,
	["R"] = 24,
	["S"] = 25,
	["T"] = 26,
	["V"] = 27,
	["W"] = 28,
	["X"] = 29,
	["Y"] = 30,
	["Z"] = 31,
}
local function decode_ulid_time(s)
	if #s < 10 then
		return 0
	end
	s = string.sub(s, 1, 10)
	local rev = tostring(s.reverse(s))
	local time = 0
	for i = 1, #rev do
		time = time + (ulidMap[string.sub(rev, i, i)] * math.pow(32, i - 1))
	end
	return time
end
local KEYS = KEYS
local ARGV = ARGV
local keyOperationIdempotency = KEYS[1]
local keyScavengerShard = KEYS[2]
local keyAccountLeases = KEYS[3]
local keyOldLeaseDetails = KEYS[4]
local keyNewLeaseDetails = KEYS[5]
local keyPrefix = ARGV[1]
local accountID = ARGV[2]
local currentLeaseID = ARGV[3]
local newLeaseID = ARGV[4]
local nowMS = tonumber(ARGV[5]) 
local leaseExpiryMS = tonumber(ARGV[6])
local operationIdempotencyTTL = tonumber(ARGV[7])
local enableDebugLogs = tonumber(ARGV[8]) == 1
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
if decode_ulid_time(currentLeaseID) < nowMS then
	local res = {}
	res["s"] = 1
	res["d"] = debugLogs
	return cjson.encode(res)
end
local leaseDetails = call("HMGET", keyOldLeaseDetails, "lik", "req", "rid")
if leaseDetails == false or leaseDetails == nil or leaseDetails[1] == nil or leaseDetails[2] == nil then
	local res = {}
	res["s"] = 2
	res["d"] = debugLogs
	return cjson.encode(res)
end
local hashedLeaseIdempotencyKey = leaseDetails[1]
local requestID = leaseDetails[2]
local leaseRunID = leaseDetails[3]
local keyRequestState = string.format("{%s}:%s:rs:%s", keyPrefix, accountID, requestID)
local requestStateStr = call("GET", keyRequestState)
if requestStateStr == nil or requestStateStr == false or requestStateStr == "" then
	debug(keyRequestState)
	local res = {}
	res["s"] = 3
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
for _, value in ipairs(constraints) do
	if value.k == 2 then
		call("ZREM", value.c.ilk, currentLeaseID)
		call("ZADD", value.c.ilk, tostring(leaseExpiryMS), newLeaseID)
	end
end
call("HSET", keyNewLeaseDetails, "lik", hashedLeaseIdempotencyKey, "rid", leaseRunID, "req", requestID)
call("DEL", keyOldLeaseDetails)
call("ZADD", keyAccountLeases, tostring(leaseExpiryMS), newLeaseID)
call("ZREM", keyAccountLeases, currentLeaseID)
local earliestScore = call("ZRANGE", keyAccountLeases, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if earliestScore ~= nil and earliestScore ~= false and earliestScore[2] ~= nil then
	call("ZADD", keyScavengerShard, tonumber(earliestScore[2]), accountID)
end
local res = {}
res["s"] = 4
res["d"] = debugLogs
res["lid"] = newLeaseID
local encoded = cjson.encode(res)
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
return encoded