--[[

Output:
  0: Successfully leased item
  1: Partition item not found
  2: Partition item already leased

]]

local partitionKey      = KEYS[1]
local partitionIndexKey = KEYS[2]

local partitionID = ARGV[1]
local leaseID     = ARGV[2]
local currentTime = tonumber(ARGV[3]) -- in ms, to check lease validation
local leaseTime   = tonumber(ARGV[4]) -- in seconds, as partition score

-- $include(get_partition_item.lua)
-- $include(decode_ulid_time.lua)

local existing = get_partition_item(partitionKey, partitionID)
if existing == nil then
	return 1
end
-- Check for an existing lease.
if existing.leaseID ~= nil and decode_ulid_time(existing.leaseID) > currentTime then
	return 2
end

existing.leaseID = leaseID
existing.at = leaseTime
existing.last = math.floor(currentTime / 1000)

-- Update item and index score
redis.call("HSET", partitionKey, partitionID, cjson.encode(existing))
-- scoresd are in seconds.
redis.call("ZADD", partitionIndexKey, leaseTime, partitionID)

return 0
