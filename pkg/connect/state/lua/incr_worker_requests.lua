--[[

Output:
  0: Successfully added lease to set
  1: Worker capacity exceeded

ARGV[1]: TTL in seconds for the set
ARGV[2]: Request lease duration in seconds
ARGV[3]: Instance ID of the worker
ARGV[4]: Request ID
ARGV[5]: Expiration time as Unix timestamp (score for sorted set)
ARGV[6]: Current time as Unix timestamp in seconds
]]

local workerTotalCapacityKey = KEYS[1]
local workerRequestsKey = KEYS[2]
local requestWorkerKey = KEYS[3]

local setTTL = tonumber(ARGV[1])
local requestLeaseDuration = tonumber(ARGV[2])
local instanceID = ARGV[3]
local requestID = ARGV[4]
local expirationTime = ARGV[5]
local currentTime = ARGV[6]

-- Separate TTL variables for different components
local workerRequestsSetTTL = setTTL
local requestWorkerMappingTTL = requestLeaseDuration

-- Get the worker's capacity limit (returns a string)
local capacity = redis.call("GET", workerTotalCapacityKey)

-- If no capacity limit is set, don't track leases
-- redis nil becomes false in lua: https://redis.io/docs/latest/commands/eval/#conversion-between-lua-and-redis-data-types
if capacity == nil or capacity == 0 or capacity == false or capacity == "0" then
  return 0
end

capacity = tonumber(capacity)

-- Get current time to filter out expired leases
-- previous second, this makes us very sensitive to time changes
-- Should we use the logical clock instead?
local currentTimeUnix = tonumber(currentTime)

-- Remove expired leases from the set first
-- with a "(" prefix, we would remove all leases inf < leases <= currentTimeUnix
redis.call("ZREMRANGEBYSCORE", workerRequestsKey, "-inf", "("..currentTimeUnix)

-- Get current number of active leases (those with expiration time >= current time)
-- without a "(" prefix we do currentTimeUnix < lease < +inf
local currentLeases = redis.call("ZCOUNT", workerRequestsKey, tostring(currentTimeUnix), "+inf")

-- Check if at capacity
if currentLeases >= capacity then
  return 1
end

-- Add the lease to the sorted set with expiration time as score
local expTime = tonumber(expirationTime)
redis.call("ZADD", workerRequestsKey, expTime, requestID)

-- Set/refresh TTL on the set to ensure it expires if worker stops processing
redis.call("EXPIRE", workerRequestsKey, workerRequestsSetTTL)

-- Store the mapping of request ID to worker instance ID
redis.call("SET", requestWorkerKey, instanceID, "EX", requestWorkerMappingTTL)

return 0
