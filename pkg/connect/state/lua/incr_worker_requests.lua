--[[

Output:
  0: Successfully incremented lease counter
  1: Worker capacity exceeded

ARGV[1]: TTL in seconds for the counter and mapping keys
ARGV[2]: Instance ID of the worker
ARGV[3]: Request ID
]]

local capacityKey = KEYS[1]
local counterKey = KEYS[2]
local leaseWorkerKey = KEYS[3]

local counterTTL = tonumber(ARGV[1])
local instanceID = ARGV[2]
local requestID = ARGV[3]

-- Get the worker's capacity limit (returns a string)
local capacity = redis.call("GET", capacityKey)

-- If no capacity limit is set, don't track leases
-- redis nil becomes false in lua: https://redis.io/docs/latest/commands/eval/#conversion-between-lua-and-redis-data-types
if capacity == nil or capacity == 0 or capacity == false or capacity == "0" then
  return 0
end

capacity = tonumber(capacity)

-- Get current number of active leases
local currentLeases = tonumber(redis.call("GET", counterKey) or "0")

-- If current leases is not a number (and doesn't exist), we assume that
-- there are no active leases
if currentLeases == nil or currentLeases == 0 or currentLeases == false then
  currentLeases = 0
end

-- Check if at capacity
if currentLeases >= capacity then
  return 1
end

-- Increment the lease counter
redis.call("INCR", counterKey)

-- Set/refresh TTL on the counter to ensure it expires if worker stops processing
redis.call("EXPIRE", counterKey, counterTTL)

-- Store the mapping of request ID to worker instance ID
redis.call("SET", leaseWorkerKey, instanceID, "EX", counterTTL)

return 0
