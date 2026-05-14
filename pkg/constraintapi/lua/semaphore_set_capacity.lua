--
-- semaphore_set_capacity
-- Idempotently sets total capacity for a named semaphore.
--
-- Returns {capacity, applied} where applied is 1 when the capacity was set
-- and 0 when this call was an idempotent replay of a previously-seen key.
--

local keyCapacity = KEYS[1]
local keyIdempotency = KEYS[2]

local capacity = ARGV[1]
local idempotencyTTL = tonumber(ARGV[2])

-- Check idempotency
local existing = redis.call("GET", keyIdempotency)
if existing ~= nil and existing ~= false then
	return {tonumber(existing), 0}
end

redis.call("SET", keyCapacity, capacity)
redis.call("SET", keyIdempotency, capacity, "EX", tostring(idempotencyTTL))

return {tonumber(capacity), 1}
