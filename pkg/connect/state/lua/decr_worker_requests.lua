--[[

Decrements the worker lease counter and manages TTL/cleanup.

Output:
  0: Successfully decremented, counter deleted (reached 0 or below)
  1: Successfully decremented, counter still active
  2: Counter doesn't exist, nothing to decrement

ARGV[1]: TTL in seconds for the counter key
]]

local counterKey = KEYS[1]
local leaseWorkerKey = KEYS[2]
local counterTTL = tonumber(ARGV[1])

-- Check if counter exists
local currentValue = redis.call("GET", counterKey)

-- If counter doesn't exist, nothing to decrement
if currentValue == false or currentValue == nil then
  return 2
end

-- Decrement the counter
local newValue = redis.call("DECR", counterKey)

-- If counter is now 0 or negative, delete it and the mapping
if newValue <= 0 then
  redis.call("DEL", counterKey)
  redis.call("DEL", leaseWorkerKey)
  return 0
end

-- Counter is still positive, refresh TTL
redis.call("EXPIRE", counterKey, counterTTL)
-- Also delete the specific request's worker mapping
redis.call("DEL", leaseWorkerKey)
return 1
