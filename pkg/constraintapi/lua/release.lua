---@module 'cjson'
local cjson = cjson

---@param command string
---@param ... string
local function call(command, ...)
	return redis.call(command, ...)
end

---@type string[]
local KEYS = KEYS

---@type string[]
local ARGV = ARGV

local keyOperationIdempotency = KEYS[1]
local keyScavengerShard = KEYS[2]
local keyAccountLeases = KEYS[3]
local keyLeaseDetails = KEYS[4]

local scopedKeyPrefix = ARGV[1]
local accountID = ARGV[2]
local currentLeaseID = ARGV[3]
local operationIdempotencyTTL = tonumber(ARGV[4])--[[@as integer]]
local enableDebugLogs = tonumber(ARGV[5]) == 1

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

-- Check if lease details still exist
local requestID = call("HGET", keyLeaseDetails, "req")
if requestID == false or requestID == nil or requestID == "" then
	local res = {}
	res["s"] = 1
	res["d"] = debugLogs
	return cjson.encode(res)
end

-- Request state must still exist
local keyRequestState = string.format("%s:rs:%s", scopedKeyPrefix, requestID)
local requestStateStr = call("GET", keyRequestState)
if requestStateStr == nil or requestStateStr == false or requestStateStr == "" then
	local res = {}
	res["s"] = 2
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

---@type { k: integer, c: { m: integer?, s: integer?, h: string?, eh: string?, l: integer?, ilk: string?, iik: string? }?, t: { s: integer?, h: string?, eh: string?, l: integer?, b: integer?, p: integer? }?, r: { s: integer?, h: string?, eh: string?, l: integer?, p: integer? }? }[]
local constraints = requestDetails.s
if not constraints then
	return redis.error_reply("ERR constraints array is nil")
end

for _, c in ipairs(constraints) do
	-- for concurrency constraints
	if c.k == 2 then
		debug("removing in progress lease", c.c.ilk)
		call("ZREM", c.c.ilk, currentLeaseID)
	end
end

-- remove lease details
call("DEL", keyLeaseDetails)

-- remove from account leases
call("ZREM", keyAccountLeases, currentLeaseID)

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
else
	-- Store request details
	call("SET", keyRequestState, cjson.encode(requestDetails))
end

---@type { s: integer, lid: string, r: integer }
local res = {}

res["s"] = 3
res["d"] = debugLogs
res["r"] = requestDetails.a

local encoded = cjson.encode(res)

-- Set operation idempotency TTL
call("SET", keyOperationIdempotency, encoded, "EX", tostring(operationIdempotencyTTL))

return encoded
