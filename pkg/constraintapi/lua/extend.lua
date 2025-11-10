---@module 'cjson'
local cjson = cjson

---@param command string
---@param ... string
local function call(command, ...)
	return redis.call(command, unpack(arg))
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
local keyLeaseDetails = KEYS[4]

local keyPrefix = ARGV[1]
local accountID = ARGV[2]
local leaseIdempotencyKey = ARGV[3]
local currentLeaseID = ARGV[4]
local newLeaseID = ARGV[5]
local nowMS = tonumber(ARGV[6]) --[[@as integer]]
local leaseExpiryMS = tonumber(ARGV[7])
local operationIdempotencyTTL = tonumber(ARGV[8])--[[@as integer]]
local enableDebugLogs = tonumber(ARGV[9]) == 1

---@type string[]
local debugLogs = {}
---@param message string
local function debug(...)
	if enableDebugLogs then
		table.insert(debugLogs, table.concat(arg, " "))
	end
end

-- Handle operation idempotency
local opIdempotency = call("GET", keyOperationIdempotency)
if opIdempotency ~= nil and opIdempotency ~= false then
	debug("hit operation idempotency")

	-- Return idempotency state to user (same as initial response)
	return opIdempotency
end

-- Check if lease details still exist
local leaseDetails = call("HMGET", keyLeaseDetails, "lid", "oik")
if leaseDetails == false or leaseDetails == nil or leaseDetails[1] == nil or leaseDetails[2] == nil then
	local res = {}
	res["s"] = 1
	res["d"] = debugLogs
	return cjson.encode(res)
end

local leaseDetailsCurrentLeaseID = leaseDetails[1]
local leaseOperationIdempotencyKey = leaseDetails[2]

-- Request state must still exist
local keyRequestState = string.format("{%s}:%s:rs:%s", keyPrefix, accountID, leaseOperationIdempotencyKey)
local requestStateStr = call("GET", keyRequestState)
if requestStateStr == nil or requestStateStr == false or requestStateStr == "" then
	debug(keyRequestState)

	local res = {}
	res["s"] = 2
	res["d"] = debugLogs
	return cjson.encode(res)
end

---@type { k: string, e: string, f: string, s: {}[], cv: integer?, r: integer?, g: integer?, a: integer?, l: integer?, lik: string[]?, lri: table<string, string>? }
local requestDetails = cjson.decode(requestStateStr)

-- Check if current lease still matches
local storedLeaseID = leaseDetailsCurrentLeaseID
if storedLeaseID == nil or storedLeaseID == false or storedLeaseID ~= currentLeaseID then
	local res = {}
	res["s"] = 3
	res["d"] = debugLogs
	return cjson.encode(res)
end

-- Check if lease already expired
if decode_ulid_time(storedLeaseID) < nowMS then
	local res = {}
	res["s"] = 4
	res["d"] = debugLogs
	return cjson.encode(res)
end

-- At this point, we know that
-- - The request state still exists and
-- - The lease is still active
-- - Thus, acquired capacity is still held

---@type { k: integer, c: { m: integer?, s: integer?, h: string?, eh: string?, l: integer?, ilk: string?, iik: string? }?, t: { s: integer?, h: string?, eh: string?, l: integer?, b: integer?, p: integer? }?, r: { s: integer?, h: string, eh: string, l: integer, p: integer, k: string }? }[]
local constraints = requestDetails.s

for _, value in ipairs(constraints) do
	-- for concurrency constraints
	if value.k == 2 then
		call("ZADD", value.c.ilk, tostring(leaseExpiryMS), leaseIdempotencyKey)
	end
end

-- update current leaseID to new lease ID
call("HSET", keyLeaseDetails, "lid", newLeaseID)

-- update account leases for scavenger (do not clean up active lease)
call("ZADD", keyAccountLeases, tostring(leaseExpiryMS), leaseIdempotencyKey)

-- Update scavenger shard score (do not process account too early)
local accountScore = call("ZSCORE", keyScavengerShard, accountID)
if accountScore == nil or accountScore == false or tonumber(accountScore) > leaseExpiryMS then
	call("ZADD", keyScavengerShard, tonumber(leaseExpiryMS), accountID)
end

---@type { s: integer, lid: string }
local res = {}

res["s"] = 5
res["d"] = debugLogs
res["lid"] = newLeaseID

local encoded = cjson.encode(res)

-- Set operation idempotency TTL
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))

return encoded
