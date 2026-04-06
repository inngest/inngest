--
-- semaphore_set_capacity
-- Idempotently sets total capacity for a named semaphore.
--

local keyCapacity = KEYS[1]
local keyIdempotency = KEYS[2]

local capacity = ARGV[1]
local idempotencyTTL = tonumber(ARGV[2])

-- Check idempotency
local existing = redis.call("GET", keyIdempotency)
if existing ~= nil and existing ~= false then
	return existing
end

redis.call("SET", keyCapacity, capacity)
redis.call("SET", keyIdempotency, capacity, "EX", tostring(idempotencyTTL))

return capacity
