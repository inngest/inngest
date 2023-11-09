--[[

Requeues a job by its given ID.  This returns an error if the job
does not exist within the queue index (outstanding queue).

NOTE: This

Return values:

- 0:  Successfully requeued
- -1: Queue item not found
- -2: Queue item is leased and being worked on.

]]--

local keyQueueIndex     = KEYS[1]
local keyQueueHash      = KEYS[2]
local keyPartitionIndex = KEYS[3] -- partition:sorted - zset
local keyPartitionHash  = KEYS[4]

local jobID       = ARGV[1] -- queue item ID
local jobScore    = tonumber(ARGV[2]) -- enqueue at, in milliseconds
local partitionID = ARGV[3] -- function ID
local currentTime = tonumber(ARGV[4]) -- in ms

if redis.call("ZSCORE", keyQueueIndex, jobID) == false then
	-- This doesn't exist.
	return -1
end

-- $include(get_queue_item.lua)
local item = get_queue_item(keyQueueHash, jobID)
if item == nil then
	return -1
end

-- Ensure that we're not requeueing a leased job.
if item.leaseID ~= nil and item.leaseID ~= cjson.null and decode_ulid_time(item.leaseID) > currentTime then
	-- This is already leased;  don't let this requester lease the item.
	return -2
end


-- $include(get_partition_item.lua)
local existing = get_partition_item(keyPartitionHash, partitionID)
if existing == nil then
	return -1
end

redis.call("ZADD", keyQueueIndex, jobScore, jobID)

-- Update the "at" time of the job
item.at = jobScore
redis.call("HSET", keyQueueHash, jobID, cjson.encode(item))


-- Get the current score of the partition;  if queueScore < currentScore update the
-- partition's score so that we can work on this workflow when the earliest member
-- is available.
-- 
-- We might have just pushed back the earliest job, so the partitions pointer
-- could have an earlier score than necessary.  In order to fix this, we want to scan
-- and take the minimum time from the keyQueueIndex and use this as the score
local minScore = redis.call("ZRANGEBYSCORE", keyQueueIndex, "-inf", "+inf", "WITHSCORES", "LIMIT", "0", "1")
local partitionScore = math.floor(minScore[2] / 1000)

local currentScore = redis.call("ZSCORE", keyPartitionIndex, partitionID)
if currentScore == false or tonumber(currentScore) ~= partitionScore then
	redis.call("ZADD", keyPartitionIndex, partitionScore, partitionID)
end

-- Update the partition pointer's actual AtS timestamp in the struct.
existing.at = partitionScore
redis.call("HSET", keyPartitionHash, partitionID, cjson.encode(existing))

return 0
