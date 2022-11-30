--[[

Re-enqueus a queue item within its queue, removing any lease.

Output:
  0: Successfully re-enqueued item
  1: Queue item not found

]]

local queueKey          = KEYS[1] -- queue:item - hash: { $itemID: $item }
local queueIndexKey     = KEYS[2] -- queue:sorted:$workflowID - zset
local partitionKey      = KEYS[3] -- partition:item:$workflowID - hash { item: $partition, n: $leased, len: $enqueued }
local partitionIndexKey = KEYS[4] -- partition:sorted - zset

local queueItem      = ARGV[1] -- {id, lease id, attempt, max attempt, data, etc...}
local queueID        = ARGV[2] -- id
local queueScore     = tonumber(ARGV[3]) -- vesting time, in seconds
local partitionIndex = ARGV[4] -- {workflow, priority}
local partitionItem  = ARGV[5] -- {workflow, priority, leasedAt, etc}

-- $include(fetch_queue_item.lua)
local item = get_queue_item(queueKey, queueID)
if item == nil then
	return 1
end

if item.leaseID ~= nil then
	-- Remove total number in progress if there's a lease.
	redis.call("HINCRBY", partitionKey, "n", -1)
end

redis.call("HSET", queueKey, queueID, queueItem) 
-- Update the queue score
redis.call("ZADD", queueIndexKey, queueScore, queueID)

-- Fetch partition index;  ensure this is the same as our lowest queue item score
local currentScore = redis.call("ZSCORE", partitionIndexKey, partitionIndex)

-- Peek the earliest time from the queue index.  We need to know
-- the earliest time for the entire function set, as we may be
-- rescheduling the only time in the queue;  this is the only way
-- to update the partiton index.
local earliest = redis.call("ZRANGE", queueIndexKey, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")

-- earliest is a table containing {item, score}
if currentScore == false or tonumber(currentScore) ~= tonumber(earliest[2]) then
	redis.call("ZADD", partitionIndexKey, earliest[2], partitionIndex)
	-- Update the partition item too
	redis.call("HSET", partitionKey, "item", partitionItem)
end

return 0
