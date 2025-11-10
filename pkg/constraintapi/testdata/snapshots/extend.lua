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
local keyRequestState = KEYS[1]
local keyOperationIdempotency = KEYS[2]
local keyScavengerShard = KEYS[3]
local keyAccountLeases = KEYS[4]
local keyLeaseDetails = KEYS[5]
local accountID = ARGV[1]
local leaseIdempotencyKey = ARGV[2]
local currentLeaseID = ARGV[3]
local newLeaseID = ARGV[4]
local nowMS = tonumber(ARGV[5]) 
local leaseExpiryMS = tonumber(ARGV[6])
local operationIdempotencyTTL = tonumber(ARGV[7])
local enableDebugLogs = tonumber(ARGV[8]) == 1
local requestDetails = cjson.decode(call("GET", keyRequestState))
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
local storedLeaseID = call("HGET", keyLeaseDetails, "lid")
if storedLeaseID ~= currentLeaseID then
	local res = {}
	res["s"] = 1
	return cjson.encode(res)
end
if decode_ulid_time(storedLeaseID) < nowMS then
	local res = {}
	res["s"] = 2
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
local result = {}
result["lid"] = newLeaseID
local encoded = cjson.encode(result)
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))
return encoded