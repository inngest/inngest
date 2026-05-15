--[[

Removes a lease from the worker's sorted set and manages TTL/cleanup.

Output:
  0: Successfully removed, set deleted (empty) -- This return code is no longer used
  1: Successfully removed, set still active
  2: Set doesn't exist, nothing to remove
  3: Instance ID mismatch, operation denied

ARGV[1]: TTL in seconds for the set key
ARGV[2]: Request ID to remove
ARGV[3]: Instance ID that is requesting the deletion
]]

local workerTotalCapacityKey = KEYS[1]
local workerRequestsKey = KEYS[2]
local requestWorkerKey = KEYS[3]
local setTTL = tonumber(ARGV[1])
local requestID = ARGV[2]
local instanceID = ARGV[3]

-- Separate TTL variables for different components
local workerTotalCapacityTTL = setTTL
local workerRequestsSetTTL = setTTL

-- Check if set exists
local setExists = redis.call("EXISTS", workerRequestsKey)

-- If set doesn't exist, nothing to remove
if setExists == 0 then
  -- Still clean up the mapping in case it exists
  redis.call("DEL", requestWorkerKey)
  return 2
end

-- Check if the lease mapping exists and verify instance ID
local leaseInstanceID = redis.call("GET", requestWorkerKey)
if leaseInstanceID and leaseInstanceID ~= instanceID then
  -- Instance ID mismatch, deny the operation
  return 3
end

-- Remove the specific request ID from the set
redis.call("ZREM", workerRequestsKey, requestID)
-- Delete the specific request's worker mapping
redis.call("DEL", requestWorkerKey)
-- Refresh capacity TTL
redis.call("EXPIRE", workerTotalCapacityKey, workerTotalCapacityTTL)
-- Refresh TTL (if it's empty/has zero members, then redis will ignore it since empty sets in redis are cleaned up automatically)
redis.call("EXPIRE", workerRequestsKey, workerRequestsSetTTL)

return 1
