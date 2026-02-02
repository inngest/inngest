---@module 'cjson'
local cjson = cjson

---@param command string
---@param ... string
local function call(command, ...)
	return redis.call(command, ...)
end

-- This table is used when decoding ulid timestamps.
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

--- decode_ulid_time decodes a ULID into a ms epoch
local function decode_ulid_time(s)
	if #s < 10 then
		return 0
	end

	-- Take first 10 characters of the ULID, which is the time portion.
	s = string.sub(s, 1, 10)
	local rev = tostring(s.reverse(s))
	local time = 0
	for i = 1, #rev do
		time = time + (ulidMap[string.sub(rev, i, i)] * math.pow(32, i - 1))
	end
	return time
end

---@type string[]
local KEYS = KEYS

---@type string[]
local ARGV = ARGV

local keyOperationIdempotency = KEYS[1]
local keyScavengerShard = KEYS[2]
local keyAccountLeases = KEYS[3]
local keyOldLeaseDetails = KEYS[4]
local keyNewLeaseDetails = KEYS[5]

local scopedKeyPrefix = ARGV[1]
local accountID = ARGV[2]
local currentLeaseID = ARGV[3]
local newLeaseID = ARGV[4]
local nowMS = tonumber(ARGV[5]) --[[@as integer]]
local leaseExpiryMS = tonumber(ARGV[6])
local operationIdempotencyTTL = tonumber(ARGV[7])--[[@as integer]]
local enableDebugLogs = tonumber(ARGV[8]) == 1

---@type string[]
local debugLogs = {}
---@param ... string
local function debug(...)
	if enableDebugLogs then
		local args = { ... }
		table.insert(debugLogs, table.concat(args, " "))
	end
end

-- Handle operation idempotency
local opIdempotency = call("GET", keyOperationIdempotency)
if opIdempotency ~= nil and opIdempotency ~= false then
	debug("hit operation idempotency")

	-- Return idempotency state to user (same as initial response)
	return opIdempotency
end

-- Check if lease already expired
if decode_ulid_time(currentLeaseID) < nowMS then
	local res = {}
	res["s"] = 1
	res["d"] = debugLogs
	return cjson.encode(res)
end

-- Check if lease details still exist
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

-- Request state must still exist
local keyRequestState = string.format("%s:rs:%s", scopedKeyPrefix, requestID)
local requestStateStr = call("GET", keyRequestState)
if requestStateStr == nil or requestStateStr == false or requestStateStr == "" then
	debug(keyRequestState)

	local res = {}
	res["s"] = 3
	res["d"] = debugLogs
	return cjson.encode(res)
end

---@type { k: string, e: string, f: string, s: {}[], cv: integer?, r: integer?, g: integer?, a: integer?, l: integer?, lik: string[]?, lri: table<string, string>?, m: { ss: integer?, sl: integer?, sm: integer? }? }
local requestDetails = cjson.decode(requestStateStr)
if not requestDetails then
	return redis.error_reply("ERR requestDetails is nil after JSON decode")
end

-- At this point, we know that
-- - The request state still exists and
-- - The lease is still active
-- - Thus, acquired capacity is still held

---@type { k: integer, c: { m: integer?, s: integer?, h: string?, eh: string?, l: integer?, ilk: string?, iik: string? }?, t: { s: integer?, h: string?, eh: string?, l: integer?, b: integer?, p: integer? }?, r: { s: integer?, h: string, eh: string, l: integer, p: integer, k: string }? }[]
local constraints = requestDetails.s
if not constraints then
	return redis.error_reply("ERR constraints array is nil")
end

for _, value in ipairs(constraints) do
	-- for concurrency constraints, update score to new expiry
	if value.k == 2 then
		call("ZREM", value.c.ilk, currentLeaseID)
		call("ZADD", value.c.ilk, tostring(leaseExpiryMS), newLeaseID)
	end
end

-- update lease details
call("HSET", keyNewLeaseDetails, "lik", hashedLeaseIdempotencyKey, "rid", leaseRunID, "req", requestID)
call("DEL", keyOldLeaseDetails)

-- update account leases for scavenger (do not clean up active lease)
call("ZADD", keyAccountLeases, tostring(leaseExpiryMS), newLeaseID)
call("ZREM", keyAccountLeases, currentLeaseID)

local earliestScore = call("ZRANGE", keyAccountLeases, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if earliestScore ~= nil and earliestScore ~= false and earliestScore[2] ~= nil then
	-- Update to earliest score
	call("ZADD", keyScavengerShard, tonumber(earliestScore[2]), accountID)
end

---@type { s: integer, lid: string }
local res = {}

res["s"] = 4
res["d"] = debugLogs
res["lid"] = newLeaseID

local encoded = cjson.encode(res)

-- Set operation idempotency TTL
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))

return encoded
