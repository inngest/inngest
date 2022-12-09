--[[

Re-enqueus a queue item within its queue, removing any lease.

Output:
  0: Successfully re-enqueued item
  1: Queue item not found

]]

local queueKey          = KEYS[1] -- queue:item - hash: { $itemID: $item }
local queueIndexKey     = KEYS[2] -- queue:sorted:$workflowID - zset
local partitionKey      = KEYS[3] -- partition:item:$workflowID - hash { n: $leased, len: $enqueued }
local partitionIndexKey = KEYS[4] -- partition:sorted - zset

local queueItem      = ARGV[1] -- {id, lease id, attempt, max attempt, data, etc...}
local queueID        = ARGV[2] -- id
local queueScore     = tonumber(ARGV[3]) -- vesting time, in ms
local partitionIndex = ARGV[4] -- workflowID
local partitionItem  = ARGV[5] -- {workflow, priority, leasedAt, etc}

-- $include(get_queue_item.lua)
local item = get_queue_item(queueKey, queueID)
if item == nil then
	return 1
end

if item.leaseID ~= nil and item.leaseID ~= cjson.null then
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
local queueScore = redis.call("ZRANGE", queueIndexKey, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")

-- queues are ordered by ms precision, whereas pointers are second precision.
local earliestTime = math.floor(tonumber(queueScore[2]) / 1000)

-- earliest is a table containing {item, score}
if currentScore == false or tonumber(currentScore) ~= earliestTime then
	redis.call("ZADD", partitionIndexKey, earliestTime, partitionIndex)
	-- Update the partition item too
	redis.call("HSET", partitionKey, "item", partitionItem)
end

return 0
