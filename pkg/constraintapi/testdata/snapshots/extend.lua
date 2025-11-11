local cjson = cjson
local function call(command, ...)
	return redis.call(command, unpack(arg))
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
local keyLeaseDetails = KEYS[4]
local keyPrefix = ARGV[1]
local accountID = ARGV[2]
local leaseIdempotencyKey = ARGV[3]
local currentLeaseID = ARGV[4]
local newLeaseID = ARGV[5]
local nowMS = tonumber(ARGV[6]) 
local leaseExpiryMS = tonumber(ARGV[7])
local operationIdempotencyTTL = tonumber(ARGV[8])
local enableDebugLogs = tonumber(ARGV[9]) == 1
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
local leaseDetails = call("HMGET", keyLeaseDetails, "lid", "oik")
if leaseDetails == false or leaseDetails == nil or leaseDetails[1] == nil or leaseDetails[2] == nil then
	local res = {}
	res["s"] = 1
	res["d"] = debugLogs
	return cjson.encode(res)
end
local leaseDetailsCurrentLeaseID = leaseDetails[1]
local leaseOperationIdempotencyKey = leaseDetails[2]
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
local storedLeaseID = leaseDetailsCurrentLeaseID
if storedLeaseID == nil or storedLeaseID == false or storedLeaseID ~= currentLeaseID then
	local res = {}
	res["s"] = 3
	res["d"] = debugLogs
	return cjson.encode(res)
end
if decode_ulid_time(storedLeaseID) < nowMS then
	local res = {}
	res["s"] = 4
	res["d"] = debugLogs
	return cjson.encode(res)
end
local constraints = requestDetails.s
for _, value in ipairs(constraints) do
	if value.k == 2 then
		call("ZADD", value.c.ilk, tostring(leaseExpiryMS), leaseIdempotencyKey)
	end
end
call("HSET", keyLeaseDetails, "lid", newLeaseID)
call("ZADD", keyAccountLeases, tostring(leaseExpiryMS), leaseIdempotencyKey)
local accountScore = call("ZSCORE", keyScavengerShard, accountID)
if accountScore == nil or accountScore == false or tonumber(accountScore) > leaseExpiryMS then
	call("ZADD", keyScavengerShard, tonumber(leaseExpiryMS), accountID)
end
local res = {}
res["s"] = 5
res["d"] = debugLogs
res["lid"] = newLeaseID
local encoded = cjson.encode(res)
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
return encoded