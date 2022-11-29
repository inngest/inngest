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

-- Fetch partition index;  if the current score is lower decrease it.
local currentScore = redis.call("ZSCORE", partitionIndexKey, partitionIndex)
if currentScore == false or tonumber(currentScore) > queueScore then
	redis.call("ZADD", partitionIndexKey, queueScore, partitionIndex)
	-- Update the partition item too
	redis.call("HSET", partitionKey, "item", partitionItem)
end

return 0
