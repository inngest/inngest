--[[

Output:
  0: Successfully added lease to set
  1: Worker capacity exceeded

ARGV[1]: TTL in seconds for the set
ARGV[2]: Request lease duration in seconds
ARGV[3]: Instance ID of the worker
ARGV[4]: Request ID
ARGV[5]: Expiration time as Unix timestamp (score for sorted set)
ARGV[6]: Current time as Unix timestamp in milliseconds
]]

local capacityKey = KEYS[1]
local workerRequestsKey = KEYS[2]
local requestWorkerKey = KEYS[3]

local setTTL = tonumber(ARGV[1])
local requestLeaseDuration = tonumber(ARGV[2])
local instanceID = ARGV[3]
local requestID = ARGV[4]
local expirationTime = ARGV[5]
local currentTime = ARGV[6]

-- Get the worker's capacity limit (returns a string)
local capacity = redis.call("GET", capacityKey)

-- If no capacity limit is set, don't track leases
-- redis nil becomes false in lua: https://redis.io/docs/latest/commands/eval/#conversion-between-lua-and-redis-data-types
if capacity == nil or capacity == 0 or capacity == false or capacity == "0" then
  return 0
end

capacity = tonumber(capacity)

-- Get current time to filter out expired leases
-- previous second, this makes us very sensitive to time changes
-- Should we use the logical clock instead?
local currentTimeUnix = tonumber(currentTime) / 1000

-- Remove expired leases from the set first
redis.call("ZREMRANGEBYSCORE", workerRequestsKey, "-inf", tostring(currentTimeUnix))

-- Get current number of active leases (those with expiration time > current time)
local currentLeases = redis.call("ZCOUNT", workerRequestsKey, tostring(currentTimeUnix + 1), "+inf")

-- Check if at capacity
if currentLeases >= capacity then
  return 1
end

-- Add the lease to the sorted set with expiration time as score
local expTime = tonumber(expirationTime)
redis.call("ZADD", workerRequestsKey, expTime, requestID)

-- Set/refresh TTL on the set to ensure it expires if worker stops processing
redis.call("EXPIRE", workerRequestsKey, setTTL)

-- Store the mapping of request ID to worker instance ID
redis.call("SET", requestWorkerKey, instanceID, "EX", requestLeaseDuration)

return 0
