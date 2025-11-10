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

local keyRequestState = KEYS[1]
local keyOperationIdempotency = KEYS[2]
local keyScavengerShard = KEYS[3]
local keyAccountLeases = KEYS[4]
local keyLeaseDetails = KEYS[5]

local accountID = ARGV[1]
local leaseIdempotencyKey = ARGV[2]
local currentLeaseID = ARGV[3]
local newLeaseID = ARGV[4]
local nowMS = tonumber(ARGV[5]) --[[@as integer]]
local leaseExpiryMS = tonumber(ARGV[6])
local operationIdempotencyTTL = tonumber(ARGV[7])--[[@as integer]]
local enableDebugLogs = tonumber(ARGV[8]) == 1

---@type { k: string, e: string, f: string, s: {}[], cv: integer?, r: integer?, g: integer?, a: integer?, l: integer?, lik: string[]?, lri: table<string, string>? }
local requestDetails = cjson.decode(call("GET", keyRequestState))

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

-- Check if current lease still matches
local storedLeaseID = call("HGET", keyLeaseDetails, "lid")
if storedLeaseID ~= currentLeaseID then
	local res = {}
	res["s"] = 1
	return cjson.encode(res)
end

-- Check if lease already expired
if decode_ulid_time(storedLeaseID) < nowMS then
	local res = {}
	res["s"] = 2
	return cjson.encode(res)
end

---@type { k: integer, c: { m: integer?, s: integer?, h: string?, eh: string?, l: integer?, ilk: string?, iik: string? }?, t: { s: integer?, h: string?, eh: string?, l: integer?, b: integer?, p: integer? }?, r: { s: integer?, h: string?, eh: string?, l: integer?, p: integer? }? }[]
local constraints = requestDetails.s

for _, value in ipairs(constraints) do
	-- for concurrency constraints
	if value.k == 2 then
		call("ZADD", value.c.ilk, tostring(leaseExpiryMS), leaseIdempotencyKey)
	end
end

-- update current leaseID
call("HSET", keyLeaseDetails, "lid", newLeaseID)

-- update account leases for scavenger
call("ZADD", keyAccountLeases, tostring(leaseExpiryMS), leaseIdempotencyKey)

-- Update scavenger shard score
local accountScore = call("ZSCORE", keyScavengerShard, accountID)
if accountScore == nil or accountScore == false or tonumber(accountScore) > leaseExpiryMS then
	call("ZADD", keyScavengerShard, tonumber(leaseExpiryMS), accountID)
end

---@type { lid: string }
local result = {}

result["lid"] = newLeaseID

local encoded = cjson.encode(result)

-- Set operation idempotency TTL
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))

return encoded
