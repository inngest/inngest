local keyUsage = KEYS[1]
local keyIdempotency = KEYS[2]
local keyOwner = KEYS[3]
local weight = tonumber(ARGV[1])
local idempotencyTTL = tonumber(ARGV[2])
local retention = tonumber(ARGV[3])
local function decodeManualOwner(value)
	local weight, request = string.match(value, "^(%d+):([^:]*)$")
	return tonumber(weight), request
end
local owned = redis.call("GET", keyOwner)
if owned ~= false and owned ~= nil then
	local held = decodeManualOwner(owned)
	if held == nil then return -1 end
	if held == 0 then return 0 end
	local usage = tonumber(redis.call("GET", keyUsage)) or 0
	if usage < held then return -1 end
	local remaining = redis.call("DECRBY", keyUsage, held)
	redis.call("SET", keyOwner, "0:", "EX", retention)
	return remaining
end
local tracked = redis.call("EXISTS", keyUsage .. ":ownership") == 1
if tracked then
	local usage = tonumber(redis.call("GET", keyUsage)) or 0
	if tracked and usage == 0 then return 0 end
	return -1
end
local existing = redis.call("GET", keyIdempotency)
if existing ~= nil and existing ~= false then
	return existing
end
local newVal = redis.call("DECRBY", keyUsage, weight)
if newVal < 0 then
	redis.call("SET", keyUsage, "0")
	newVal = 0
end
redis.call("SET", keyIdempotency, tostring(newVal), "EX", tostring(idempotencyTTL))
return newVal