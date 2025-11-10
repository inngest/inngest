---@module 'cjson'
local cjson = cjson

---@param command string
---@param ... string
local function call(command, ...)
	return redis.call(command, unpack(arg))
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
local operationIdempotencyTTL = tonumber(ARGV[5])--[[@as integer]]
local enableDebugLogs = tonumber(ARGV[6]) == 1

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

-- At this point, we know that
-- - The request state still exists and
-- - The lease is still active
-- - Thus, acquired capacity is still held

---@type { k: integer, c: { m: integer?, s: integer?, h: string?, eh: string?, l: integer?, ilk: string?, iik: string? }?, t: { s: integer?, h: string?, eh: string?, l: integer?, b: integer?, p: integer? }?, r: { s: integer?, h: string?, eh: string?, l: integer?, p: integer? }? }[]
local constraints = requestDetails.s

for _, value in ipairs(constraints) do
	-- for concurrency constraints
	if value.k == 2 then
		call("ZREM", value.c.ilk, leaseIdempotencyKey)
	end
end

-- remove lease details
call("DEL", keyLeaseDetails)

-- remove from account leases
call("ZREM", keyAccountLeases, leaseIdempotencyKey)

local earliestScore = call("ZRANGE", keyAccountLeases, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
if earliestScore == nil or earliestScore == false or earliestScore[2] == nil then
	-- Remove from scavenger shard if this was the last item
	call("ZREM", keyScavengerShard, accountID)
else
	-- Update to earliest score
	call("ZADD", keyScavengerShard, tonumber(earliestScore[2]), accountID)
end

-- Decrease number of active leases (and delete request state if this was the last remaining lease)
requestDetails.a = requestDetails.a - 1
if requestDetails.a == 0 then
	call("DEL", keyRequestState)
end

---@type { s: integer, lid: string, r: integer }
local res = {}

res["s"] = 5
res["d"] = debugLogs
res["r"] = requestDetails.a

local encoded = cjson.encode(res)

-- Set operation idempotency TTL
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))

return encoded
