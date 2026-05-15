local keyUsage = KEYS[1]
local keyIdempotency = KEYS[2]
local weight = tonumber(ARGV[1])
local idempotencyTTL = tonumber(ARGV[2])
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