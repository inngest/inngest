--
-- semaphore_adjust_capacity
-- Idempotently adjusts capacity by delta.
--
-- Returns {capacity, applied} where applied is 1 when the delta was applied
-- and 0 when this call was an idempotent replay of a previously-seen key.
--

local keyCapacity = KEYS[1]
local keyIdempotency = KEYS[2]

local delta = tonumber(ARGV[1])
local idempotencyTTL = tonumber(ARGV[2])

-- Check idempotency
local existing = redis.call("GET", keyIdempotency)
if existing ~= nil and existing ~= false then
	return {tonumber(existing), 0}
end

local newCapacity = redis.call("INCRBY", keyCapacity, delta)
if tonumber(newCapacity) < 0 then
	redis.call("SET", keyCapacity, "0")
	newCapacity = 0
end
redis.call("SET", keyIdempotency, tostring(newCapacity), "EX", tostring(idempotencyTTL))

return {tonumber(newCapacity), 1}
