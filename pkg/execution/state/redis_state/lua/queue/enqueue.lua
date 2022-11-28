--[[

Enqueus an item within the queue.


--]]

local queueKey          = KEYS[1] -- queue:item:$itemID
local queueIndexKey     = KEYS[2] -- queue:sorted:$workflowID
local partitionKey      = KEYS[3] -- partition:item:$workflowID
local partitionIndexKey = KEYS[4] -- partition:sorted

local queueItem      = ARGV[1] -- {id, lease id, attempt, max attempt, data, etc...}
local queueID        = ARGV[2] -- id
local queueScore     = tonumber(ARGV[3]) -- vesting time, in seconds
local partitionIndex = ARGV[4] -- {workflow, priority}
local partitionItem  = ARGV[5] -- {workflow, priority, leasedAt, etc}

if redis.call("SETNX", queueKey, queueItem) == 0 then
	-- This already exists;  return an error.
	return 1
end

-- We score the queue items separately, as we need to continually update the score
-- when adding leases.  Doing so means we can't ZADD to update sorted sets, as each
-- time the lease ID changes the data structure changes; zsets require static members
-- when updating scores.
redis.call("ZADD", queueIndexKey, queueScore, queueID)

-- We store partitions and their leases separately from the queue-partition ZSET
-- as we want a static member excluding eg. lease IDs.  This allows us to update
-- scores idempotently.
redis.call("HSETNX", partitionKey, "item", partitionItem)
redis.call("HSETNX", partitionKey, "n", 0)   -- Atomic counter, currently leased (in progress) items.
redis.call("HINCRBY", partitionKey, "len", 1) -- Atomic counter, length of enqueued items, set to 1 or increased.

-- Get the current score of the partition;  if queueScore < currentScore update the
-- partition's score so that we can work on this workflow when the earliest member
-- is available.
local currentScore = redis.call("ZSCORE", partitionIndexKey, partitionIndex)
if currentScore == false or tonumber(currentScore) > queueScore then
	redis.call("ZADD", partitionIndexKey, queueScore, partitionIndex)
	-- Update the partition item too
	redis.call("HSET", partitionKey, "item", partitionItem)
end

-- TODO: For the given workflow ID increase scheduled count, store a history item,
-- etc:  this can be atomic in the redis queue as it combines state + queue.

return 0
