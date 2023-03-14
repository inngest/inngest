--[[

Output:
 +1-N: Successfully leased item, with remaining partition capacity left
    0: No capacity left, not leased
   -1: Partition item not found
   -2: Partition item already leased

]]

local partitionKey            = KEYS[1]
local partitionIndexKey       = KEYS[2]
local partitionConcurrencyKey = KEYS[3]

local partitionID = ARGV[1]
local leaseID     = ARGV[2]
local currentTime = tonumber(ARGV[3]) -- in ms, to check lease validation
local leaseTime   = tonumber(ARGV[4]) -- in seconds, as partition score
local concurrency = tonumber(ARGV[5]) -- concurrency limit for this partition

-- $include(check_concurrency.lua)
-- $include(get_partition_item.lua)
-- $include(decode_ulid_time.lua)

local existing = get_partition_item(partitionKey, partitionID)
if existing == nil or existing == false then
	return -1
end

-- Check for an existing lease.
if existing.leaseID ~= nil and existing.leaseID ~= cjson.null and decode_ulid_time(existing.leaseID) > currentTime then
	return -2
end

local now_seconds = math.floor(currentTime / 1000)
local capacity = concurrency -- initialize as the default concurrency limit

if concurrency > 0 and #partitionConcurrencyKey > 0 then
	-- Check that there's capacity for this partition, based off of partition-level
	-- concurrency keys.
	capacity = check_concurrency(currentTime, partitionConcurrencyKey, concurrency)
	if capacity <= 0 then
		-- There's no capacity available.  Increase the score for this partition so that
		-- it's not immediately re-scanned.
		redis.call("ZADD", partitionIndexKey, leaseTime, partitionID)
		return 0
	end
end

existing.leaseID = leaseID
existing.at = leaseTime
existing.last = now_seconds

-- Update item and index score
redis.call("HSET", partitionKey, partitionID, cjson.encode(existing))
redis.call("ZADD", partitionIndexKey, leaseTime, partitionID) -- partition scored are in seconds.

return capacity
